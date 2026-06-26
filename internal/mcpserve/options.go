package mcpserve

import (
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// config holds the resolved construction parameters for a Server.
type config struct {
	name         string
	version      string
	projectRoot  string
	instructions string
	adapters     *spi.AdapterRegistry
	chain        *spi.Chain
	multiAdapter bool
}

// Option configures a Server at construction time.
type Option func(*config)

// defaultConfig returns a config populated with sensible defaults. The server
// name derives from branding and the version from the build info so the
// advertised serverInfo matches the rest of the CLI.
//
// The default chain is the real built-in middleware chain (redaction, guardrail,
// rate-limiting, audit), NOT an empty chain: a Server constructed without an
// explicit WithChain must still enforce on every surface. An empty default would
// be fail-open — handlers would run with no redaction, no guardrail, and no audit.
func defaultConfig() config {
	return config{
		name:     branding.Get().AppName + "-mcp",
		version:  version.Info().Version,
		adapters: spi.DefaultRegistry(),
		chain:    middleware.DefaultChain(),
		instructions: "qsdev universal MCP server. Exposes qsdev tooling over MCP; " +
			"tools operate within the resolved project root.",
	}
}

// WithName overrides the advertised server implementation name.
func WithName(name string) Option {
	return func(c *config) {
		if name != "" {
			c.name = name
		}
	}
}

// WithVersion overrides the advertised server implementation version.
func WithVersion(v string) Option {
	return func(c *config) {
		if v != "" {
			c.version = v
		}
	}
}

// WithProjectRoot sets the resolved project root the server operates within.
func WithProjectRoot(root string) Option {
	return func(c *config) { c.projectRoot = root }
}

// WithInstructions overrides the server instructions surfaced to clients.
func WithInstructions(s string) Option {
	return func(c *config) { c.instructions = s }
}

// WithAdapterRegistry overrides the adapter registry consumed at construction.
// When nil the package-level DefaultRegistry is used.
func WithAdapterRegistry(r *spi.AdapterRegistry) Option {
	return func(c *config) {
		if r != nil {
			c.adapters = r
		}
	}
}

// WithChain overrides the tool-handler middleware chain. A nil chain is ignored,
// leaving the built-in middleware.DefaultChain installed by defaultConfig — the
// server is never left fail-open.
func WithChain(chain *spi.Chain) Option {
	return func(c *config) {
		if chain != nil {
			c.chain = chain
		}
	}
}

// WithMultiAdapter forces every registered adapter to be mounted regardless of
// its Applies() result and makes the per-request tool filter expose all mounted
// tools regardless of the connected client. It backs the serve command's
// --multi-adapter flag, used for integration testing and diagnostics where the
// client identifies as none of the supported frameworks.
func WithMultiAdapter(enabled bool) Option {
	return func(c *config) { c.multiAdapter = enabled }
}
