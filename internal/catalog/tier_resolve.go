package catalog

import "fmt"

// ResolvedTier holds a fully-resolved tier definition with all
// inherited properties merged.
type ResolvedTier struct {
	Name                    string
	Order                   int
	Description             string
	DefaultPermissionPreset string
	Security                SecurityConfig
	Tools                   ToolsConfig
	ClaudeCode              ClaudeCodeConfig
}

// ResolveTier resolves a tier by name, recursively applying inheritance.
func (c *Catalog) ResolveTier(name string) (*ResolvedTier, error) {
	return c.resolveTier(name, make(map[string]bool))
}

func (c *Catalog) resolveTier(name string, visited map[string]bool) (*ResolvedTier, error) {
	if visited[name] {
		return nil, fmt.Errorf("circular inheritance detected at tier %q", name)
	}
	visited[name] = true

	def, ok := c.tiers.Tiers[name]
	if !ok {
		return nil, fmt.Errorf("unknown tier %q", name)
	}

	if def.Inherits == "" {
		return tierDefToResolved(name, def), nil
	}

	parent, err := c.resolveTier(def.Inherits, visited)
	if err != nil {
		return nil, fmt.Errorf("resolving parent of %q: %w", name, err)
	}

	return mergeTierIntoParent(parent, name, def), nil
}

func tierDefToResolved(name string, def TierDef) *ResolvedTier {
	r := &ResolvedTier{
		Name:                    name,
		Order:                   def.Order,
		Description:             def.Description,
		DefaultPermissionPreset: def.DefaultPermissionPreset,
	}

	if def.Security != nil {
		r.Security = *def.Security
	}
	if def.Tools != nil {
		r.Tools = *def.Tools
	}
	if def.ClaudeCode != nil {
		r.ClaudeCode = *def.ClaudeCode
	}

	return r
}

func mergeTierIntoParent(parent *ResolvedTier, name string, child TierDef) *ResolvedTier {
	r := &ResolvedTier{
		Name:                    name,
		Order:                   child.Order,
		Description:             child.Description,
		DefaultPermissionPreset: parent.DefaultPermissionPreset,
		Security:                parent.Security,
		Tools:                   parent.Tools,
		ClaudeCode:              parent.ClaudeCode,
	}

	if child.DefaultPermissionPreset != "" {
		r.DefaultPermissionPreset = child.DefaultPermissionPreset
	}

	if child.Security != nil {
		if child.Security.Level != "" {
			r.Security.Level = child.Security.Level
		}
		if child.Security.AgeGating != nil {
			r.Security.AgeGating = child.Security.AgeGating
		}
		if child.Security.ScriptBlocking != nil {
			r.Security.ScriptBlocking = child.Security.ScriptBlocking
		}
		if child.Security.LockEnforcement != nil {
			r.Security.LockEnforcement = child.Security.LockEnforcement
		}
		if child.Security.VulnScanning != nil {
			r.Security.VulnScanning = child.Security.VulnScanning
		}
	}

	if child.Tools != nil {
		seen := make(map[string]bool)
		var merged []string
		for _, t := range parent.Tools.Enabled {
			if !seen[t] {
				seen[t] = true
				merged = append(merged, t)
			}
		}
		for _, t := range child.Tools.Enabled {
			if !seen[t] {
				seen[t] = true
				merged = append(merged, t)
			}
		}
		r.Tools.Enabled = merged
	}

	if child.ClaudeCode != nil {
		if child.ClaudeCode.Enabled != nil {
			r.ClaudeCode.Enabled = child.ClaudeCode.Enabled
		}
		if child.ClaudeCode.PermissionLevel != "" {
			r.ClaudeCode.PermissionLevel = child.ClaudeCode.PermissionLevel
		}
		if len(child.ClaudeCode.MCPServers) > 0 {
			seen := make(map[string]bool)
			var merged []string
			for _, s := range parent.ClaudeCode.MCPServers {
				if !seen[s] {
					seen[s] = true
					merged = append(merged, s)
				}
			}
			for _, s := range child.ClaudeCode.MCPServers {
				if !seen[s] {
					seen[s] = true
					merged = append(merged, s)
				}
			}
			r.ClaudeCode.MCPServers = merged
		}
	}

	return r
}
