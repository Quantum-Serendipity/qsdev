package mcpregistry

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserver"
)

// buildDefault creates and populates the default MCP server registry from
// catalog-driven definitions, catalog tool metadata, and embedded MCP server providers.
func buildDefault() *McpServerRegistry {
	return buildRegistry(catalog.Default)
}

// buildRegistry populates a registry from the catalog returned by loadCatalog
// plus the built-in and embedded servers. The catalog is loaded once. If it
// fails to load, the built-in servers are still registered and the failure is
// recorded for CatalogErr, so callers can report that the catalog-defined
// servers are missing instead of silently showing a smaller registry.
func buildRegistry(loadCatalog func() (*catalog.Catalog, error)) *McpServerRegistry {
	r := NewRegistry()

	cat, err := loadCatalog()
	if err != nil {
		r.catalogErr = fmt.Errorf("loading MCP server catalog: %w", err)
	} else {
		for name, def := range cat.MCPServers() {
			r.MustRegister(catalogDefToRegistryDef(name, def))
		}
		enrichFromCatalog(r, cat)
	}

	registerUniversalServer(r)
	populateFromEmbeddedProviders(r)

	return r
}

// catalogDefToRegistryDef converts a catalog MCPServerDef into a McpServerDefinition.
func catalogDefToRegistryDef(name string, def catalog.MCPServerDef) McpServerDefinition {
	return McpServerDefinition{
		Name:          name,
		DisplayName:   def.DisplayName,
		Category:      parseMcpCategory(def.Category),
		Description:   def.Description,
		Command:       def.Command,
		Args:          def.Args,
		URL:           def.URL,
		Env:           def.Env,
		RequiredEnv:   def.RequiredEnv,
		Transport:     parseMcpTransport(def.Transport),
		Source:        SourceBuiltin,
		InstallMethod: parseMcpInstallMethod(def.InstallMethod),
		PackageName:   def.PackageName,
	}
}

// parseMcpCategory maps a YAML category string to the typed McpCategory constant.
func parseMcpCategory(s string) McpCategory {
	switch s {
	case "documentation":
		return CategoryDocumentation
	case "security":
		return CategorySecurity
	case "integration":
		return CategoryIntegration
	case "agent":
		return CategoryAgent
	case "infrastructure":
		return CategoryInfrastructure
	default:
		return McpCategory(s)
	}
}

// parseMcpTransport maps a YAML transport string to the typed McpTransport constant.
func parseMcpTransport(s string) McpTransport {
	switch s {
	case "stdio", "":
		return TransportStdio
	case "sse":
		return TransportSSE
	case "http":
		return TransportHTTP
	default:
		return McpTransport(s)
	}
}

// parseMcpInstallMethod maps a YAML install_method string to the typed McpInstallMethod constant.
func parseMcpInstallMethod(s string) McpInstallMethod {
	switch s {
	case "uv-tool":
		return InstallUvTool
	case "npm-global":
		return InstallNpmGlobal
	case "nix-package":
		return InstallNixPackage
	case "manual":
		return InstallManual
	default:
		return InstallManual
	}
}

// enrichFromCatalog cross-references the registry with the tool catalog.
// For each catalog tool that declares an MCPServerName, the matching registry
// entry gets its ToolRegName set and its DisplayName filled in if empty.
func enrichFromCatalog(r *McpServerRegistry, cat *catalog.Catalog) {
	for toolName, tool := range cat.Tools() {
		if tool.MCPServerName == "" {
			continue
		}

		r.Modify(tool.MCPServerName, func(def *McpServerDefinition) {
			def.ToolRegName = toolName
			if def.DisplayName == "" {
				def.DisplayName = tool.DisplayName
			}
		})
	}
}

// populateFromEmbeddedProviders adds any embedded MCP server providers that
// are not already present in the registry. This ensures servers registered
// via mcpserver.DefaultRegistry().Register() at init time are included.
func populateFromEmbeddedProviders(r *McpServerRegistry) {
	for _, p := range mcpserver.DefaultRegistry().All() {
		if _, exists := r.ByName(p.Name()); exists {
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
