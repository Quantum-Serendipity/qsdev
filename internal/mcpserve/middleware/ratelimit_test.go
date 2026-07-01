package middleware

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// limitsWith builds a single-category limit table for tests.
func limitsWith(category string, l Limit) CategoryLimits {
	return CategoryLimits{
		Default: Limit{Rate: 1, Burst: 1, Concurrency: 0},
		Limits:  map[string]Limit{category: l},
	}
}

func ccFor(agent, category, tool string) *spi.ToolCallContext {
	return &spi.ToolCallContext{AgentID: agent, Category: category, ToolName: tool}
}

// TestRateLimitBucketExhaustionAndRefill proves: the first Burst calls are
// allowed, the next is throttled, and after enough simulated time passes a token
// refills and the call is allowed again — all without sleeping.
func TestRateLimitBucketExhaustionAndRefill(t *testing.T) {
	t.Parallel()

	clk := newFakeClock()
	// Burst 3, refill 1 token/sec, unbounded concurrency.
	rl := RateLimit{limiter: newLimiter(limitsWith(CategorySearch, Limit{Rate: 1, Burst: 3, Concurrency: 0}), clk.Now)}
	cc := ccFor("bot", CategorySearch, "grep")

	call := func() (*spi.ToolResult, bool) {
		var ran bool
		res, err := rl.Handle(context.Background(), cc, &spi.ToolRequest{Name: "grep"}, okHandler(&ran, "ok"))
		if err != nil {
			t.Fatalf("unexpected Go error: %v", err)
		}
		return res, ran
	}

	// First 3 (burst) succeed.
	for i := 0; i < 3; i++ {
		res, ran := call()
		if !ran || res.IsError {
			t.Fatalf("call %d: expected allowed, got ran=%v res=%+v", i, ran, res)
		}
	}

	// 4th is throttled: handler not run, IsError result.
	res, ran := call()
	if ran {
		t.Error("handler ran while bucket exhausted")
	}
	if res == nil || !res.IsError {
		t.Fatalf("exhausted result = %+v, want IsError", res)
	}

	// Still throttled before any time passes.
	if _, ran := call(); ran {
		t.Error("handler ran on second exhausted call")
	}

	// Advance 1s -> exactly one token refills -> one more call allowed, then
	// throttled again.
	clk.Advance(1_000_000_000) // 1s in nanoseconds
	if _, ran := call(); !ran {
		t.Error("handler did not run after refill")
	}
	if _, ran := call(); ran {
		t.Error("handler ran again without further refill")
	}
}

// TestRateLimitPerAgentCategoryIsolation proves buckets are keyed by
// (agent_id, category): exhausting one agent's bucket does not throttle another.
func TestRateLimitPerAgentCategoryIsolation(t *testing.T) {
	t.Parallel()

	clk := newFakeClock()
	rl := RateLimit{limiter: newLimiter(limitsWith(CategorySearch, Limit{Rate: 0, Burst: 1, Concurrency: 0}), clk.Now)}

	exhaust := func(agent string) bool {
		var ran bool
		_, err := rl.Handle(context.Background(), ccFor(agent, CategorySearch, "grep"), &spi.ToolRequest{Name: "grep"}, okHandler(&ran, "ok"))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		return ran
	}

	if !exhaust("bot-a") {
		t.Fatal("bot-a first call should be allowed")
	}
	if exhaust("bot-a") {
		t.Error("bot-a second call should be throttled (burst 1, rate 0)")
	}
	if !exhaust("bot-b") {
		t.Error("bot-b should have its own full bucket")
	}
}

