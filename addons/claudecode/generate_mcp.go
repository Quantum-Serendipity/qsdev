package claudecode

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// McpJSON is the top-level structure that marshals to .mcp.json.
type McpJSON struct {
	MCPServers map[string]MCPServerEntry `json:"mcpServers"`
}

// MCPServerEntry represents a single MCP server entry in .mcp.json.
type MCPServerEntry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args"`
	Env     map[string]string `json:"env,omitempty"`
}

// knownMCPServers maps well-known server names to their default templates.
var knownMCPServers = map[string]MCPServerEntry{
	"github": {
		Command: "npx",
		Args:    []string{"@anthropic-ai/mcp-github"},
		Env:     map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"},
	},
	"filesystem": {
		Command: "npx",
		Args:    []string{"@anthropic-ai/mcp-filesystem"},
	},
	"postgres": {
		Command: "npx",
		Args:    []string{"@anthropic-ai/mcp-postgres"},
		Env:     map[string]string{"DATABASE_URL": "${DATABASE_URL}"},
	},
	"fetch": {
		Command: "npx",
		Args:    []string{"@anthropic-ai/mcp-fetch"},
	},
	"socket": {
		Command: "npx",
		Args:    []string{"@anthropic-ai/mcp-socket"},
		Env:     map[string]string{"SOCKET_SECURITY_API_KEY": "${SOCKET_SECURITY_API_KEY}"},
	},
	"semble": {
		Command: "uvx",
		Args:    []string{"--from", "semble[mcp]", "semble"},
	},
}

// GenerateMcpJson produces a .mcp.json file from the wizard answers and addon
// configuration. It returns nil, nil when no MCP servers are requested.
func GenerateMcpJson(answers types.WizardAnswers, cfg Config) (*types.GeneratedFile, error) {
	if len(answers.MCPServers) == 0 && len(cfg.MCPServers) == 0 {
		return nil, nil
	}

	mcp := McpJSON{
		MCPServers: make(map[string]MCPServerEntry),
	}

	// Populate from wizard-selected known servers.
	for _, name := range answers.MCPServers {
		tmpl, ok := knownMCPServers[name]
		if !ok {
			return nil, fmt.Errorf("unknown MCP server %q: must be one of %s", name, knownServerNames())
		}
		mcp.MCPServers[name] = tmpl
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

	// Apply semble text-files flag if enabled.
	if entry, ok := mcp.MCPServers["semble"]; ok && answers.AgentTools.SembleTextFiles {
		entry.Args = append(append([]string{}, entry.Args...), "--include-text-files")
		mcp.MCPServers["semble"] = entry
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
		Mode:     0o644,
		Strategy: types.ThreeWayMerge,
	}, nil
}

// knownServerNames returns a sorted, comma-separated list of known server
// names for use in error messages.
func knownServerNames() string {
	names := make([]string, 0, len(knownMCPServers))
	for k := range knownMCPServers {
		names = append(names, k)
	}
	sort.Strings(names)

	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += fmt.Sprintf("%q", n)
	}
	return out
}
