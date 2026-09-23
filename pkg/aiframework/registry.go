package aiframework

import (
	"context"

	"github.com/Quantum-Serendipity/qsdev/internal/enumtext"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// MCPTransport identifies the transport protocol for an MCP server.
type MCPTransport int

const (
	TransportStdio MCPTransport = iota
	TransportStreamableHTTP
	TransportSSE
)

var mcpTransportNames = [...]string{
	TransportStdio:          "stdio",
	TransportStreamableHTTP: "streamable-http",
	TransportSSE:            "sse",
}

var mCPTransportText = enumtext.New[MCPTransport]("MCPTransport", "MCP transport", "unknown", mcpTransportNames[:])

func (t MCPTransport) String() string { return mCPTransportText.String(t) }

func (t MCPTransport) MarshalText() ([]byte, error) { return mCPTransportText.MarshalText(t) }

func (t *MCPTransport) UnmarshalText(text []byte) error {
	return mCPTransportText.UnmarshalText(text, t)
}

// MCPServerSpec describes an MCP server for registration and config generation.
type MCPServerSpec struct {
	Name         string
	Description  string
	Command      string
	Args         []string
	Env          map[string]string
	URL          string
	Transport    MCPTransport
	Tools        []MCPToolSpec
	SecurityTier int
	Priority     int
	Categories   []string
}

// MCPToolSpec describes a single tool provided by an MCP server.
type MCPToolSpec struct {
	Name        string
	Description string
	Category    string
}

const (
	ToolCeilingCursor   = 40
	ToolCeilingWindsurf = 100
	ToolCeilingCopilot  = 128
)

// RegistryClient manages MCP server configuration for a specific framework.
type RegistryClient interface {
	FrameworkID() FrameworkID
	SupportedTransports() []MCPTransport
	ToolCeiling() int
	GenerateMCPConfig(ctx context.Context, servers []MCPServerSpec, credentials map[string]string) ([]types.GeneratedFile, error)
	FilterServers(servers []MCPServerSpec) []MCPServerSpec
	ValidateServers(ctx context.Context, servers []MCPServerSpec) []ValidationIssue
}
