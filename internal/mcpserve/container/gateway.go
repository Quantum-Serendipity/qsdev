package container

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// orderGatewayAuth places the gateway AUTHORIZATION layer INSIDE the Audit layer
// (Order 15) and OUTSIDE Guardrail (Order 20). Authentication itself now happens
// at the TLS layer (the mTLS client-certificate CN/SAN becomes the authoritative
// cc.AgentID; see internal/mcpserve identity/transport), so this layer no longer
// authenticates — it authorizes the already-verified identity against the
// allow-list. Sitting inside Audit (16 > 15) is deliberate: a denial here is
// recorded by Audit (it marks the outcome DecisionDenied before short-circuiting)
// rather than going unaudited as it would at the former outermost Order 5. It
// stays outside Guardrail (16 < 20) so an un-allow-listed identity is rejected
// before policy/rate-limit/content-safety run. Lower Order() is more outer (see
// spi.Chain.Execute).
const orderGatewayAuth = 16

// unknownAgent mirrors the sentinel the universal server assigns when neither a
// verified transport identity, the _meta override, nor the handshake clientInfo
// yields an identity. The gateway treats it as "no usable identity" and never
// admits it when auth is required, regardless of allow-list contents.
const unknownAgent = "unknown"

// GatewayInterceptor is the container-specific AUTHORIZATION layer of the
// Gateway enforcement chain. Authentication is performed at the TLS layer (mTLS
// client certificates), which sets the cryptographically-verified caller
// identity as cc.AgentID; this layer authorizes that verified identity against a
// configured allow-list of trusted cert identities (CNs). It is the only NEW
// middleware the gateway adds; rate limiting and content safety are reused
// verbatim from the built-in middleware package (see GatewayChain). It
// short-circuits with a tool-level error result (IsError, never a Go error —
// that would surface as a JSON-RPC protocol error) when the verified identity is
// not in the allow-list, and records the denial on the audit outcome so the
// Audit layer (which wraps it) reports it as denied.
type GatewayInterceptor struct {
	// allowed is the set of trusted (verified) identities permitted to call
	// through the gateway.
	allowed map[string]struct{}
	// requireAuth gates enforcement. When false the interceptor is a
	// transparent pass-through (used when the gateway runs without an
	// allow-list); when true an identity absent from allowed is denied, and an
	// EMPTY allow-list therefore denies every caller (fail-closed).
	requireAuth bool
}

// NewGatewayInterceptor builds an authorization interceptor from an allow-list
// of trusted cert identities (CNs). Blank entries are ignored. When requireAuth
// is true and the resulting allow-list is empty, every request is denied
// (fail-closed) — the caller is responsible for supplying identities when it
// turns enforcement on.
func NewGatewayInterceptor(allowedAgents []string, requireAuth bool) *GatewayInterceptor {
	allowed := make(map[string]struct{}, len(allowedAgents))
	for _, a := range allowedAgents {
		if a = strings.TrimSpace(a); a != "" {
			allowed[a] = struct{}{}
		}
	}
	return &GatewayInterceptor{allowed: allowed, requireAuth: requireAuth}
}

// Order returns orderGatewayAuth (16): inside Audit, outside Guardrail.
func (*GatewayInterceptor) Order() int { return orderGatewayAuth }

// Handle authorizes the (TLS-authenticated) caller, then continues the chain. A
// rejected caller short-circuits: next is never invoked, so none of the inner
// enforcement layers or the tool handler run. Before returning the denial it
// records DecisionDenied on the audit outcome so the Audit layer (Order 15,
// which wraps this one) reports the call as denied rather than a generic error.
func (g *GatewayInterceptor) Handle(ctx context.Context, cc *spi.ToolCallContext, req *spi.ToolRequest, next spi.ToolHandler) (*spi.ToolResult, error) {
	if g.requireAuth && !g.authorized(cc) {
		middleware.MarkDecision(ctx, middleware.DecisionDenied)
		return &spi.ToolResult{
			IsError: true,
			Text:    fmt.Sprintf("gateway authorization failed: identity %q is not in the allowed-agents set", agentIDOf(cc)),
		}, nil
	}
	return next(ctx, cc, req)
}

// authorized reports whether cc carries a usable, allow-listed (verified)
// identity.
func (g *GatewayInterceptor) authorized(cc *spi.ToolCallContext) bool {
	id := agentIDOf(cc)
	if id == "" || id == unknownAgent {
		return false
	}
	_, ok := g.allowed[id]
	return ok
}

// agentIDOf reads the resolved agent id from cc, tolerating a nil context.
func agentIDOf(cc *spi.ToolCallContext) string {
	if cc == nil {
		return ""
	}
	return cc.AgentID
}

