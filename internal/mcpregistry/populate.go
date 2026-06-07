package mcpregistry

import (
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserver"
)

// buildDefault creates and populates the default MCP server registry from
// built-in definitions, catalog metadata, and embedded MCP server providers.
func buildDefault() *McpServerRegistry {
	r := NewRegistry()

	for _, def := range knownServers {
		r.MustRegister(def)
	}

	enrichFromCatalog(r)
	populateFromEmbeddedProviders(r)

	return r
}

// enrichFromCatalog cross-references the registry with the tool catalog.
// For each catalog tool that declares an MCPServerName, the matching registry
// entry gets its ToolRegName set and its DisplayName filled in if empty.
func enrichFromCatalog(r *McpServerRegistry) {
	cat, err := catalog.Default()
	if err != nil {
		// Catalog unavailable — skip enrichment silently so the registry
		// still works with the built-in definitions alone.
		return
	}

	for toolName, tool := range cat.Tools() {
		if tool.MCPServerName == "" {
			continue
		}

		def, ok := r.Get(tool.MCPServerName)
		if !ok {
			continue
		}

		r.mu.Lock()
		def.ToolRegName = toolName
		if def.DisplayName == "" {
			def.DisplayName = tool.DisplayName
		}
		r.mu.Unlock()
	}
}

// populateFromEmbeddedProviders adds any embedded MCP server providers that
// are not already present in the registry. This ensures servers registered
// via mcpserver.DefaultRegistry().Register() at init time are included.
func populateFromEmbeddedProviders(r *McpServerRegistry) {
	for _, p := range mcpserver.DefaultRegistry().All() {
		if _, exists := r.Get(p.Name()); exists {
			continue
		}

		def := McpServerDefinition{
			Name:        p.Name(),
			DisplayName: p.Description(),
			Description: p.Description(),
			Command:     "qsdev",
			Args:        []string{"mcp", p.Name()},
			Transport:   TransportStdio,
			Source:      SourceBuiltin,
		}

		// Best-effort; skip if a race somehow registered it.
		_ = r.Register(def)
	}
}
