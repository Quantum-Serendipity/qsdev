package claudecode

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// McpJSON is the top-level structure that marshals to .mcp.json.
type McpJSON struct {
	MCPServers map[string]MCPServerEntry `json:"mcpServers"`
}

// MCPServerEntry represents a single MCP server entry in .mcp.json.
type MCPServerEntry struct {
	Command     string            `json:"command"`
	Args        []string          `json:"args"`
	Env         map[string]string `json:"env,omitempty"`
	RequiredEnv []string          `json:"-"`
}

// GenerateMcpJson produces a .mcp.json file from the wizard answers and addon
// configuration. It returns nil, nil when no MCP servers are requested.
func GenerateMcpJson(answers types.WizardAnswers, cfg Config) (*types.GeneratedFile, error) {
	if len(answers.MCPServers) == 0 && len(cfg.MCPServers) == 0 {
		return nil, nil
	}

	cat, err := catalog.Default()
	if err != nil {
		return nil, fmt.Errorf("loading catalog for MCP server definitions: %w", err)
	}

	mcp := McpJSON{
		MCPServers: make(map[string]MCPServerEntry),
	}

	// Populate from wizard-selected known servers.
	for _, name := range answers.MCPServers {
		def, ok := cat.MCPServer(name)
		if !ok {
			return nil, fmt.Errorf("unknown MCP server %q: must be one of %s", name, mcpServerNameList(cat))
		}
		mcp.MCPServers[name] = MCPServerEntry{
			Command: def.Command,
			Args:    def.Args,
			Env:     def.Env,
		}
	}

	// Populate from config-provided servers (overrides wizard on collision).
	for _, srv := range cfg.MCPServers {
		entry := MCPServerEntry{
			Command: srv.Command,
			Args:    srv.Args,
			Env:     srv.Env,
		}
		mcp.MCPServers[srv.Name] = entry
	}

	jsonBytes, err := json.MarshalIndent(mcp, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling .mcp.json: %w", err)
	}

	// Append trailing newline for POSIX compliance.
	jsonBytes = append(jsonBytes, '\n')

	return &types.GeneratedFile{
		Path:     ".mcp.json",
		Content:  jsonBytes,
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.ThreeWayMerge,
	}, nil
}

// mcpServerNameList returns a sorted, comma-separated list of known server
// names from the catalog for use in error messages.
func mcpServerNameList(cat *catalog.Catalog) string {
	names := cat.MCPServerNames()

	var parts []string
	for _, n := range names {
		parts = append(parts, fmt.Sprintf("%q", n))
	}
	return strings.Join(parts, ", ")
}
