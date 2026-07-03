package spi

import "context"

// ToolRequest is the mcp-go-independent representation of a tool invocation.
type ToolRequest struct {
	// Name is the tool name being invoked.
	Name string
	// Arguments holds the decoded call arguments.
	Arguments map[string]any
	// Meta holds the raw request _meta map (reverse-DNS extension keys, etc.),
	// or nil when the client supplied none.
	Meta map[string]any
}

// ToolResult is the mcp-go-independent representation of a tool result.
type ToolResult struct {
	// Text is the unstructured textual result content.
	Text string
	// Structured optionally carries a structured JSON-serializable result.
	Structured any
	// IsError marks the result as a tool-level (non-protocol) error.
	IsError bool
}

// ResourceRequest is the neutral representation of a resources/read invocation.
type ResourceRequest struct {
	URI       string
	Arguments map[string]any
}

// ResourceResult is the neutral representation of a resources/read result. Each
// entry becomes one returned resource-contents block.
type ResourceResult struct {
	Contents []ResourceContent
}

// ResourceContent is a single text resource-contents block.
type ResourceContent struct {
	URI      string
	MIMEType string
	Text     string
}

// PromptRequest is the neutral representation of a prompts/get invocation.
type PromptRequest struct {
	Name      string
	Arguments map[string]string
}

// PromptResult is the neutral representation of a prompts/get result.
type PromptResult struct {
	Description string
	Messages    []PromptMessage
}

// PromptRole identifies the sender of a prompt message.
type PromptRole string

const (
	// PromptRoleUser marks a message authored by the user.
	PromptRoleUser PromptRole = "user"
	// PromptRoleAssistant marks a message authored by the assistant.
	PromptRoleAssistant PromptRole = "assistant"
)

// PromptMessage is a single text message within a prompt result.
type PromptMessage struct {
	Role PromptRole
	Text string
}

// ToolHandler executes a tool. It receives the resolved ToolCallContext so it
// can act on the caller's identity and project root without re-deriving them.
type ToolHandler func(ctx context.Context, cc *ToolCallContext, req *ToolRequest) (*ToolResult, error)

// ResourceHandler reads a resource.
type ResourceHandler func(ctx context.Context, cc *ToolCallContext, req *ResourceRequest) (*ResourceResult, error)

// PromptHandler renders a prompt.
type PromptHandler func(ctx context.Context, cc *ToolCallContext, req *PromptRequest) (*PromptResult, error)

// ToolRegistration declares a single tool exposed by the server.
type ToolRegistration struct {
	// Name is the unique tool name.
	Name string
	// Description is the human-readable tool description.
	Description string
	// InputSchema is the tool's JSON Schema input definition as a decoded map
	// (e.g. {"type":"object","properties":{...},"required":[...]}). A nil or
	// empty map yields a permissive object schema.
	InputSchema map[string]any
	// Category groups related tools (e.g. "filesystem", "search").
	Category string
	// Tier is an ordering/grouping hint; lower tiers are considered more core.
	Tier int
	// Handler executes the tool.
	Handler ToolHandler
}

// ResourceRegistration declares a single resource exposed by the server.
type ResourceRegistration struct {
	URI         string
	Name        string
	Description string
	MIMEType    string
	// Category groups the resource under a taxonomy category (mirroring
	// ToolRegistration.Category), so category-scoped Guardrail denies and the
	// per-category rate limiter apply to resource reads. An empty category leaves
	// the read uncategorized (Guardrail permissive-by-default, Default rate
	// limit), matching prior behavior — the point is that a category-scoped policy
	// CAN now cover a resource.
	Category string
	Handler  ResourceHandler
}

// PromptRegistration declares a single prompt exposed by the server.
type PromptRegistration struct {
	Name        string
	Description string
	Arguments   []PromptArgument
	// Category groups the prompt under a taxonomy category (mirroring
	// ToolRegistration.Category), so category-scoped Guardrail denies and the
	// per-category rate limiter apply to prompt renders. An empty category leaves
	// the render uncategorized (matching prior behavior).
	Category string
	Handler  PromptHandler
}

// PromptArgument describes one templating argument a prompt accepts.
type PromptArgument struct {
	Name        string
	Description string
	Required    bool
}
