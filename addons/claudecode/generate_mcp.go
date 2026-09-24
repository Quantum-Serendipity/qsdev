package claudecode

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/merge"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// McpJSON is the top-level structure that marshals to .mcp.json.
type McpJSON struct {
	MCPServers map[string]MCPServerEntry `json:"mcpServers"`
}

// MCPServerEntry represents a single MCP server entry in .mcp.json.
// HTTP-transport servers use Type+URL (plus optional Headers); stdio servers
// use Command+Args.
type MCPServerEntry struct {
	Type        string            `json:"type,omitempty"`
	URL         string            `json:"url,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Command     string            `json:"command,omitempty"`
	Args        []string          `json:"args,omitempty"`
	Env         map[string]string `json:"env,omitempty"`
	RequiredEnv []string          `json:"-"`
}

// GenerateMcpJson produces a .mcp.json file from the wizard answers and addon
// configuration. Servers the client MCP policy (answers.MCPPolicy) does not
// permit are left out, whichever source requested them. It returns nil, nil
// when no permitted MCP servers are requested and there is no policy; under a
// policy it always returns a file (possibly with no servers), so the writers
// that merge it over an existing .mcp.json remove the servers the policy
// forbids (merge.EnforceMCPPolicy).
func GenerateMcpJson(answers types.WizardAnswers, cfg Config) (*types.GeneratedFile, error) {
	policy := answers.MCPPolicy
	names := policy.Filter(answers.MCPServers)
	var configured []MCPServerConfig
	for _, srv := range cfg.MCPServers {
		if policy.Permits(srv.Name) {
			configured = append(configured, srv)
		}
	}
	if len(names) == 0 && len(configured) == 0 && policy.IsZero() {
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
	for _, name := range names {
		def, ok := cat.MCPServer(name)
		if !ok {
			return nil, fmt.Errorf("unknown MCP server %q: must be one of %s", name, mcpServerNameList(cat))
		}
		mcp.MCPServers[name] = catalogDefToEntry(def)
	}

	// Populate from config-provided servers (overrides wizard on collision).
	for _, srv := range configured {
		entry := MCPServerEntry{
			Command: srv.Command,
			Args:    srv.Args,
			Env:     srv.Env,
		}
		mcp.MCPServers[srv.Name] = entry
	}

	raw, err := json.Marshal(mcp)
	if err != nil {
		return nil, fmt.Errorf("marshaling .mcp.json: %w", err)
	}
	// Emit the same canonical form the three-way merge writes.
	jsonBytes, err := merge.CanonicalJSON(raw)
	if err != nil {
		return nil, fmt.Errorf("canonicalizing .mcp.json: %w", err)
	}

	return &types.GeneratedFile{
		Path:     ".mcp.json",
		Content:  jsonBytes,
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.ThreeWayMerge,
	}, nil
}

// catalogDefToEntry converts a catalog MCP server definition to a .mcp.json entry.
// HTTP-transport servers get type+url; stdio servers get command+args.
func catalogDefToEntry(def catalog.MCPServerDef) MCPServerEntry {
	if def.Transport == "http" && def.URL != "" {
		return MCPServerEntry{
			Type: "http",
			URL:  def.URL,
			Env:  def.Env,
		}
	}
	return MCPServerEntry{
		Command: def.Command,
		Args:    def.Args,
		Env:     def.Env,
	}
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
