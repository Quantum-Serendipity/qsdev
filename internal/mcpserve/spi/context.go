package spi

import "context"

// ClientInfo is the mcp-go-independent representation of an MCP client's
// self-reported identity, taken from the initialize handshake's clientInfo
// object. It deliberately mirrors only the fields qsdev cares about so that the
// spi package never needs to import mcp-go.
type ClientInfo struct {
	// Name is the client's reported implementation name (e.g. "claude-code").
	Name string
	// Version is the client's reported implementation version, if any.
	Version string
	// Title is the optional human-readable display name, if any.
	Title string
}

// ToolCallContext carries the resolved, request-scoped identity and environment
// for a single tool/resource/prompt invocation. Instances are treated as
// immutable once constructed: handlers and middleware read from a
// *ToolCallContext but must not mutate it. Build a derived copy if a layer
// needs to refine a field.
type ToolCallContext struct {
	// AgentID is the resolved agent identity. It is the per-request _meta
	// override when present, otherwise the client's reported Name, otherwise
	// "unknown".
	AgentID string
	// Client is the identity reported during the initialize handshake.
	Client ClientInfo
	// ProjectRoot is the resolved absolute project root for this session.
	ProjectRoot string
	// ToolName is the name of the tool/resource/prompt being invoked.
	ToolName string
}

// ctxKey is the unexported context key type used to store a *ToolCallContext.
type ctxKey struct{}

// WithToolCallContext returns a copy of ctx that carries cc.
func WithToolCallContext(ctx context.Context, cc *ToolCallContext) context.Context {
	return context.WithValue(ctx, ctxKey{}, cc)
}

// FromContext returns the *ToolCallContext stored in ctx, if any. The boolean
// reports whether a context was present.
func FromContext(ctx context.Context) (*ToolCallContext, bool) {
	cc, ok := ctx.Value(ctxKey{}).(*ToolCallContext)
	return cc, ok
}
