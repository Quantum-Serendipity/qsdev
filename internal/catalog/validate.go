package catalog

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strings"
)

// maxInheritanceDepth is the maximum allowed tier inheritance chain length.
const maxInheritanceDepth = 10

// CatalogError describes a validation problem in the loaded catalog.
type CatalogError struct {
	File    string
	Field   string
	Message string
}

func (e CatalogError) Error() string {
	return fmt.Sprintf("%s: %s: %s", e.File, e.Field, e.Message)
}

// Validate checks cross-references and invariants across all loaded
// catalog data. It returns all errors found rather than stopping at
// the first.
func (c *Catalog) Validate() []CatalogError {
	var errs []CatalogError

	errs = append(errs, c.validateTiers()...)
	errs = append(errs, c.validateDerivations()...)
	errs = append(errs, c.validateHookTiers()...)
	errs = append(errs, c.validateProjectProfiles()...)
	errs = append(errs, c.validateTierOrders()...)
	errs = append(errs, c.validatePermissionSets()...)
	errs = append(errs, c.validateMCPServerRefs()...)
	errs = append(errs, c.validatePresetRefs()...)
	errs = append(errs, c.validateComplianceHooks()...)
	errs = append(errs, c.validateTools()...)

	return errs
}

func (c *Catalog) validateTiers() []CatalogError {
	var errs []CatalogError

	for name, def := range c.tiers.Tiers {
		if def.Description == "" {
			errs = append(errs, CatalogError{"tiers.yaml", name, "missing description"})
		}
		if def.Order == 0 {
			errs = append(errs, CatalogError{"tiers.yaml", name, "order must be > 0"})
		}
		if def.Inherits != "" {
			if _, ok := c.tiers.Tiers[def.Inherits]; !ok {
				errs = append(errs, CatalogError{
					"tiers.yaml", name,
					fmt.Sprintf("inherits unknown tier %q", def.Inherits),
				})
			}
		}
	}

	// Check for inheritance cycles.
	for name := range c.tiers.Tiers {
		if err := c.checkTierCycle(name); err != nil {
			errs = append(errs, CatalogError{"tiers.yaml", name, err.Error()})
		}
	}

	return errs
}

func (c *Catalog) checkTierCycle(start string) error {
	visited := make(map[string]bool)
	current := start
	for i := 0; i < maxInheritanceDepth; i++ {
		def, ok := c.tiers.Tiers[current]
		if !ok || def.Inherits == "" {
			return nil
		}
		if visited[def.Inherits] {
			return fmt.Errorf("circular inheritance detected")
		}
		visited[current] = true
		current = def.Inherits
	}
	return fmt.Errorf("inheritance chain exceeds maximum depth of %d", maxInheritanceDepth)
}

func (c *Catalog) validateDerivations() []CatalogError {
	var errs []CatalogError

	if _, ok := c.tiers.Tiers[c.derivations.DefaultTier]; !ok {
		errs = append(errs, CatalogError{
			"derivations.yaml", "default_tier",
			fmt.Sprintf("references unknown tier %q", c.derivations.DefaultTier),
		})
	}

	for tierName, complianceName := range c.derivations.TierToCompliance {
		if _, ok := c.tiers.Tiers[tierName]; !ok {
			errs = append(errs, CatalogError{
				"derivations.yaml", "tier_to_compliance",
				fmt.Sprintf("references unknown tier %q", tierName),
			})
		}
		if _, ok := c.compliance.Levels[complianceName]; !ok {
			errs = append(errs, CatalogError{
				"derivations.yaml", "tier_to_compliance",
				fmt.Sprintf("tier %q maps to unknown compliance level %q", tierName, complianceName),
			})
		}
	}

	for tierName, tools := range c.derivations.TierToEnabledTools {
		if _, ok := c.tiers.Tiers[tierName]; !ok {
			errs = append(errs, CatalogError{
				"derivations.yaml", "tier_to_enabled_tools",
				fmt.Sprintf("references unknown tier %q", tierName),
			})
		}
		for _, toolName := range tools {
			if _, ok := c.tools.Tools[toolName]; !ok {
				errs = append(errs, CatalogError{
					"derivations.yaml", "tier_to_enabled_tools",
					fmt.Sprintf("tier %q references unknown tool %q", tierName, toolName),
				})
			}
		}
	}

	return errs
}

