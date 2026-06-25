package middleware

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// bucket is a single token bucket keyed by (agent_id, category). It is guarded
// by the owning limiter's mutex; it carries no lock of its own.
type bucket struct {
	tokens     float64
	lastRefill time.Time
}

// limiter holds the per-(agent,category) token buckets and the per-category
// concurrency semaphores. Buckets are created lazily on first use. Token refill
// is time-based via an injectable clock so tests need not sleep.
type limiter struct {
	limits CategoryLimits
	clock  func() time.Time

	mu      sync.Mutex
	buckets map[string]*bucket
	// sems holds one buffered-channel semaphore per category, sized to that
	// category's Concurrency. A nil entry means the category is unbounded.
	sems map[string]chan struct{}
}

func newLimiter(limits CategoryLimits, clock func() time.Time) *limiter {
	if clock == nil {
		clock = time.Now
	}
	return &limiter{
		limits:  limits,
		clock:   clock,
		buckets: make(map[string]*bucket),
		sems:    make(map[string]chan struct{}),
	}
}

// bucketKey composes the (agent_id, category) bucket key. A NUL separator keeps
// it unambiguous regardless of the values.
func bucketKey(agentID, category string) string {
	return agentID + "\x00" + category
}

// allow attempts to consume one token from the (agent,category) bucket, refilling
// it first based on elapsed time. It reports whether a token was available.
func (l *limiter) allow(agentID, category string) bool {
	lim := l.limits.limitFor(category)
	now := l.clock()

	l.mu.Lock()
	defer l.mu.Unlock()

	key := bucketKey(agentID, category)
	b, ok := l.buckets[key]
	if !ok {
		// A fresh bucket starts full so the first burst is honored.
		b = &bucket{tokens: float64(lim.Burst), lastRefill: now}
		l.buckets[key] = b
	}

	elapsed := now.Sub(b.lastRefill).Seconds()
	if elapsed > 0 {
		b.tokens += elapsed * lim.Rate
		if capacity := float64(lim.Burst); b.tokens > capacity {
			b.tokens = capacity
		}
		b.lastRefill = now
	}

	if b.tokens >= 1 {
		b.tokens--
		return true
	}
	return false
}

// acquire takes one concurrency slot for category without blocking. It returns a
// release func and true on success, or (nil, false) when the category is at its
// concurrency cap. An unbounded category always succeeds with a no-op release.
func (l *limiter) acquire(category string) (release func(), ok bool) {
	conc := l.limits.limitFor(category).Concurrency
	if conc <= 0 {
		return func() {}, true
	}

	l.mu.Lock()
	sem, exists := l.sems[category]
	if !exists {
		sem = make(chan struct{}, conc)
		l.sems[category] = sem
	}
	l.mu.Unlock()

	select {
	case sem <- struct{}{}:
		var once sync.Once
		return func() { once.Do(func() { <-sem }) }, true
	default:
		return nil, false
	}
}

// RateLimit is the rate-limiting layer (Order 30). It applies a token bucket per
// (agent_id, category) and a per-category concurrency semaphore. On bucket
// exhaustion or concurrency saturation it SHORT-CIRCUITS with a tool-level error
// result (not a Go error, which would become a protocol error) and marks the
// audit outcome as rate-limited. The semaphore slot is always released via defer.
type RateLimit struct {
	limiter *limiter
}

// Order returns 30.
func (RateLimit) Order() int { return orderRateLimit }

// Handle enforces the bucket then the concurrency semaphore before continuing.
func (rl RateLimit) Handle(ctx context.Context, cc *spi.ToolCallContext, req *spi.ToolRequest, next spi.ToolHandler) (*spi.ToolResult, error) {
	cat := category(cc)
	agent := agentID(cc)

	if !rl.limiter.allow(agent, cat) {
		markDecision(ctx, DecisionRateLimited)
		return &spi.ToolResult{
			IsError: true,
			Text:    fmt.Sprintf("rate limit exceeded for category %q (agent %q)", cat, agent),
		}, nil
	}

	release, ok := rl.limiter.acquire(cat)
	if !ok {
		markDecision(ctx, DecisionRateLimited)
		return &spi.ToolResult{
			IsError: true,
			Text:    fmt.Sprintf("concurrency limit reached for category %q", cat),
		}, nil
	}
	defer release()

	return next(ctx, cc, req)
}
