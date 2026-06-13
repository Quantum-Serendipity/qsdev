package catalog

import "github.com/Quantum-Serendipity/qsdev/internal/sliceutil"

// --- Validation accessors ---

// Languages returns all supported language names.
func (c *Catalog) Languages() []string {
	out := make([]string, len(c.validation.Languages.All))
	copy(out, c.validation.Languages.All)
	return out
}

// CoreLanguages returns the core language names.
func (c *Catalog) CoreLanguages() []string {
	out := make([]string, len(c.validation.Languages.Core))
	copy(out, c.validation.Languages.Core)
	return out
}

// Services returns all supported service names.
func (c *Catalog) Services() []string {
	out := make([]string, len(c.validation.Services))
	copy(out, c.validation.Services)
	return out
}

// PermissionPresets returns the valid permission preset names.
func (c *Catalog) PermissionPresets() []string {
	out := make([]string, len(c.validation.PermissionPresets))
	copy(out, c.validation.PermissionPresets)
	return out
}

// HookPresets returns the valid hook preset names.
func (c *Catalog) HookPresets() []string {
	out := make([]string, len(c.validation.HookPresets))
	copy(out, c.validation.HookPresets)
	return out
}

// SecurityLevels returns the valid security level names.
func (c *Catalog) SecurityLevels() []string {
	out := make([]string, len(c.validation.SecurityLevels))
	copy(out, c.validation.SecurityLevels)
	return out
}

// DataClassifications returns the valid data classification names.
func (c *Catalog) DataClassifications() []string {
	out := make([]string, len(c.validation.DataClassifications))
	copy(out, c.validation.DataClassifications)
	return out
}

// PackageManagers returns the package manager names for an ecosystem.
func (c *Catalog) PackageManagers(ecosystem string) []string {
	pms, ok := c.validation.PackageManagers[ecosystem]
	if !ok {
		return nil
	}
	out := make([]string, len(pms))
	copy(out, pms)
	return out
}

// ToolCategories returns the tool category definitions.
func (c *Catalog) ToolCategories() []ToolCategoryDef {
	out := make([]ToolCategoryDef, len(c.validation.ToolCategories))
	copy(out, c.validation.ToolCategories)
	return out
}

// --- Permission rule accessors ---

// PermissionDenyRules returns the deny rules for a named set.
func (c *Catalog) PermissionDenyRules(setName string) []string {
	rules, ok := c.permissionRules.DenyRules[setName]
	if !ok {
		return nil
	}
	out := make([]string, len(rules))
	copy(out, rules)
	return out
}

// AllPermissionDenyRules returns all deny rules from all deny sets listed
// in permission_all_deny_sets, concatenated and deduplicated.
func (c *Catalog) AllPermissionDenyRules() []string {
	var rules []string
	for _, setName := range c.permissionRules.AllDenySets {
		rules = append(rules, c.PermissionDenyRules(setName)...)
	}
	return sliceutil.Dedup(rules)
}

// SupplyChainDenyRules returns deny rules from the supply chain deny sets.
func (c *Catalog) SupplyChainDenyRules() []string {
	var rules []string
	for _, setName := range c.permissionRules.SupplyChainDenySets {
		rules = append(rules, c.PermissionDenyRules(setName)...)
	}
	return sliceutil.Dedup(rules)
}

// PermissionAllowRules returns the allow rules for a named set.
func (c *Catalog) PermissionAllowRules(setName string) []string {
	rules, ok := c.permissionRules.AllowRules[setName]
	if !ok {
		return nil
	}
	out := make([]string, len(rules))
	copy(out, rules)
	return out
}

// PermissionAskRules returns the ask rules for a named set.
func (c *Catalog) PermissionAskRules(setName string) []string {
	rules, ok := c.permissionRules.AskRules[setName]
	if !ok {
		return nil
	}
	out := make([]string, len(rules))
	copy(out, rules)
	return out
}

// AllPackageInstallAskRules returns all package install ask rules.
func (c *Catalog) AllPackageInstallAskRules() []string {
	var rules []string
	for _, setName := range c.permissionRules.PackageAskSets {
		rules = append(rules, c.PermissionAskRules(setName)...)
	}
	return sliceutil.Dedup(rules)
}

// PermissionPreset returns the preset definition for a named preset.
func (c *Catalog) PermissionPreset(name string) (PermissionPresetDef, bool) {
	d, ok := c.permissionRules.PresetDefs[name]
	return d, ok
}
