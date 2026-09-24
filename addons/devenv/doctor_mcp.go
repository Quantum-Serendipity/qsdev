package devenv

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
)

// mcpConfigSection statically validates the MCP servers the .mcp.json at
// projectRoot configures (see mcphealth.ValidateConfig): each command is looked
// up on PATH, each URL is checked for https, and each required or referenced
// environment variable must be set. No server is started, since .mcp.json is
// repository content and may name any command. Required variables come from
// the matching definition in reg. It returns nil outside a project and for a
// project with no MCP servers; an unreadable .mcp.json or catalog becomes a
// warning in the section.
func mcpConfigSection(projectRoot string, reg *mcpregistry.McpServerRegistry) *doctor.MCPSection {
	if projectRoot == "" {
		return nil
	}
	servers, err := mcpregistry.ConfiguredServers(projectRoot, reg)
	if err != nil {
		return doctor.NewMCPSection(nil, nil, []string{
			fmt.Sprintf("MCP servers not validated: %v", err),
		})
	}
	if len(servers) == 0 {
		return nil
	}

	var warnings []string
	if err := reg.CatalogErr(); err != nil {
		warnings = append(warnings, fmt.Sprintf("required environment of catalog-defined servers not checked: %v", err))
	}
	byName := make(map[string]mcphealth.ServerConfig, len(servers))
	for _, s := range servers {
		byName[s.Name] = s
	}
	return doctor.NewMCPSection(servers, mcphealth.ValidateConfig(byName), warnings)
}
