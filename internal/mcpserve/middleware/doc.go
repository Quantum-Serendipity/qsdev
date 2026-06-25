// Package middleware implements the universal MCP server's built-in tool-call
// middleware chain. Each layer is a spi.Middleware ordered by Order(): lower
// values wrap further OUT (they see the request first and the response last),
// higher values wrap further IN (closest to the real tool handler). The chain
// is assembled by DefaultChain into a framework-neutral *spi.Chain and injected
// into the server via mcpserve.WithChain.
//
// Onion order (outermost -> innermost):
//
//	10  ContextInjection  publishes the resolved *spi.ToolCallContext onto ctx
//	15  Audit             records every call decision (OUTSIDE Guardrail/RateLimit
//	                      so denied/throttled calls are still audited — corr. C9)
//	20  Guardrail         permission cascade; DENY short-circuits with a tool error
//	30  RateLimit         token bucket per (agent,category) + per-category semaphore
//	45  ContentSafety     post-processes the result, redacting secret patterns
//	50  ErrorHandling     converts handler errors/panics into tool-level errors
//	                      (propagating only context cancellation/deadline)
//
// Addon and external middleware MUST use Order() >= AddonOrderFloor (100); the
// 0..99 band is reserved for these built-in layers. This package does not
// implement addon middleware.
//
// The spec (internal-docs/implementation-plan/phases/32-universal-mcp-server.md,
// Unit 32.8) cites two research spikes for the 19-category taxonomy, the
// per-category rate-limit defaults, and the permission-cascade schema. Those
// spike directories are not present in the tree, so the taxonomy and limit
// defaults in categories.go are principled defaults documented at their
// definition. The one concrete value the surrounding spec pins down — a maximum
// of 3 concurrent process-execution (nix_run) invocations — is honored.
package middleware

// AddonOrderFloor is the lowest Order() value reserved for addon/external
// middleware. Built-in chain layers occupy 0..99; addons register at 100+.
const AddonOrderFloor = 100

// Built-in layer Order() values. Lower is more outer.
const (
	orderContextInjection = 10
	orderAudit            = 15
	orderGuardrail        = 20
	orderRateLimit        = 30
	orderContentSafety    = 45
	orderErrorHandling    = 50
)
