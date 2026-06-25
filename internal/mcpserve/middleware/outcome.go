package middleware

import "context"

// Decision classifies how a tool call resolved, for audit purposes.
type Decision string

const (
	// DecisionAllowed means the call reached the handler and returned a
	// non-error result.
	DecisionAllowed Decision = "allowed"
	// DecisionError means the call reached the handler (or a post-processing
	// layer) and produced a tool-level error result.
	DecisionError Decision = "error"
	// DecisionDenied means the Guardrail layer rejected the call by policy.
	DecisionDenied Decision = "denied"
	// DecisionRateLimited means the RateLimit layer rejected the call (token
	// bucket exhausted or concurrency cap reached).
	DecisionRateLimited Decision = "rate_limited"
	// DecisionPanic means the handler panicked and ErrorHandling recovered it.
	DecisionPanic Decision = "panic"
)

// outcome is a mutable carrier placed onto the context by the Audit layer
// before it calls inward. Inner layers that short-circuit (Guardrail, RateLimit)
// or recover a panic (ErrorHandling) record their decision here so that Audit —
// which sits OUTSIDE Guardrail and RateLimit (correction C9) — can report the
// precise reason a call never reached the handler, not merely "IsError".
//
// A context-scoped pointer is used (rather than a return value) because the
// short-circuiting layer is nested several frames below Audit; the pointer lets
// the inner frame write a value the outer frame reads after next returns.
type outcome struct {
	decision Decision
	set      bool
}

type outcomeKey struct{}

// withOutcome returns a context carrying a fresh *outcome plus that pointer.
func withOutcome(ctx context.Context) (context.Context, *outcome) {
	o := &outcome{}
	return context.WithValue(ctx, outcomeKey{}, o), o
}

// markDecision records d on the outcome carrier in ctx, if one is present. The
// first decision wins so an outer short-circuit is not overwritten by a stale
// inner value. It is a no-op when no carrier is installed (e.g. a chain built
// without the Audit layer), keeping the inner layers usable in isolation.
func markDecision(ctx context.Context, d Decision) {
	if o, ok := ctx.Value(outcomeKey{}).(*outcome); ok && !o.set {
		o.decision = d
		o.set = true
	}
}