func (c *Catalog) validateHookTiers() []CatalogError {
	var errs []CatalogError

	for _, name := range c.hookTiers.TierOrder {
		if _, ok := c.hookTiers.Tiers[name]; !ok {
			errs = append(errs, CatalogError{
				"hook_tiers.yaml", "tier_order",
				fmt.Sprintf("tier %q listed in order but not defined", name),
			})
		}
	}

	return errs
}

func (c *Catalog) validateProjectProfiles() []CatalogError {
	var errs []CatalogError

	for name, def := range c.projectProfiles.Profiles {
		if def.Tier != "" {
			if _, ok := c.tiers.Tiers[def.Tier]; !ok {
				errs = append(errs, CatalogError{
					"project_profiles.yaml", name,
					fmt.Sprintf("references unknown tier %q", def.Tier),
				})
			}
		}
	}

	return errs
}

// validateTierOrders checks that no two tiers share an order. The order is a
// tier's numeric level, so a duplicate would make the tier a level maps back
// to ambiguous.
func (c *Catalog) validateTierOrders() []CatalogError {
	var errs []CatalogError

	byOrder := make(map[int][]string)
	for name, def := range c.tiers.Tiers {
		byOrder[def.Order] = append(byOrder[def.Order], name)
	}
	for _, order := range slices.Sorted(maps.Keys(byOrder)) {
		names := byOrder[order]
		if len(names) < 2 {
			continue
		}
		slices.Sort(names)
		errs = append(errs, CatalogError{
			"tiers.yaml", strings.Join(names, ","),
			fmt.Sprintf("tiers share order %d; tier orders must be unique", order),
		})
	}

	return errs
}

// validatePermissionSets checks that every permission set name referenced by
// the deny/ask set lists and the preset definitions exists. An unknown set
// name would otherwise silently contribute no rules at all.
func (c *Catalog) validatePermissionSets() []CatalogError {
	var errs []CatalogError
	rules := c.permissionRules

	check := func(field string, names []string, sets map[string][]string, kind string) {
		for _, name := range names {
			if _, ok := sets[name]; !ok {
				errs = append(errs, CatalogError{
					"permission_rules", field,
					fmt.Sprintf("references unknown %s set %q", kind, name),
				})
			}
		}
	}

	check("permission_all_deny_sets", rules.AllDenySets, rules.DenyRules, "deny")
	check("permission_supply_chain_deny_sets", rules.SupplyChainDenySets, rules.DenyRules, "deny")
	check("permission_package_ask_sets", rules.PackageAskSets, rules.AskRules, "ask")
	for _, name := range slices.Sorted(maps.Keys(rules.PresetDefs)) {
		def := rules.PresetDefs[name]
		field := "permission_preset_defs." + name
		check(field+".allow_sets", def.AllowSets, rules.AllowRules, "allow")
		check(field+".deny_sets", def.DenySets, rules.DenyRules, "deny")
		check(field+".ask_sets", def.AskSets, rules.AskRules, "ask")
	}

	return errs
}

// validateMCPServerRefs checks that MCP servers referenced by tiers and
// default_mcp_servers are defined in mcp_servers.
func (c *Catalog) validateMCPServerRefs() []CatalogError {
	var errs []CatalogError

	check := func(file, field string, names []string) {
		for _, name := range names {
			if _, ok := c.mcpServers[name]; !ok {
				errs = append(errs, CatalogError{file, field, fmt.Sprintf("references unknown MCP server %q", name)})
			}
		}
	}

	for name, def := range c.tiers.Tiers {
		if def.ClaudeCode != nil {
			check("tiers.yaml", name+".claude_code.mcp_servers", def.ClaudeCode.MCPServers)
		}
	}
	check("derivations.yaml", "default_mcp_servers", c.derivations.DefaultMCPServers)

	return errs
}

