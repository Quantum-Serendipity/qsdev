package toolreg

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// BuildFromCatalogE creates a Registry pre-loaded with tool metadata from
// the YAML catalog. Returns an error if the catalog cannot be loaded.
func BuildFromCatalogE() (*Registry, error) {
	cat, err := catalog.Default()
	if err != nil {
		return nil, fmt.Errorf("loading catalog: %w", err)
	}
	return buildRegistryFromCatalog(cat), nil
}

// BuildFromCatalog creates a Registry pre-loaded with tool metadata from
// the YAML catalog. Tools get declarative fields (name, display name,
// category, description, default policy, owned files) from YAML. Behavioral
// functions (EnableFunc, etc.) are attached later via AttachBehavior.
func BuildFromCatalog() *Registry {
	cat := catalog.MustDefault()
	return buildRegistryFromCatalog(cat)
}

func buildRegistryFromCatalog(cat *catalog.Catalog) *Registry {
	r := NewRegistry()

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

		// Auto-populate SharedContent from catalog section_content values.
		// Tools with dynamic templates override these via AttachBehavior.
		for _, owned := range t.OwnedFiles {
			if owned.Ownership == Shared && owned.SectionID != "" && owned.SectionContent != "" {
				if t.SharedContent == nil {
					t.SharedContent = make(map[string]SharedContentFunc)
				}
				content := owned.SectionContent
				t.SharedContent[owned.SectionID] = func(_ types.WizardAnswers) ([]byte, error) {
					return []byte(content), nil
				}
			}
		}

		// Use Register which stores &t. Errors should not occur since
		// catalog tool names are unique, but ignore them defensively.
		_ = r.Register(t)
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
