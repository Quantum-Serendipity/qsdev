package catalog

import (
	"maps"
	"slices"
)

// --- MCP Server accessors ---

// MCPServers returns a copy of all MCP server definitions.
func (c *Catalog) MCPServers() map[string]MCPServerDef {
	out := make(map[string]MCPServerDef, len(c.mcpServers))
	maps.Copy(out, c.mcpServers)
	return out
}

// MCPServer returns the definition for a named MCP server.
func (c *Catalog) MCPServer(name string) (MCPServerDef, bool) {
	d, ok := c.mcpServers[name]
	return d, ok
}

// MCPServerNames returns all MCP server names sorted alphabetically.
func (c *Catalog) MCPServerNames() []string {
	names := make([]string, 0, len(c.mcpServers))
	for k := range c.mcpServers {
		names = append(names, k)
	}
	slices.Sort(names)
	return names
}

// DefaultMCPServers returns the default MCP server names.
func (c *Catalog) DefaultMCPServers() []string {
	out := make([]string, len(c.derivations.DefaultMCPServers))
	copy(out, c.derivations.DefaultMCPServers)
	return out
}

// AlwaysOnMCPServers returns, sorted, the MCP servers that back the
// catalog's always-on tools (their mcp_server_name): enforcement adds each to
// every configuration the tool applies to, as a default init does.
func (c *Catalog) AlwaysOnMCPServers() []string {
	var out []string
	for _, def := range c.Tools() {
		if def.DefaultPolicy == "always-on" && def.MCPServerName != "" {
			out = append(out, def.MCPServerName)
		}
	}
	slices.Sort(out)
	return slices.Compact(out)
}

// MCPServeOptIns is the mcp_serve section of the user's org overlay: the
// operator's opt-ins for the MCP server tools that start processes
// (qsdev_nix_run) or hand out credentials (qsdev_credential_vend). The
// embedded defaults leave both off, and a project defaults file cannot set
// the section, so a repository can never opt itself in. Read it from
// LoadUserScope.
type MCPServeOptIns struct {
	AllowNixRun         bool `yaml:"allow_nix_run"`
	AllowCredentialVend bool `yaml:"allow_credential_vend"`
}

// MCPServeOptIns returns the catalog's mcp_serve opt-ins.
func (c *Catalog) MCPServeOptIns() MCPServeOptIns {
	return c.mcpServe
}

// --- Bootstrap tool accessors ---

// BootstrapToolClaudeCode is the bootstrap_tools entry that pins the Claude
// Code release the machine bootstrap and `devenv setup` install.
const BootstrapToolClaudeCode = "claude-code"

// BootstrapToolDevenv and BootstrapToolDirenv are the bootstrap_tools
// entries that pin the devenv and direnv the machine bootstrap and `devenv
// setup` install with `nix profile install`.
const (
	BootstrapToolDevenv = "devenv"
	BootstrapToolDirenv = "direnv"
)

// BootstrapTool returns the pinned install definition of a tool the qsdev
// bootstrap installs, such as "claude-code".
func (c *Catalog) BootstrapTool(name string) (BootstrapToolDef, bool) {
	d, ok := c.bootstrapTools[name]
	return d, ok
}
