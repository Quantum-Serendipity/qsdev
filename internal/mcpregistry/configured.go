package mcpregistry

import (
	"fmt"
	"maps"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
)

// ConfiguredServers returns the servers the .mcp.json at projectRoot
// configures, as mcphealth configs sorted by name. Only the project's own
// servers are returned, not the whole catalog. A server's required environment
// variables come from the definition of the same name in reg, since .mcp.json
// does not record them; a nil reg adds none. A missing .mcp.json yields no
// servers and no error.
func ConfiguredServers(projectRoot string, reg *McpServerRegistry) ([]mcphealth.ServerConfig, error) {
	defs, err := ScanMcpJSON(projectRoot)
	if err != nil {
		return nil, fmt.Errorf("loading configured MCP servers: %w", err)
	}
	out := make([]mcphealth.ServerConfig, 0, len(defs))
	for _, name := range slices.Sorted(maps.Keys(defs)) {
		def := defs[name]
		cfg := mcphealth.ServerConfig{
			Name:    name,
			Command: def.Command,
			Args:    def.Args,
			URL:     def.URL,
			Env:     def.Env,
			Headers: def.Headers,
		}
		if reg != nil {
			if known, ok := reg.ByName(name); ok {
				cfg.RequiredEnv = known.RequiredEnv
			}
		}
		out = append(out, cfg)
	}
	return out, nil
}
