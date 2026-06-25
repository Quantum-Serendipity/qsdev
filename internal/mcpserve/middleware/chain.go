package middleware

import (
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// config holds the resolved construction parameters for the default chain.
type config struct {
	policy   *Policy
	limits   CategoryLimits
	sink     AuditSink
	clock    func() time.Time
	redactor *logging.Redactor
}

// Option configures the default middleware chain.
type Option func(*config)

// WithPolicy overrides the Guardrail permission policy. A nil policy is ignored
// (the safe permissive-by-default policy is kept).
func WithPolicy(p *Policy) Option {
	return func(c *config) {
		if p != nil {
			c.policy = p
		}
	}
}

// WithLimits overrides the per-category rate-limit table. A fully zero-value
// CategoryLimits is ignored (it would otherwise install an all-zero Default
// bucket that rate-limits everything); a default-only table (nil per-category
// map but a non-zero Default) is accepted.
func WithLimits(l CategoryLimits) Option {
	return func(c *config) {
		if l.Limits == nil && l.Default == (Limit{}) {
			return
		}
		c.limits = l
	}
}

// WithAuditSink overrides the audit sink. A nil sink is ignored (the default
// slog sink is kept).
func WithAuditSink(s AuditSink) Option {
	return func(c *config) {
		if s != nil {
			c.sink = s
		}
	}
}

// WithClock injects the clock used by the rate-limiter (token refill) and the
// audit timer. Intended for tests; a nil clock is ignored (time.Now is used).
func WithClock(clock func() time.Time) Option {
	return func(c *config) {
		if clock != nil {
			c.clock = clock
		}
	}
}

// WithRedactor overrides the ContentSafety redactor. A nil redactor is ignored
// (a default logging.NewRedactor is used).
func WithRedactor(r *logging.Redactor) Option {
	return func(c *config) {
		if r != nil {
			c.redactor = r
		}
	}
}

// DefaultChain assembles the six built-in middleware layers into a *spi.Chain in
// onion order (10 ContextInjection, 15 Audit, 20 Guardrail, 30 RateLimit, 45
// ContentSafety, 50 ErrorHandling). Defaults: permissive-by-default policy,
// DefaultLimits, a slog audit sink on slog.Default(), the wall clock, and a
// default secret redactor. Options refine each. The returned chain is exactly
// what mcpserve.WithChain expects.
func DefaultChain(opts ...Option) *spi.Chain {
	cfg := config{
		policy:   DefaultPolicy(),
		limits:   DefaultLimits(),
		sink:     NewSlogSink(nil),
		clock:    time.Now,
		redactor: logging.NewRedactor(),
	}
	for _, o := range opts {
		o(&cfg)
	}

	return spi.NewChain(
		ContextInjection{},
		Audit{sink: cfg.sink, clock: cfg.clock},
		Guardrail{policy: cfg.policy},
		RateLimit{limiter: newLimiter(cfg.limits, cfg.clock)},
		ContentSafety{redactor: cfg.redactor},
		ErrorHandling{},
	)
}
