package toolreg

import (
	"errors"
	"fmt"
	"sort"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// BuildFromCatalogE creates a Registry pre-loaded with tool metadata from
// the YAML catalog. Returns an error if the catalog cannot be loaded or a
// tool definition is invalid.
func BuildFromCatalogE() (*Registry, error) {
	cat, err := catalog.Default()
	if err != nil {
		return nil, fmt.Errorf("loading catalog: %w", err)
	}
	return buildRegistryFromCatalog(cat)
}

// BuildFromCatalog creates a Registry pre-loaded with tool metadata from
// the YAML catalog. Tools get declarative fields (name, display name,
// category, description, default policy, owned files) from YAML. Behavioral
// functions (EnableFunc, etc.) are attached later via AttachBehavior. It
// panics if the catalog cannot be loaded or a tool definition is invalid.
func BuildFromCatalog() *Registry {
	r, err := buildRegistryFromCatalog(catalog.MustDefault())
	if err != nil {
		panic(fmt.Sprintf("toolreg: %v", err))
	}
	return r
}

// buildRegistryFromCatalog converts every catalog tool definition into a
// registry Tool. catalog.Validate has already checked the catalog's own
// invariants (ownership, default_policy, owned paths, tool references); a
// value this conversion does not recognize is still an error rather than a
// silent default, so an unrecognized ownership can never turn a shared file
// into an exclusive one that disable would delete.
func buildRegistryFromCatalog(cat *catalog.Catalog) (*Registry, error) {
	defs := cat.Tools()
	names := make([]string, 0, len(defs))
	for name := range defs {
		names = append(names, name)
	}
	sort.Strings(names)

	r := NewRegistry()
	var errs []error
	for _, name := range names {
		t, err := toolFromDef(name, defs[name])
		if err == nil {
			err = r.Register(t)
		}
		if err != nil {
			errs = append(errs, err)
		}
	}
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("invalid tool catalog: %w", err)
	}
	return r, nil
}

// toolFromDef converts one catalog tool definition into a Tool, attaching the
// declarative enable/disable behavior its fields describe.
func toolFromDef(name string, def catalog.ToolDef) (Tool, error) {
	policy, err := parseDefaultPolicy(def.DefaultPolicy)
	if err != nil {
		return Tool{}, fmt.Errorf("tool %q: %w", name, err)
	}
	owned, err := convertOwnedFiles(def.OwnedFiles)
	if err != nil {
		return Tool{}, fmt.Errorf("tool %q: %w", name, err)
	}

	t := Tool{
		Name:          name,
		DisplayName:   def.DisplayName,
		Category:      ToolCategory(def.Category),
		Description:   def.Description,
		Default:       policy,
		Prerequisites: def.Prerequisites,
		Conflicts:     def.Conflicts,
		OwnedFiles:    owned,
	}

	// A tool may declare several behavior sources (agent-postmortem is both
	// a toggle and an MCP server); every one of them applies.
	var enables, disables []func(*types.WizardAnswers)
	if def.MCPServerName != "" {
		enables = append(enables, mcpEnableFunc(def.MCPServerName))
		disables = append(disables, mcpDisableFunc(def.MCPServerName))
	}
	if def.SkillName != "" {
		enables = append(enables, skillEnableFunc(def.SkillName))
		disables = append(disables, skillDisableFunc(def.SkillName))
	}
	if def.ToggleField != "" {
		if _, ok := toggleFields[def.ToggleField]; !ok {
			return Tool{}, fmt.Errorf("tool %q: unknown toggle_field %q", name, def.ToggleField)
		}
		enables = append(enables, toggleEnableFunc(def.ToggleField))
		disables = append(disables, toggleDisableFunc(def.ToggleField))
	}
	t.EnableFunc = chainAnswerFuncs(enables)
	t.DisableFunc = chainAnswerFuncs(disables)

	// Auto-populate SharedContent from catalog section_content values.
	// Tools with dynamic templates override these via AttachBehavior.
	for _, o := range t.OwnedFiles {
		if o.Ownership == Shared && o.SectionID != "" && o.SectionContent != "" {
			if t.SharedContent == nil {
				t.SharedContent = make(map[SharedSection]SharedContentFunc)
			}
			content := o.SectionContent
			t.SharedContent[SectionOf(o)] = func(_ types.WizardAnswers) ([]byte, error) {
				return []byte(content), nil
			}
		}
	}

	return t, nil
}

// chainAnswerFuncs combines answer mutators into one that applies each in
// order. It returns nil when there are none.
func chainAnswerFuncs(fns []func(*types.WizardAnswers)) func(*types.WizardAnswers) {
	switch len(fns) {
	case 0:
		return nil
	case 1:
		return fns[0]
	}
	return func(a *types.WizardAnswers) {
		for _, fn := range fns {
			fn(a)
		}
	}
}

// parseDefaultPolicy maps a catalog default_policy to a DefaultPolicy. An
// omitted policy means opt-in, the least-enabling choice; an unrecognized
// one is an error rather than a silent opt-in.
func parseDefaultPolicy(s string) (DefaultPolicy, error) {
	switch s {
	case "always-on":
		return AlwaysOn, nil
	case "on-when-detected":
		return OnWhenDetected, nil
	case "opt-in", "":
		return OptIn, nil
	case "always-off":
		return AlwaysOff, nil
	default:
		return OptIn, fmt.Errorf("unknown default_policy %q (want always-on, on-when-detected, opt-in or always-off)", s)
	}
}

func convertOwnedFiles(defs []catalog.ToolOwnedFileDef) ([]FileOwnership, error) {
	if len(defs) == 0 {
		return nil, nil
	}
	result := make([]FileOwnership, len(defs))
	for i, d := range defs {
		var ownership OwnershipType
		switch d.Ownership {
		case "exclusive":
			ownership = Exclusive
		case "shared":
			ownership = Shared
		default:
			return nil, fmt.Errorf("owned file %q: unknown ownership %q (want shared or exclusive)", d.Path, d.Ownership)
		}
		result[i] = FileOwnership{
			Path:           d.Path,
			Ownership:      ownership,
			SectionID:      d.SectionID,
			SectionContent: d.SectionContent,
		}
	}
	return result, nil
}
