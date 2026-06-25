package mcpserve

import (
	"context"

	"github.com/mark3labs/mcp-go/server"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// Server is qsdev's universal MCP server. It wraps a vendored mcp-go
// *server.MCPServer and holds the resolved project root, adapter registry, and
// tool-handler middleware chain. Construct one with New, then run it over a
// transport (see transport.go).
type Server struct {
	mcp         *server.MCPServer
	projectRoot string
	adapters    *spi.AdapterRegistry
	chain       *spi.Chain
}

// New constructs a Server. It builds the underlying mcp-go server with tool and
// logging capabilities advertised (the protocol version, 2025-11-25, is
// negotiated by mcp-go against the client's request), applies panic recovery,
// and mounts every applicable framework adapter's contributions.
func New(opts ...Option) *Server {
	cfg := defaultConfig()
	for _, o := range opts {
		o(&cfg)
	}

	mcpOpts := []server.ServerOption{
		// Advertise tools even before any are mounted: the universal server's
		// purpose is to expose qsdev tooling, and adapters add tools at mount
		// time. listChanged=true lets late-mounted tools notify clients.
		server.WithToolCapabilities(true),
		// Advertise log-message support; diagnostics may be surfaced to clients.
		server.WithLogging(),
		// Recover panics in tool handlers into protocol errors.
		server.WithRecovery(),
	}
	if cfg.instructions != "" {
		mcpOpts = append(mcpOpts, server.WithInstructions(cfg.instructions))
	}

	s := &Server{
		mcp:         server.NewMCPServer(cfg.name, cfg.version, mcpOpts...),
		projectRoot: cfg.projectRoot,
		adapters:    cfg.adapters,
		chain:       cfg.chain,
	}

	// Mounting resources/prompts implicitly advertises those capabilities via
	// mcp-go; tools are already advertised above.
	s.mountAdapters(context.Background())

	return s
}

// MCPServer returns the underlying mcp-go server. Exposed so transports and
// tests can drive it directly (e.g. NewStdioServer(s.MCPServer())).
func (s *Server) MCPServer() *server.MCPServer { return s.mcp }

// ProjectRoot returns the resolved project root the server operates within.
func (s *Server) ProjectRoot() string { return s.projectRoot }

// mountAdapters mounts the contributions of every applicable adapter in the
// registry. With no registered adapters (the Task-1 state) this is a no-op.
func (s *Server) mountAdapters(ctx context.Context) {
	for _, a := range s.adapters.All() {
		if !a.Applies(ctx, s.projectRoot) {
			continue
		}
		for _, t := range a.Tools() {
			s.mountTool(t)
		}
		for _, r := range a.Resources() {
			s.mountResource(r)
		}
		for _, p := range a.Prompts() {
			s.mountPrompt(p)
		}
	}
}
