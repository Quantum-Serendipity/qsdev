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

// orderGatewayAuth places the gateway authentication layer OUTSIDE every
// built-in middleware. It is below Guardrail's Order 20 (it must see the
// request before authorization, rate limiting, or content safety) and below
// ContextInjection's Order 10 so an unauthenticated request is rejected before
// any inner layer runs. Lower Order() is more outer (see spi.Chain.Execute).
const orderGatewayAuth = 5

// unknownAgent mirrors the sentinel the universal server assigns when neither
// the _meta override nor the handshake clientInfo yields an identity. The
// gateway treats it as "no usable identity" and never admits it when auth is
// required, regardless of allow-list contents.
const unknownAgent = "unknown"

// GatewayInterceptor is the container-specific AUTHENTICATION layer (layer 1 of
// the 4-layer Gateway enforcement). It is the only NEW middleware the gateway
// adds; authorization, rate limiting, and content safety are reused verbatim
// from the built-in middleware package (see GatewayChain). It validates the
// request's resolved agent identity (cc.AgentID) against a configured
// allow-list and short-circuits with a tool-level error result (IsError, never
// a Go error — that would surface as a JSON-RPC protocol error) when the agent
// is not permitted.
type GatewayInterceptor struct {
	// allowed is the set of agent ids permitted to call through the gateway.
	allowed map[string]struct{}
	// requireAuth gates enforcement. When false the interceptor is a
	// transparent pass-through (used when the gateway runs without an
	// allow-list); when true an agent absent from allowed is denied, and an
	// EMPTY allow-list therefore denies every agent (fail-closed).
	requireAuth bool
}

// NewGatewayInterceptor builds an authentication interceptor from an allow-list
// of agent ids. Blank entries are ignored. When requireAuth is true and the
// resulting allow-list is empty, every request is denied (fail-closed) — the
// caller is responsible for supplying agents when it turns enforcement on.
func NewGatewayInterceptor(allowedAgents []string, requireAuth bool) *GatewayInterceptor {
	allowed := make(map[string]struct{}, len(allowedAgents))
	for _, a := range allowedAgents {
		if a = strings.TrimSpace(a); a != "" {
			allowed[a] = struct{}{}
		}
	}
	return &GatewayInterceptor{allowed: allowed, requireAuth: requireAuth}
}

// Order returns 5, the outermost position in a gateway chain.
func (*GatewayInterceptor) Order() int { return orderGatewayAuth }

// Handle authenticates the caller, then continues the chain. A rejected caller
// short-circuits: next is never invoked, so none of the inner enforcement
// layers or the tool handler run for an unauthenticated request.
func (g *GatewayInterceptor) Handle(ctx context.Context, cc *spi.ToolCallContext, req *spi.ToolRequest, next spi.ToolHandler) (*spi.ToolResult, error) {
	if g.requireAuth && !g.authenticated(cc) {
		return &spi.ToolResult{
			IsError: true,
			Text:    fmt.Sprintf("gateway authentication failed: agent %q is not in the allowed-agents set", agentIDOf(cc)),
		}, nil
	}
	return next(ctx, cc, req)
}

// authenticated reports whether cc carries a usable, allow-listed identity.
func (g *GatewayInterceptor) authenticated(cc *spi.ToolCallContext) bool {
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
// (stricter default limits, permissive-by-default authorization policy, no auth
// enforcement). Set AllowedAgents + RequireAuth to turn authentication on.
type GatewayOptions struct {
	// AllowedAgents is the authentication allow-list (see NewGatewayInterceptor).
	AllowedAgents []string
	// RequireAuth turns the authentication layer from pass-through into
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

// GatewayChain assembles the 4-layer Gateway enforcement chain and returns it as
// a plain *spi.Chain the serve command installs in gateway mode.
//
// Layers, from outermost to innermost:
//
//  1. Authentication  — GatewayInterceptor (Order 5, NEW).
//  2. Authorization   — middleware.Guardrail (Order 20, REUSED).
//  3. Rate limiting   — middleware.RateLimit (Order 30, REUSED, stricter limits).
//  4. Content safety  — middleware.ContentSafety (Order 45, REUSED).
//
// It builds the full built-in chain via middleware.DefaultChain (which also
// brings ContextInjection, Audit, and ErrorHandling — the gateway keeps those:
// dropping them would lose request-context propagation, audit, and panic
// recovery) and then wraps it with the authentication interceptor using
// spi.Chain.With. Because Execute orders by Order(), the interceptor's Order 5
// lands outermost without any manual re-ordering.
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
