package mcpserve

import (
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
}

// Option configures a Server at construction time.
type Option func(*config)

// defaultConfig returns a config populated with sensible defaults. The server
// name derives from branding and the version from the build info so the
// advertised serverInfo matches the rest of the CLI.
func defaultConfig() config {
	return config{
		name:     branding.Get().AppName + "-mcp",
		version:  version.Info().Version,
		adapters: spi.DefaultRegistry(),
		chain:    spi.NewChain(),
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

// WithChain sets the tool-handler middleware chain. When nil an empty chain is
// used (handlers run directly).
func WithChain(chain *spi.Chain) Option {
	return func(c *config) {
		if chain != nil {
			c.chain = chain
		}
	}
}