// validatePresetRefs checks that every permission level referenced by tiers,
// project profiles and compliance levels is a valid permission
// preset, and that every preset definition is a listed preset.
func (c *Catalog) validatePresetRefs() []CatalogError {
	var errs []CatalogError

	valid := make(map[string]bool, len(c.validation.PermissionPresets))
	for _, p := range c.validation.PermissionPresets {
		valid[p] = true
	}
	check := func(file, field, preset string) {
		if preset != "" && !valid[preset] {
			errs = append(errs, CatalogError{file, field, fmt.Sprintf("references unknown permission preset %q", preset)})
		}
	}

	for name, def := range c.tiers.Tiers {
		check("tiers.yaml", name+".default_permission_preset", def.DefaultPermissionPreset)
		if def.ClaudeCode != nil {
			check("tiers.yaml", name+".claude_code.permission_level", def.ClaudeCode.PermissionLevel)
		}
	}
	for name, def := range c.projectProfiles.Profiles {
		check("project_profiles.yaml", name+".permission_level", def.PermissionLevel)
	}
	for name, def := range c.compliance.Levels {
		check("compliance.yaml", name+".claude_permission_level", def.ClaudePermissionLevel)
	}
	for name := range c.permissionRules.PresetDefs {
		if !valid[name] {
			errs = append(errs, CatalogError{
				"permission_rules", "permission_preset_defs." + name,
				"preset is not listed in permission_presets",
			})
		}
	}
	errs = append(errs, c.validatePresetStrictness()...)

	return errs
}

// validatePresetStrictness checks that strictness ranks are non-negative and
// that no two presets share one, so tightening a permission level is never
// ambiguous.
func (c *Catalog) validatePresetStrictness() []CatalogError {
	var errs []CatalogError
	byRank := make(map[int]string)
	for _, name := range slices.Sorted(maps.Keys(c.permissionRules.PresetDefs)) {
		rank := c.permissionRules.PresetDefs[name].Strictness
		field := "permission_preset_defs." + name + ".strictness"
		switch {
		case rank < 0:
			errs = append(errs, CatalogError{"permission_rules", field, "strictness must not be negative"})
		case rank == 0:
		case byRank[rank] != "":
			errs = append(errs, CatalogError{"permission_rules", field,
				fmt.Sprintf("strictness %d is also used by preset %q", rank, byRank[rank])})
		default:
			byRank[rank] = name
		}
	}
	return errs
}

// validateComplianceHooks checks that each compliance level's required
// pre-commit hooks name a known pre-commit hook (security_hooks, hook_tiers
// or custom_hooks) or a known tool.
func (c *Catalog) validateComplianceHooks() []CatalogError {
	var errs []CatalogError

	known := make(map[string]bool)
	for _, h := range c.security.Hooks.Default {
		known[h] = true
	}
	for _, hooks := range c.hookTiers.Tiers {
		for _, h := range hooks {
			known[h] = true
		}
	}
	for _, h := range c.security.CustomHooks {
		known[h.ID] = true
	}
	for name := range c.tools.Tools {
		known[name] = true
	}

	for level, def := range c.compliance.Levels {
		for _, hook := range def.RequiredPreCommitHooks {
			if !known[hook] {
				errs = append(errs, CatalogError{
					"compliance.yaml", level + ".required_pre_commit_hooks",
					fmt.Sprintf("references unknown hook or tool %q", hook),
				})
			}
		}
	}

	return errs
}

// validateTools checks tool definitions. Tools can come from user-editable
// org and project overlays, and the tool registry turns their owned files
// into what enable writes and disable deletes, so closed-set fields are
// rejected rather than silently defaulted (a mis-cased "Shared" must not
// become an exclusive file) and owned paths must stay inside the project.
func (c *Catalog) validateTools() []CatalogError {
	var errs []CatalogError

	for _, name := range slices.Sorted(maps.Keys(c.tools.Tools)) {
		def := c.tools.Tools[name]
		addErr := func(format string, args ...any) {
			errs = append(errs, CatalogError{"tools.yaml", name, fmt.Sprintf(format, args...)})
		}

		switch def.DefaultPolicy {
		case "", "always-on", "on-when-detected", "opt-in", "always-off":
		default:
			addErr("unknown default_policy %q (want always-on, on-when-detected, opt-in or always-off)", def.DefaultPolicy)
		}

		for _, f := range def.OwnedFiles {
			if !filepath.IsLocal(filepath.FromSlash(f.Path)) {
				addErr("owned file %q must be a relative path inside the project", f.Path)
			}
			switch f.Ownership {
			case "exclusive":
			case "shared":
				if f.SectionID == "" {
					addErr("shared owned file %q has no section_id", f.Path)
				}
			default:
				addErr("owned file %q: unknown ownership %q (want shared or exclusive)", f.Path, f.Ownership)
			}
		}

		for _, ref := range slices.Concat(def.Prerequisites, def.Conflicts) {
			if _, ok := c.tools.Tools[ref]; !ok {
				addErr("references unknown tool %q", ref)
			}
		}
	}

	return errs
}
