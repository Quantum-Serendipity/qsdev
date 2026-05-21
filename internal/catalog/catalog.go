package catalog

import (
	"fmt"
	"sync"
)

// Catalog holds all loaded configuration data. It is populated once
// at startup and is immutable thereafter.
type Catalog struct {
	tiers           TiersFile
	compliance      ComplianceFile
	profiles        ProfilesFile
	projectProfiles ProjectProfilesFile
	tools           ToolsFile
	security        SecurityFile
	hookTiers       HookTiersFile
	derivations     DerivationsFile
	validation      ValidationFile
}

var (
	defaultOnce sync.Once
	defaultCat  *Catalog
)

// Default returns the lazily-initialized global catalog loaded from
// embedded defaults. It panics if loading fails.
func Default() *Catalog {
	defaultOnce.Do(func() {
		var err error
		defaultCat, err = Load()
		if err != nil {
			panic(fmt.Sprintf("catalog: failed to load: %v", err))
		}
	})
	return defaultCat
}

// ResetDefault clears the cached default catalog, forcing the next
// call to Default() to reload. Intended for testing only.
func ResetDefault() {
	defaultOnce = sync.Once{}
	defaultCat = nil
}

// --- Tier accessors ---

// TierOrder returns the tier names sorted by ascending order.
func (c *Catalog) TierOrder() []string {
	type kv struct {
		name  string
		order int
	}
	items := make([]kv, 0, len(c.tiers.Tiers))
	for name, def := range c.tiers.Tiers {
		items = append(items, kv{name, def.Order})
	}
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].order < items[j-1].order; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
	result := make([]string, len(items))
	for i, item := range items {
		result[i] = item.name
	}
	return result
}

// TierDefs returns a copy of all tier definitions.
func (c *Catalog) TierDefs() map[string]TierDef {
	out := make(map[string]TierDef, len(c.tiers.Tiers))
	for k, v := range c.tiers.Tiers {
		out[k] = v
	}
	return out
}

// TierDef returns the definition for a named tier.
func (c *Catalog) TierDef(name string) (TierDef, bool) {
	d, ok := c.tiers.Tiers[name]
	return d, ok
}

// --- Compliance accessors ---

// ComplianceLevels returns a copy of all compliance level definitions.
func (c *Catalog) ComplianceLevels() map[string]ComplianceLevelDef {
	out := make(map[string]ComplianceLevelDef, len(c.compliance.Levels))
	for k, v := range c.compliance.Levels {
		out[k] = v
	}
	return out
}

// ComplianceLevel returns the definition for a named compliance level.
func (c *Catalog) ComplianceLevel(name string) (ComplianceLevelDef, bool) {
	d, ok := c.compliance.Levels[name]
	return d, ok
}

// --- Profile accessors ---

// Profiles returns a copy of all tier-based profiles.
func (c *Catalog) Profiles() map[string]ProfileDef {
	out := make(map[string]ProfileDef, len(c.profiles.Profiles))
	for k, v := range c.profiles.Profiles {
		out[k] = v
	}
	return out
}

// Profile returns the definition for a named profile.
func (c *Catalog) Profile(name string) (ProfileDef, bool) {
	d, ok := c.profiles.Profiles[name]
	return d, ok
}

// ProfileAliases returns a copy of the profile alias map.
func (c *Catalog) ProfileAliases() map[string]string {
	out := make(map[string]string, len(c.profiles.Aliases))
	for k, v := range c.profiles.Aliases {
		out[k] = v
	}
	return out
}

// --- Project profile accessors ---

// ProjectProfiles returns a copy of all project-type profiles.
func (c *Catalog) ProjectProfiles() map[string]ProjectProfileDef {
	out := make(map[string]ProjectProfileDef, len(c.projectProfiles.Profiles))
	for k, v := range c.projectProfiles.Profiles {
		out[k] = v
	}
	return out
}

// ProjectProfile returns a named project-type profile.
func (c *Catalog) ProjectProfile(name string) (ProjectProfileDef, bool) {
	d, ok := c.projectProfiles.Profiles[name]
	return d, ok
}

// --- Tool accessors ---

// Tools returns a copy of all tool definitions.
func (c *Catalog) Tools() map[string]ToolDef {
	out := make(map[string]ToolDef, len(c.tools.Tools))
	for k, v := range c.tools.Tools {
		out[k] = v
	}
	return out
}

// Tool returns the definition for a named tool.
func (c *Catalog) Tool(name string) (ToolDef, bool) {
	d, ok := c.tools.Tools[name]
	return d, ok
}

// --- Security accessors ---

// SecurityHooks returns the default security hook names.
func (c *Catalog) SecurityHooks() []string {
	out := make([]string, len(c.security.Hooks.Default))
	copy(out, c.security.Hooks.Default)
	return out
}

// BasePackages returns the default base package names.
func (c *Catalog) BasePackages() []string {
	out := make([]string, len(c.security.BasePackages))
	copy(out, c.security.BasePackages)
	return out
}

// UnsetVars returns the credential env vars to strip from the shell.
func (c *Catalog) UnsetVars() []string {
	out := make([]string, len(c.security.CleanEnvironment.UnsetVars))
	copy(out, c.security.CleanEnvironment.UnsetVars)
	return out
}

// KeepVars returns the env vars to preserve in clean mode.
func (c *Catalog) KeepVars() []string {
	out := make([]string, len(c.security.CleanEnvironment.KeepVars))
	copy(out, c.security.CleanEnvironment.KeepVars)
	return out
}

// CustomHooks returns the custom hook definitions.
func (c *Catalog) CustomHooks() []CustomHookDef {
	out := make([]CustomHookDef, len(c.security.CustomHooks))
	copy(out, c.security.CustomHooks)
	return out
}

// --- Hook tier accessors ---

// HookTierOrder returns the hook tier names in order.
func (c *Catalog) HookTierOrder() []string {
	out := make([]string, len(c.hookTiers.TierOrder))
	copy(out, c.hookTiers.TierOrder)
	return out
}

// HookTiers returns a copy of the hook tier membership map.
func (c *Catalog) HookTiers() map[string][]string {
	out := make(map[string][]string, len(c.hookTiers.Tiers))
	for k, v := range c.hookTiers.Tiers {
		cp := make([]string, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}

// --- Derivation accessors ---

// TierToCompliance returns the tier→compliance level mapping.
func (c *Catalog) TierToCompliance() map[string]string {
	out := make(map[string]string, len(c.derivations.TierToCompliance))
	for k, v := range c.derivations.TierToCompliance {
		out[k] = v
	}
	return out
}

// TierToEnabledTools returns the tier→enabled tools mapping.
func (c *Catalog) TierToEnabledTools() map[string][]string {
	out := make(map[string][]string, len(c.derivations.TierToEnabledTools))
	for k, v := range c.derivations.TierToEnabledTools {
		cp := make([]string, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}

// DefaultMCPServers returns the default MCP server names.
func (c *Catalog) DefaultMCPServers() []string {
	out := make([]string, len(c.derivations.DefaultMCPServers))
	copy(out, c.derivations.DefaultMCPServers)
	return out
}

// DefaultAgentToolConfig returns the default agent tool settings.
func (c *Catalog) DefaultAgentToolConfig() DefaultAgentTools {
	return c.derivations.DefaultAgentTools
}

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
