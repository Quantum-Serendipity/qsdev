package catalog

import "maps"

// --- Tool accessors ---

// Tools returns a copy of all tool definitions.
func (c *Catalog) Tools() map[string]ToolDef {
	out := make(map[string]ToolDef, len(c.tools.Tools))
	maps.Copy(out, c.tools.Tools)
	return out
}

// Tool returns the definition for a named tool.
func (c *Catalog) Tool(name string) (ToolDef, bool) {
	d, ok := c.tools.Tools[name]
	return d, ok
}

// ToolNixPackages returns a map of tool name to Nix package attribute
// for tools that declare a nix_package field.
func (c *Catalog) ToolNixPackages() map[string]string {
	out := make(map[string]string)
	for name, def := range c.tools.Tools {
		if def.NixPackage != "" {
			out[name] = def.NixPackage
		}
	}
	return out
}

// ToolNixExprs returns a map of tool name to raw Nix expression
// for tools that declare a nix_expr field.
func (c *Catalog) ToolNixExprs() map[string]string {
	out := make(map[string]string)
	for name, def := range c.tools.Tools {
		if def.NixExpr != "" {
			out[name] = def.NixExpr
		}
	}
	return out
}
