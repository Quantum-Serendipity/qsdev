package middleware

import (
	"context"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// TestAuditRecordsDurationAndOutcome exercises the Audit layer in isolation: it
// captures the inner-chain duration (via the injected clock advanced by the
// handler) and infers the decision when no inner layer marked one.
func TestAuditRecordsDurationAndOutcome(t *testing.T) {
	t.Parallel()

	clk := newFakeClock()
	sink := &recordingSink{}
	a := Audit{sink: sink, clock: clk.Now}
	cc := &spi.ToolCallContext{AgentID: "bot", ToolName: "grep", Category: CategorySearch}

	handler := func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
		clk.Advance(5 * time.Millisecond)
		return &spi.ToolResult{Text: "hit"}, nil
	}
	if _, err := a.Handle(context.Background(), cc, &spi.ToolRequest{Name: "grep"}, handler); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	recs := sink.all()
	if len(recs) != 1 {
		t.Fatalf("got %d records, want 1", len(recs))
	}
	r := recs[0]
	if r.AgentID != "bot" || r.Tool != "grep" || r.Category != CategorySearch {
		t.Errorf("record metadata = %+v", r)
	}
	if r.Decision != DecisionAllowed {
		t.Errorf("decision = %q, want allowed", r.Decision)
	}
	if r.Duration != 5*time.Millisecond {
		t.Errorf("duration = %v, want 5ms", r.Duration)
	}
	if r.IsError {
		t.Error("IsError = true, want false for a clean result")
	}
}

// TestAuditInfersErrorOutcome checks a tool-error result with no marked decision
// is recorded as error.
func TestAuditInfersErrorOutcome(t *testing.T) {
	t.Parallel()

	sink := &recordingSink{}
	a := Audit{sink: sink, clock: newFakeClock().Now}
	handler := func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
		return &spi.ToolResult{IsError: true, Text: "nope"}, nil
	}
	if _, err := a.Handle(context.Background(), &spi.ToolCallContext{ToolName: "x"}, &spi.ToolRequest{}, handler); err != nil {
		t.Fatal(err)
	}
	recs := sink.all()
	if len(recs) != 1 || recs[0].Decision != DecisionError || !recs[0].IsError {
		t.Errorf("records = %+v, want one error record", recs)
	}
}

// TestAuditOutsideGuardrail is the C9 proof: with Audit (15) OUTSIDE Guardrail
// (20), a call DENIED by Guardrail is still audited, recorded with the precise
// "denied" decision the Guardrail layer marked — even though the handler never
// ran. The allowed call records "allowed".
func TestAuditOutsideGuardrail(t *testing.T) {
	t.Parallel()

	sink := &recordingSink{}
	clk := newFakeClock()
	policy := &Policy{
		ByToolType: map[string]Verdict{CategoryCredential: VerdictDeny},
		Default:    VerdictAllow,
	}
	chain := DefaultChain(
		WithAuditSink(sink),
		WithPolicy(policy),
		WithClock(clk.Now),
	)

	var allowedRan, deniedRan bool

	// Allowed call.
	allowedCC := &spi.ToolCallContext{AgentID: "bot", ToolName: "grep", Category: CategorySearch}
	if _, err := chain.Execute(context.Background(), allowedCC, &spi.ToolRequest{Name: "grep"}, okHandler(&allowedRan, "hit")); err != nil {
		t.Fatalf("allowed Execute error: %v", err)
	}

	// Denied call.
	deniedCC := &spi.ToolCallContext{AgentID: "bot", ToolName: "credential_vend", Category: CategoryCredential}
	res, err := chain.Execute(context.Background(), deniedCC, &spi.ToolRequest{Name: "credential_vend"}, okHandler(&deniedRan, "secret"))
	if err != nil {
		t.Fatalf("denied Execute error: %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("denied result = %+v, want IsError", res)
	}

	if !allowedRan {
		t.Error("allowed handler did not run")
	}
	if deniedRan {
		t.Error("denied handler ran despite guardrail deny")
	}

	recs := sink.all()
	if len(recs) != 2 {
		t.Fatalf("got %d audit records, want 2 (allowed + denied)", len(recs))
	}
	got := map[string]Decision{}
	for _, r := range recs {
		got[r.Tool] = r.Decision
	}
	if got["grep"] != DecisionAllowed {
		t.Errorf("grep decision = %q, want allowed", got["grep"])
	}
	if got["credential_vend"] != DecisionDenied {
		t.Errorf("credential_vend decision = %q, want denied (C9: audited despite guardrail short-circuit)", got["credential_vend"])
	}
}

// TestAuditRecordsRateLimited proves a throttled call is audited as
// rate_limited (RateLimit sits inside Audit).
func TestAuditRecordsRateLimited(t *testing.T) {
	t.Parallel()

	sink := &recordingSink{}
	clk := newFakeClock()
	chain := DefaultChain(
		WithAuditSink(sink),
		WithClock(clk.Now),
		WithLimits(limitsWith(CategorySearch, Limit{Rate: 0, Burst: 1, Concurrency: 0})),
	)
	cc := &spi.ToolCallContext{AgentID: "bot", ToolName: "grep", Category: CategorySearch}

	// First call consumes the single token; second is throttled.
	if _, err := chain.Execute(context.Background(), cc, &spi.ToolRequest{Name: "grep"}, okHandler(nil, "ok")); err != nil {
		t.Fatal(err)
	}
	if _, err := chain.Execute(context.Background(), cc, &spi.ToolRequest{Name: "grep"}, okHandler(nil, "ok")); err != nil {
		t.Fatal(err)
	}

	recs := sink.all()
	if len(recs) != 2 {
		t.Fatalf("got %d records, want 2", len(recs))
	}
	if recs[0].Decision != DecisionAllowed {
		t.Errorf("first decision = %q, want allowed", recs[0].Decision)
	}
	if recs[1].Decision != DecisionRateLimited {
		t.Errorf("second decision = %q, want rate_limited", recs[1].Decision)
	}
}

// TestNewSlogSinkNilLoggerFallsBack confirms the default sink tolerates a nil
// logger (uses slog.Default) without panicking.
func TestNewSlogSinkNilLoggerFallsBack(t *testing.T) {
	t.Parallel()
	sink := NewSlogSink(nil)
	// Must not panic when emitting to the fallback slog.Default() logger.
	sink.Record(context.Background(), AuditRecord{AgentID: "a", Tool: "t", Decision: DecisionAllowed})
}
