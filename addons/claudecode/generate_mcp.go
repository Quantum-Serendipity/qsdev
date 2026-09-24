package claudecode

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
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
//
// A catalog server that `qsdev mcp install` installed at its pinned release
// (recorded in the project state under answers.ProjectRoot) runs its installed
// binary; any other runs the catalog's pinned launcher. GenerateMcpJson
// refuses, with ErrUnpinnedMCPServer, any server whose command would fetch a
// package at launch without an exact version, whether it comes from the
// catalog or the addon configuration.
func GenerateMcpJson(answers types.WizardAnswers, cfg Config) (*types.GeneratedFile, error) {
	servers, err := buildMcpServers(answers, cfg)
	if err != nil {
		return nil, err
	}
	if len(servers) == 0 && answers.MCPPolicy.IsZero() {
		return nil, nil
	}
	mcp := McpJSON{MCPServers: servers}

	for _, name := range slices.Sorted(maps.Keys(mcp.MCPServers)) {
		if err := requirePinnedLaunch(name, mcp.MCPServers[name]); err != nil {
			return nil, fmt.Errorf("generating .mcp.json: %w", err)
		}
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

// buildMcpServers returns the .mcp.json server entries qsdev generates: the
// wizard-selected catalog servers and the config-provided servers (which win on
// a name collision), less those the client MCP policy (answers.MCPPolicy) does
// not permit. A catalog server `qsdev mcp install` installed runs its installed
// binary (catalogServerEntry).
func buildMcpServers(answers types.WizardAnswers, cfg Config) (map[string]MCPServerEntry, error) {
	policy := answers.MCPPolicy
	servers := make(map[string]MCPServerEntry)

	if names := policy.Filter(answers.MCPServers); len(names) > 0 {
		cat, err := catalog.Default()
		if err != nil {
			return nil, fmt.Errorf("loading catalog for MCP server definitions: %w", err)
		}
		installed := installedMCPServers(answers.ProjectRoot)
		for _, name := range names {
			def, ok := cat.MCPServer(name)
			if !ok {
				return nil, fmt.Errorf("unknown MCP server %q: must be one of %s", name, mcpServerNameList(cat))
			}
			servers[name] = catalogServerEntry(name, def, installed)
		}
	}

	for _, srv := range cfg.MCPServers {
		if policy.Permits(srv.Name) {
			servers[srv.Name] = MCPServerEntry{
				Command: srv.Command,
				Args:    srv.Args,
				Env:     srv.Env,
			}
		}
	}
	return servers, nil
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
