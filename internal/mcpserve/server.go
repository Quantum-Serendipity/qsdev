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
	mcp          *server.MCPServer
	projectRoot  string
	adapters     *spi.AdapterRegistry
	chain        *spi.Chain
	catalog      *catalog
	multiAdapter bool
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

	// Construct the Server shell first so the tool filter installed below can
	// close over it. s.mcp is assigned immediately after; the filter only runs
	// at tools/list time, long after the catalog has been populated by mounting.
	s := &Server{
		projectRoot:  cfg.projectRoot,
		adapters:     cfg.adapters,
		chain:        cfg.chain,
		catalog:      newCatalog(),
		multiAdapter: cfg.multiAdapter,
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
		// Per-tools/list-request filter implementing the mount-all-then-filter
		// model: the server mounts every project-applicable adapter's tools at
		// construction, and this narrows each client's view to generic tools plus
		// the tools of the frameworks that client matches (see toolFilter).
		server.WithToolFilter(s.toolFilter),
	}
	if cfg.instructions != "" {
		mcpOpts = append(mcpOpts, server.WithInstructions(cfg.instructions))
	}

	s.mcp = server.NewMCPServer(cfg.name, cfg.version, mcpOpts...)

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

// ProjectContributor supplies the generic, framework-agnostic tool, resource,
// and prompt registrations rendered by the project context engine
// (internal/mcpserve/projectctx). Its *ProjectContext satisfies this interface.
// Defining the seam as a local interface keeps server.go free of an import on
// projectctx (projectctx imports spi, never mcpserve), so there is no cycle.
type ProjectContributor interface {
	Tools() []spi.ToolRegistration
	Resources() []spi.ResourceRegistration
	Prompts() []spi.PromptRegistration
}

// MountProjectContext mounts every tool, resource, and prompt contributed by pc.
// Unlike adapter contributions these are framework-agnostic and always mounted.
// Call it before serving. Mounting resources/prompts implicitly advertises those
// capabilities via mcp-go; tools are already advertised by New.
func (s *Server) MountProjectContext(pc ProjectContributor) {
	for _, t := range pc.Tools() {
		s.mountTool(t)
	}
	for _, r := range pc.Resources() {
		s.mountResource(r)
	}
	for _, p := range pc.Prompts() {
		s.mountPrompt(p)
	}
}

// MountTools mounts a set of framework-agnostic tool registrations (e.g. the
// security and devenv tools from internal/mcpserve/tools). Like the project
// context surface they are recorded under the generic owner so the per-request
// tool filter always keeps them visible. Call it before serving.
func (s *Server) MountTools(regs []spi.ToolRegistration) {
	for _, t := range regs {
		s.mountTool(t)
	}
}

// mountAdapters mounts the contributions of every applicable adapter in the
// registry, recording each tool/resource under its owning framework id so the
// tool filter can later scope visibility per client. An adapter is mounted when
// it Applies to the resolved project root, or unconditionally in multi-adapter
// mode. With no registered adapters (the Task-1 state) this is a no-op.
func (s *Server) mountAdapters(ctx context.Context) {
	for _, a := range s.adapters.All() {
		if !s.multiAdapter && !a.Applies(ctx, s.projectRoot) {
			continue
		}
		owner := string(a.ID())
		for _, t := range a.Tools() {
			s.mountToolOwned(t, owner)
		}
		for _, r := range a.Resources() {
			s.mountResourceOwned(r, owner)
		}
		for _, p := range a.Prompts() {
			s.mountPrompt(p)
		}
	}
}
