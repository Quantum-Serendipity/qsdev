package aiframework

import (
	"context"
	"fmt"

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

func (t MCPTransport) String() string {
	if int(t) >= 0 && int(t) < len(mcpTransportNames) {
		return mcpTransportNames[t]
	}
	return "unknown"
}

func (t MCPTransport) MarshalText() ([]byte, error) {
	s := t.String()
	if s == "unknown" {
		return nil, fmt.Errorf("cannot marshal unknown MCPTransport value %d", int(t))
	}
	return []byte(s), nil
}

func (t *MCPTransport) UnmarshalText(text []byte) error {
	for i, name := range mcpTransportNames {
		if name == string(text) {
			*t = MCPTransport(i)
			return nil
		}
	}
	return fmt.Errorf("unknown MCP transport: %q", string(text))
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
