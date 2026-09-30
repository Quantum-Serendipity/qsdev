package devenv

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
)

// mcpConfigSection is the doctor report's MCP section (see
// doctor.MCPFindings); an unreadable .mcp.json becomes a section warning.
func mcpConfigSection(projectRoot string, reg *mcpregistry.McpServerRegistry) *doctor.MCPSection {
	ms, err := doctor.MCPFindings(projectRoot, reg)
	if err != nil {
		return doctor.NewMCPSection(nil, nil, []string{fmt.Sprintf("MCP servers not validated: %v", err)})
	}
	return ms
}