// TestRateLimitConcurrencySemaphore proves the per-category concurrency cap
// short-circuits an over-cap call and that slots release on return.
func TestRateLimitConcurrencySemaphore(t *testing.T) {
	t.Parallel()

	clk := newFakeClock()
	// Burst large so the token bucket never interferes; concurrency capped at 2.
	rl := RateLimit{limiter: newLimiter(limitsWith(CategoryProcess, Limit{Rate: 1000, Burst: 1000, Concurrency: 2}), clk.Now)}
	cc := ccFor("bot", CategoryProcess, "nix_run")

	// Block two handlers inside the chain to occupy both slots.
	release := make(chan struct{})
	var inFlight sync.WaitGroup
	blocking := func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
		inFlight.Done()
		<-release
		return &spi.ToolResult{Text: "done"}, nil
	}

	inFlight.Add(2)
	var wg sync.WaitGroup
	wg.Add(2)
	for i := 0; i < 2; i++ {
		go func() {
			defer wg.Done()
			if _, err := rl.Handle(context.Background(), cc, &spi.ToolRequest{Name: "nix_run"}, blocking); err != nil {
				t.Errorf("blocking call error: %v", err)
			}
		}()
	}
	inFlight.Wait() // both slots now occupied

	// A third concurrent call must be rejected (slots full).
	var ran bool
	res, err := rl.Handle(context.Background(), cc, &spi.ToolRequest{Name: "nix_run"}, okHandler(&ran, "x"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ran {
		t.Error("third call ran despite concurrency cap of 2")
	}
	if res == nil || !res.IsError {
		t.Fatalf("over-cap result = %+v, want IsError", res)
	}

	// Release the two in-flight calls; slots free up.
	close(release)
	wg.Wait()

	// Now a fresh call acquires a freed slot and runs.
	var ranAfter bool
	if _, err := rl.Handle(context.Background(), cc, &spi.ToolRequest{Name: "nix_run"}, okHandler(&ranAfter, "x")); err != nil {
		t.Fatalf("post-release error: %v", err)
	}
	if !ranAfter {
		t.Error("call after slot release did not run; semaphore not released")
	}
}

// TestRateLimitConcurrencyRejectDoesNotBurnToken proves R14: when the
// concurrency acquire fails, no rate-limit token is consumed, so a later call
// that lands on a freed slot still has its token. The buggy ordering consumed a
// token BEFORE the concurrency check and never refunded it on rejection.
func TestRateLimitConcurrencyRejectDoesNotBurnToken(t *testing.T) {
	t.Parallel()

	clk := newFakeClock()
	// Burst 2 tokens, no refill (Rate 0) so the count is conserved across calls,
	// concurrency capped at 1 so a single in-flight call saturates the category.
	rl := RateLimit{limiter: newLimiter(limitsWith(CategoryProcess, Limit{Rate: 0, Burst: 2, Concurrency: 1}), clk.Now)}
	cc := ccFor("bot", CategoryProcess, "nix_run")

	// Occupy the lone concurrency slot with a blocking handler. This consumes one
	// token (2 -> 1) and holds the slot until released.
	release := make(chan struct{})
	inFlight := make(chan struct{})
	blocking := func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
		close(inFlight)
		<-release
		return &spi.ToolResult{Text: "done"}, nil
	}
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		if _, err := rl.Handle(context.Background(), cc, &spi.ToolRequest{Name: "nix_run"}, blocking); err != nil {
			t.Errorf("blocking call error: %v", err)
		}
	}()
	<-inFlight // slot occupied, one token remaining

	// A call while the slot is saturated must be rejected for CONCURRENCY and must
	// NOT consume the remaining token.
	var ranRejected bool
	res, err := rl.Handle(context.Background(), cc, &spi.ToolRequest{Name: "nix_run"}, okHandler(&ranRejected, "x"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ranRejected {
		t.Error("rejected call ran despite concurrency cap of 1")
	}
	if res == nil || !res.IsError {
		t.Fatalf("saturated result = %+v, want IsError", res)
	}

	// Free the slot.
	close(release)
	wg.Wait()

	// The freed-up call must still find a token in the bucket. Under the bug, the
	// rejected call burned it and this call would be throttled.
	var ranAfter bool
	if _, err := rl.Handle(context.Background(), cc, &spi.ToolRequest{Name: "nix_run"}, okHandler(&ranAfter, "x")); err != nil {
		t.Fatalf("post-release error: %v", err)
	}
	if !ranAfter {
		t.Error("freed-up call did not run; a token was wrongly burned by the rejected concurrency call (R14)")
	}

	// The bucket is now genuinely empty (Rate 0, no refill): the next call is
	// throttled. This confirms the earlier success spent the *last* token, i.e.
	// the rejected call left the count untouched at exactly one.
	var ranEmpty bool
	res, err = rl.Handle(context.Background(), cc, &spi.ToolRequest{Name: "nix_run"}, okHandler(&ranEmpty, "x"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ranEmpty {
		t.Error("bucket should be empty after both tokens are spent")
	}
	if res == nil || !res.IsError {
		t.Fatalf("empty-bucket result = %+v, want IsError", res)
	}
}

// TestRateLimitBucketEviction proves R15: a bucket left idle past the limiter's
// TTL is evicted on a later allow() sweep, bounding the map's memory. The
// injected clock is driven forward deterministically (no sleeping).
func TestRateLimitBucketEviction(t *testing.T) {
	t.Parallel()

	clk := newFakeClock()
	rl := RateLimit{limiter: newLimiter(limitsWith(CategorySearch, Limit{Rate: 1, Burst: 5, Concurrency: 0}), clk.Now)}

	// Create a bucket for an agent, then let it go idle.
	idleCC := ccFor("idle-bot", CategorySearch, "grep")
	if _, err := rl.Handle(context.Background(), idleCC, &spi.ToolRequest{Name: "grep"}, okHandler(nil, "x")); err != nil {
		t.Fatal(err)
	}
	idleKey := bucketKey("idle-bot", CategorySearch)
	if _, ok := rl.limiter.buckets[idleKey]; !ok {
		t.Fatalf("idle bucket was not created")
	}

	// Advance past the TTL, then drive a sweep via a call for a DIFFERENT agent.
	clk.Advance(rl.limiter.bucketTTL + time.Second)
	activeCC := ccFor("active-bot", CategorySearch, "grep")
	if _, err := rl.Handle(context.Background(), activeCC, &spi.ToolRequest{Name: "grep"}, okHandler(nil, "x")); err != nil {
		t.Fatal(err)
	}

	// The idle bucket must have been evicted; the active one must remain.
	if _, ok := rl.limiter.buckets[idleKey]; ok {
		t.Error("idle bucket past TTL was not evicted (R15)")
	}
	if _, ok := rl.limiter.buckets[bucketKey("active-bot", CategorySearch)]; !ok {
		t.Error("active bucket should remain after eviction sweep")
	}
}

// TestRateLimitUnknownCategoryUsesDefault confirms an unmapped category falls
// back to the Default limit.
func TestRateLimitUnknownCategoryUsesDefault(t *testing.T) {
	t.Parallel()

	clk := newFakeClock()
	limits := CategoryLimits{Default: Limit{Rate: 0, Burst: 1, Concurrency: 0}}
	rl := RateLimit{limiter: newLimiter(limits, clk.Now)}
	cc := ccFor("bot", "totally-unknown-category", "x")

	var ran1 bool
	if _, err := rl.Handle(context.Background(), cc, &spi.ToolRequest{}, okHandler(&ran1, "a")); err != nil {
		t.Fatal(err)
	}
	if !ran1 {
		t.Fatal("first call should be allowed under default burst 1")
	}
	var ran2 bool
	if _, err := rl.Handle(context.Background(), cc, &spi.ToolRequest{}, okHandler(&ran2, "b")); err != nil {
		t.Fatal(err)
	}
	if ran2 {
		t.Error("second call should be throttled under default burst 1 / rate 0")
	}
}
