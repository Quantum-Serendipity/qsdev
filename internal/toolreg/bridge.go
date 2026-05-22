package toolreg

import "github.com/Quantum-Serendipity/qsdev/internal/catalog"

// BuildFromCatalog creates a Registry pre-loaded with tool metadata from
// the YAML catalog. Tools get declarative fields (name, display name,
// category, description, default policy, owned files) from YAML. Behavioral
// functions (EnableFunc, etc.) are attached later via AttachBehavior.
func BuildFromCatalog() *Registry {
	r := &Registry{
		tools: make(map[string]*Tool),
	}

	cat := catalog.Default()
	for name, def := range cat.Tools() {
		t := Tool{
			Name:          name,
			DisplayName:   def.DisplayName,
			Category:      ToolCategory(def.Category),
			Description:   def.Description,
			Default:       parseDefaultPolicy(def.DefaultPolicy),
			Prerequisites: def.Prerequisites,
			Conflicts:     def.Conflicts,
			OwnedFiles:    convertOwnedFiles(def.OwnedFiles),
		}

		if def.MCPServerName != "" {
			t.EnableFunc = mcpEnableFunc(def.MCPServerName)
			t.DisableFunc = mcpDisableFunc(def.MCPServerName)
		}
		if def.SkillName != "" {
			t.EnableFunc = skillEnableFunc(def.SkillName)
			t.DisableFunc = skillDisableFunc(def.SkillName)
		}
		if def.ToggleField != "" {
			t.EnableFunc = toggleEnableFunc(def.ToggleField)
			t.DisableFunc = toggleDisableFunc(def.ToggleField)
		}

		r.tools[name] = &t
	}

	return r
}

func parseDefaultPolicy(s string) DefaultPolicy {
	switch s {
	case "always-on":
		return AlwaysOn
	case "on-when-detected":
		return OnWhenDetected
	case "opt-in":
		return OptIn
	case "always-off":
		return AlwaysOff
	default:
		return OptIn
	}
}

func convertOwnedFiles(defs []catalog.ToolOwnedFileDef) []FileOwnership {
	if len(defs) == 0 {
		return nil
	}
	result := make([]FileOwnership, len(defs))
	for i, d := range defs {
		ownership := Exclusive
		if d.Ownership == "shared" {
			ownership = Shared
		}
		result[i] = FileOwnership{
			Path:           d.Path,
			Ownership:      ownership,
			SectionID:      d.SectionID,
			SectionContent: d.SectionContent,
		}
	}
	return result
}