// GatewayOptions configures GatewayChain. The zero value yields a usable gateway
// (stricter default limits, permissive-by-default authorization policy, no
// allow-list enforcement). Set AllowedAgents + RequireAuth to turn allow-list
// authorization on (authentication itself is the mTLS layer, not this chain).
type GatewayOptions struct {
	// AllowedAgents is the allow-list of trusted (verified) identities (see
	// NewGatewayInterceptor).
	AllowedAgents []string
	// RequireAuth turns the gateway authorization layer from pass-through into
	// enforcing. With an empty AllowedAgents this denies all callers.
	RequireAuth bool
	// Policy overrides the Guardrail authorization policy. Nil keeps the
	// built-in permissive-by-default policy (deny rules subtract from it).
	Policy *middleware.Policy
	// Limits overrides the per-category rate-limit table. The zero value
	// installs StricterGatewayLimits.
	Limits middleware.CategoryLimits
	// Clock injects the rate-limiter / audit clock (tests). Nil uses time.Now.
	Clock func() time.Time
	// AuditSink overrides the audit sink. Nil keeps the default slog sink.
	AuditSink middleware.AuditSink
	// Redactor overrides the ContentSafety redactor. Nil uses a default.
	Redactor *logging.Redactor
}

// GatewayChain assembles the Gateway enforcement chain and returns it as a plain
// *spi.Chain the serve command installs in gateway mode. Authentication is NOT a
// layer here — it is performed at the TLS layer (mTLS client certificates),
// which sets the verified identity as cc.AgentID before any middleware runs.
//
// Layers, from outermost to innermost:
//
//	10  ContextInjection  (REUSED)
//	15  Audit             (REUSED) — records every decision, incl. gateway denials
//	16  GatewayAuthz      — GatewayInterceptor (NEW): allow-list authorization
//	20  Guardrail         (REUSED) — policy authorization
//	30  RateLimit         (REUSED, stricter limits)
//	45  ContentSafety     (REUSED)
//	50  ErrorHandling     (REUSED)
//
// It builds the full built-in chain via middleware.DefaultChain (which brings
// ContextInjection, Audit, and ErrorHandling — the gateway keeps those: dropping
// them would lose request-context propagation, audit, and panic recovery) and
// then wraps it with the authorization interceptor using spi.Chain.With. Because
// Execute orders by Order(), the interceptor's Order 16 lands between Audit (15)
// and Guardrail (20) without any manual re-ordering — so a denial it issues is
// still recorded by Audit.
func GatewayChain(opts GatewayOptions) *spi.Chain {
	limits := opts.Limits
	if limits.Limits == nil && limits.Default == (middleware.Limit{}) {
		limits = StricterGatewayLimits()
	}

	mwOpts := []middleware.Option{middleware.WithLimits(limits)}
	if opts.Policy != nil {
		mwOpts = append(mwOpts, middleware.WithPolicy(opts.Policy))
	}
	if opts.Clock != nil {
		mwOpts = append(mwOpts, middleware.WithClock(opts.Clock))
	}
	if opts.AuditSink != nil {
		mwOpts = append(mwOpts, middleware.WithAuditSink(opts.AuditSink))
	}
	if opts.Redactor != nil {
		mwOpts = append(mwOpts, middleware.WithRedactor(opts.Redactor))
	}

	return middleware.DefaultChain(mwOpts...).With(NewGatewayInterceptor(opts.AllowedAgents, opts.RequireAuth))
}

// StricterGatewayLimits returns the built-in per-category limits with every
// rate, burst, and concurrency roughly halved. A gateway fronts UNTRUSTED
// frameworks that lack their own hook-level throttling, so it runs tighter
// buckets than a locally-trusted native deployment. Burst and (bounded)
// concurrency never drop below 1, so no category is accidentally throttled to a
// permanent standstill.
func StricterGatewayLimits() middleware.CategoryLimits {
	base := middleware.DefaultLimits()
	out := middleware.CategoryLimits{
		Default: tightenLimit(base.Default),
		Limits:  make(map[string]middleware.Limit, len(base.Limits)),
	}
	for cat, lim := range base.Limits {
		out.Limits[cat] = tightenLimit(lim)
	}
	return out
}

// tightenLimit halves a single Limit, flooring burst at 1 and bounded
// concurrency at 1 so a category is never wedged shut.
func tightenLimit(l middleware.Limit) middleware.Limit {
	burst := l.Burst / 2
	if burst < 1 {
		burst = 1
	}
	conc := l.Concurrency
	if conc > 0 {
		conc /= 2
		if conc < 1 {
			conc = 1
		}
	}
	return middleware.Limit{Rate: l.Rate / 2, Burst: burst, Concurrency: conc}
}
