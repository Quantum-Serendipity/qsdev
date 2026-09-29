package catalog

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
)

// ErrProjectOverlayRejected reports a project defaults file
// (.qsdev/defaults.yaml) that tries to do more than add or tighten.
var ErrProjectOverlayRejected = errors.New("project defaults may only add or tighten")

// projectOverlaySections are the top-level keys a project defaults file may
// set (see applyProjectOverlay for what each may do). The file is committed
// to the repository, so anyone who can push to it — or the author of a
// repository the developer merely cloned — controls it. It is therefore
// limited to changes that can only make the generated configuration
// stricter; every other section (tools, MCP server commands, tiers,
// compliance level definitions, allow and ask rules, keep_vars, ...) is
// rejected.
var projectOverlaySections = []string{
	sectionPermissionDenyRules,
	sectionPermissionAllDenySets,
	sectionPermissionSupplyChainDenySets,
	sectionPermissionPresetDefs,
	sectionSecurityHooks,
	sectionCustomHooks,
	sectionHookTiers,
	sectionTierToCompliance,
}

// Top-level sections a project defaults file may set.
const (
	sectionPermissionDenyRules           = "permission_deny_rules"
	sectionPermissionAllDenySets         = "permission_all_deny_sets"
	sectionPermissionSupplyChainDenySets = "permission_supply_chain_deny_sets"
	sectionSecurityHooks                 = "security_hooks"
	sectionCustomHooks                   = "custom_hooks"
	sectionHookTiers                     = "hook_tiers"
	sectionTierToCompliance              = "tier_to_compliance"
)

// presetDenySetsField is the only preset field a project defaults file may set.
const presetDenySetsField = "deny_sets"

// rejectFunc records why part of a project defaults file is rejected.
type rejectFunc func(field, format string, args ...any)

// projectOverlay is a parsed project defaults file together with the
// top-level sections it sets.
type projectOverlay struct {
	path     string
	cat      *Catalog
	sections []string
}

// loadProjectOverlay reads and strictly parses a project defaults file. A
// missing file is reported with an error satisfying os.IsNotExist.
func loadProjectOverlay(path string) (*projectOverlay, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	cat, err := parseUnifiedBytes(data)
	if err != nil {
		return nil, fmt.Errorf("parsing project defaults %s: %w", path, err)
	}
	sections, err := setTopLevelSections(data)
	if err != nil {
		return nil, fmt.Errorf("parsing project defaults %s: %w", path, err)
	}
	return &projectOverlay{path: path, cat: cat, sections: sections}, nil
}

// setTopLevelSections returns the top-level keys of a defaults document that
// carry a value. Keys left empty (`tools:`) set nothing, and removed
// sections are dropped by parseUnifiedBytes, so neither is returned.
func setTopLevelSections(data []byte) ([]string, error) {
	var doc yaml.Node
	if err := yaml.NewDecoder(bytes.NewReader(data)).Decode(&doc); err != nil {
		if errors.Is(err, io.EOF) {
			return nil, nil
		}
		return nil, err
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, nil
	}
	root := doc.Content[0]
	var keys []string
	for i := 0; i+1 < len(root.Content); i += 2 {
		key, value := root.Content[i].Value, root.Content[i+1]
		if value.Tag == "!!null" || slices.Contains(removedSections, key) {
			continue
		}
		keys = append(keys, key)
	}
	return keys, nil
}

// applyProjectOverlay returns base with the project overlay's additions
// applied, or the reasons the overlay is rejected. A project defaults file
// may only add or tighten:
//
//   - permission_deny_rules: rules are added to a set (built-in rules are
//     never removed) and new sets may be defined;
//   - permission_all_deny_sets, permission_supply_chain_deny_sets: sets
//     are added to the lists;
//   - permission_preset_defs: an existing preset gains deny_sets; no other
//     preset field may be set and no preset may be defined;
//   - security_hooks: hooks are added to the always-on list;
//   - custom_hooks: new hooks are added; an id that already names a hook
//     or tool cannot be reused;
//   - hook_tiers: hooks are added to an existing hook tier;
//   - tier_to_compliance: a tier may move to a compliance level of equal or
//     higher order, never a lower one.
//
// base is not modified.
func applyProjectOverlay(base *Catalog, ov *projectOverlay) (*Catalog, []CatalogError) {
	var errs []CatalogError
	reject := rejectFunc(func(field, format string, args ...any) {
		errs = append(errs, CatalogError{ov.path, field, fmt.Sprintf(format, args...)})
	})
	for _, section := range ov.sections {
		if !slices.Contains(projectOverlaySections, section) {
			reject(section, "section cannot be set by a project defaults file (allowed: %s)",
				strings.Join(projectOverlaySections, ", "))
		}
	}

	// MergeCatalogs with an empty overlay copies base.
	out := MergeCatalogs(base, &Catalog{})
	proj := ov.cat

	rules := &out.permissionRules
	rules.DenyRules = unionStringSliceMap(rules.DenyRules, proj.permissionRules.DenyRules)
	rules.AllDenySets = unionStrings(rules.AllDenySets, proj.permissionRules.AllDenySets)
	rules.SupplyChainDenySets = unionStrings(rules.SupplyChainDenySets, proj.permissionRules.SupplyChainDenySets)
	addProjectPresetDenySets(rules, proj, reject)

	out.security.Hooks.Default = unionStrings(out.security.Hooks.Default, proj.security.Hooks.Default)
	out.security.CustomHooks = addProjectCustomHooks(base, proj.security.CustomHooks, reject)
	out.hookTiers.Tiers = addProjectHookTierMembers(out.hookTiers.Tiers, proj.hookTiers.Tiers, reject)
	out.derivations.TierToCompliance = raiseProjectTierCompliance(out, proj.derivations.TierToCompliance, reject)

	return out, errs
}

