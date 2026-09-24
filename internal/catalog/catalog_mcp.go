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

// --- Bootstrap tool accessors ---

// BootstrapToolClaudeCode is the bootstrap_tools entry that pins the Claude
// Code release the machine bootstrap and `devenv setup` install.
const BootstrapToolClaudeCode = "claude-code"

// BootstrapTool returns the pinned install definition of a tool the qsdev
// bootstrap installs, such as "claude-code".
func (c *Catalog) BootstrapTool(name string) (BootstrapToolDef, bool) {
	d, ok := c.bootstrapTools[name]
	return d, ok
}