// addProjectPresetDenySets adds the overlay's deny_sets to existing
// presets. Setting any other preset field, or naming an unknown preset, is
// rejected.
func addProjectPresetDenySets(rules *PermissionRulesFile, proj *Catalog, reject rejectFunc) {
	if len(proj.permissionRules.PresetDefs) == 0 {
		return
	}
	nodes := proj.entryNodes[sectionPermissionPresetDefs]
	presets := make(map[string]PermissionPresetDef, len(rules.PresetDefs))
	maps.Copy(presets, rules.PresetDefs)
	for _, name := range slices.Sorted(maps.Keys(proj.permissionRules.PresetDefs)) {
		field := sectionPermissionPresetDefs + "." + name
		def, ok := presets[name]
		if !ok {
			reject(field, "a project defaults file cannot define a permission preset")
			continue
		}
		if extra := mappingKeysExcept(nodes[name], presetDenySetsField); len(extra) > 0 {
			reject(field, "only deny_sets can be added to a preset, not %s", strings.Join(extra, ", "))
			continue
		}
		def.DenySets = unionStrings(def.DenySets, proj.permissionRules.PresetDefs[name].DenySets)
		presets[name] = def
	}
	rules.PresetDefs = presets
}

// addProjectCustomHooks appends the overlay's custom hooks. A hook without
// an id, or whose id already names a hook or tool in base, is rejected:
// devenv.nix renders each custom hook as git-hooks.hooks.<id>, so reusing
// an id would let the project replace a built-in check (a custom hook, an
// always-on or tiered security hook, or a tool's hook such as
// commit-ticket) with a no-op.
func addProjectCustomHooks(base *Catalog, add []CustomHookDef, reject rejectFunc) []CustomHookDef {
	taken := knownHookIDs(base)
	out := slices.Clone(base.security.CustomHooks)
	for i, h := range add {
		field := fmt.Sprintf("%s[%d]", sectionCustomHooks, i)
		switch {
		case h.ID == "":
			reject(field, "custom hook has no id")
		case taken[h.ID]:
			reject(field, "custom hook %q is already defined as a hook or tool and cannot be redefined", h.ID)
		default:
			taken[h.ID] = true
			out = append(out, h)
		}
	}
	return out
}

// knownHookIDs returns every hook id c defines or enables: custom hooks,
// always-on security hooks, hook tier members, and tools (a tool's own
// pre-commit hook is named after it).
func knownHookIDs(c *Catalog) map[string]bool {
	ids := make(map[string]bool)
	for _, h := range c.security.CustomHooks {
		ids[h.ID] = true
	}
	for _, h := range c.security.Hooks.Default {
		ids[h] = true
	}
	for _, hooks := range c.hookTiers.Tiers {
		for _, h := range hooks {
			ids[h] = true
		}
	}
	for name := range c.tools.Tools {
		ids[name] = true
	}
	return ids
}

// addProjectHookTierMembers adds hooks to existing hook tiers. A new tier is
// rejected: hook_tier_order cannot change, so it would never take effect.
func addProjectHookTierMembers(base, add map[string][]string, reject rejectFunc) map[string][]string {
	out := mergeStringSliceMap(base, nil)
	for _, tier := range slices.Sorted(maps.Keys(add)) {
		if _, ok := out[tier]; !ok {
			reject(sectionHookTiers+"."+tier, "unknown hook tier; hooks can only be added to an existing tier")
			continue
		}
		out[tier] = unionStrings(out[tier], add[tier])
	}
	return out
}

// raiseProjectTierCompliance applies tier→compliance mappings that keep or
// raise a tier's compliance level (by the level's order). Lowering one, an
// unmapped tier or an unknown level is rejected.
func raiseProjectTierCompliance(base *Catalog, add map[string]string, reject rejectFunc) map[string]string {
	out := mergeStringMap(base.derivations.TierToCompliance, nil)
	for _, tier := range slices.Sorted(maps.Keys(add)) {
		field := sectionTierToCompliance + "." + tier
		current, ok := out[tier]
		if !ok {
			reject(field, "tier has no compliance level to raise")
			continue
		}
		want, ok := base.compliance.Levels[add[tier]]
		if !ok {
			reject(field, "unknown compliance level %q", add[tier])
			continue
		}
		if have := base.compliance.Levels[current]; want.Order < have.Order {
			reject(field, "cannot lower compliance from %q to %q", current, add[tier])
			continue
		}
		out[tier] = add[tier]
	}
	return out
}

// mappingKeysExcept returns the keys of a mapping node other than allowed.
func mappingKeysExcept(node *yaml.Node, allowed ...string) []string {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	var extra []string
	for i := 0; i+1 < len(node.Content); i += 2 {
		if key := node.Content[i].Value; !slices.Contains(allowed, key) {
			extra = append(extra, key)
		}
	}
	return extra
}

// unionStrings returns base followed by the entries of add it lacks.
func unionStrings(base, add []string) []string {
	if len(base) == 0 && len(add) == 0 {
		return nil
	}
	return sliceutil.Dedup(append(slices.Clone(base), add...))
}

// unionStringSliceMap returns base with each of add's lists unioned into
// the list of the same key.
func unionStringSliceMap(base, add map[string][]string) map[string][]string {
	out := mergeStringSliceMap(base, nil)
	for key, values := range add {
		if out == nil {
			out = make(map[string][]string, len(add))
		}
		out[key] = unionStrings(out[key], values)
	}
	return out
}
