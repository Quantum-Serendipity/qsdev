package catalog

// MergeCatalogs merges an overlay catalog into a base catalog.
// Non-empty overlay fields override or extend the base. For map fields,
// overlay entries are added or replace base entries with the same key.
// For slice fields, the overlay replaces the base if non-empty.
func MergeCatalogs(base, overlay *Catalog) *Catalog {
	result := &Catalog{}

	// Tiers: merge maps.
	result.tiers.Tiers = mergeMap(base.tiers.Tiers, overlay.tiers.Tiers)

	// Compliance: merge maps.
	result.compliance.Levels = mergeMap(base.compliance.Levels, overlay.compliance.Levels)

	// Profiles: merge maps and aliases.
	result.profiles.Profiles = mergeMap(base.profiles.Profiles, overlay.profiles.Profiles)
	result.profiles.Aliases = mergeStringMap(base.profiles.Aliases, overlay.profiles.Aliases)

	// Project profiles: merge maps.
	result.projectProfiles.Profiles = mergeMap(base.projectProfiles.Profiles, overlay.projectProfiles.Profiles)

	// Tools: merge maps.
	result.tools.Tools = mergeMap(base.tools.Tools, overlay.tools.Tools)

	// Security: merge lists and sub-structures.
	result.security.Hooks.Default = mergeStringSlice(base.security.Hooks.Default, overlay.security.Hooks.Default)
	result.security.BasePackages = mergeStringSlice(base.security.BasePackages, overlay.security.BasePackages)
	result.security.CleanEnvironment.UnsetVars = mergeStringSlice(
		base.security.CleanEnvironment.UnsetVars, overlay.security.CleanEnvironment.UnsetVars,
	)
	result.security.CleanEnvironment.KeepVars = mergeStringSlice(
		base.security.CleanEnvironment.KeepVars, overlay.security.CleanEnvironment.KeepVars,
	)
	result.security.CustomHooks = mergeCustomHooks(base.security.CustomHooks, overlay.security.CustomHooks)

	// Hook tiers: merge maps, overlay order wins if non-empty.
	result.hookTiers.Tiers = mergeStringSliceMap(base.hookTiers.Tiers, overlay.hookTiers.Tiers)
	result.hookTiers.TierOrder = mergeStringSlice(base.hookTiers.TierOrder, overlay.hookTiers.TierOrder)

	// Derivations: merge maps, overlay scalars win.
	result.derivations.TierToCompliance = mergeStringMap(base.derivations.TierToCompliance, overlay.derivations.TierToCompliance)
	result.derivations.TierToEnabledTools = mergeStringSliceMap(base.derivations.TierToEnabledTools, overlay.derivations.TierToEnabledTools)
	result.derivations.DefaultMCPServers = mergeStringSlice(base.derivations.DefaultMCPServers, overlay.derivations.DefaultMCPServers)
	if overlay.derivations.DefaultAgentTools != (DefaultAgentTools{}) {
		result.derivations.DefaultAgentTools = overlay.derivations.DefaultAgentTools
	} else {
		result.derivations.DefaultAgentTools = base.derivations.DefaultAgentTools
	}

	// Validation: overlay replaces if non-empty.
	result.validation = mergeValidation(base.validation, overlay.validation)

	return result
}

// mergeMap merges two maps of the same type, overlay entries win.
func mergeMap[V any](base, overlay map[string]V) map[string]V {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	out := make(map[string]V, len(base)+len(overlay))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		out[k] = v
	}
	return out
}

// mergeStringMap merges two string→string maps.
func mergeStringMap(base, overlay map[string]string) map[string]string {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	out := make(map[string]string, len(base)+len(overlay))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range overlay {
		out[k] = v
	}
	return out
}

// mergeStringSliceMap merges two map[string][]string, overlay entries win.
func mergeStringSliceMap(base, overlay map[string][]string) map[string][]string {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}
	out := make(map[string][]string, len(base)+len(overlay))
	for k, v := range base {
		cp := make([]string, len(v))
		copy(cp, v)
		out[k] = cp
	}
	for k, v := range overlay {
		cp := make([]string, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}

// mergeStringSlice returns the overlay if non-empty, otherwise the base.
func mergeStringSlice(base, overlay []string) []string {
	if len(overlay) > 0 {
		out := make([]string, len(overlay))
		copy(out, overlay)
		return out
	}
	if len(base) > 0 {
		out := make([]string, len(base))
		copy(out, base)
		return out
	}
	return nil
}

// mergeCustomHooks merges custom hook lists by ID.
func mergeCustomHooks(base, overlay []CustomHookDef) []CustomHookDef {
	if len(overlay) == 0 {
		out := make([]CustomHookDef, len(base))
		copy(out, base)
		return out
	}
	byID := make(map[string]CustomHookDef, len(base)+len(overlay))
	var order []string
	for _, h := range base {
		byID[h.ID] = h
		order = append(order, h.ID)
	}
	for _, h := range overlay {
		if _, exists := byID[h.ID]; !exists {
			order = append(order, h.ID)
		}
		byID[h.ID] = h
	}
	out := make([]CustomHookDef, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	return out
}

func mergeValidation(base, overlay ValidationFile) ValidationFile {
	result := base

	if len(overlay.Languages.All) > 0 {
		result.Languages.All = make([]string, len(overlay.Languages.All))
		copy(result.Languages.All, overlay.Languages.All)
	}
	if len(overlay.Languages.Core) > 0 {
		result.Languages.Core = make([]string, len(overlay.Languages.Core))
		copy(result.Languages.Core, overlay.Languages.Core)
	}
	if len(overlay.Services) > 0 {
		result.Services = make([]string, len(overlay.Services))
		copy(result.Services, overlay.Services)
	}
	if len(overlay.PermissionPresets) > 0 {
		result.PermissionPresets = make([]string, len(overlay.PermissionPresets))
		copy(result.PermissionPresets, overlay.PermissionPresets)
	}
	if len(overlay.HookPresets) > 0 {
		result.HookPresets = make([]string, len(overlay.HookPresets))
		copy(result.HookPresets, overlay.HookPresets)
	}
	if len(overlay.SecurityLevels) > 0 {
		result.SecurityLevels = make([]string, len(overlay.SecurityLevels))
		copy(result.SecurityLevels, overlay.SecurityLevels)
	}
	if len(overlay.DataClassifications) > 0 {
		result.DataClassifications = make([]string, len(overlay.DataClassifications))
		copy(result.DataClassifications, overlay.DataClassifications)
	}
	if len(overlay.PackageManagers) > 0 {
		result.PackageManagers = make(map[string][]string, len(overlay.PackageManagers))
		for k, v := range overlay.PackageManagers {
			cp := make([]string, len(v))
			copy(cp, v)
			result.PackageManagers[k] = cp
		}
	}
	if len(overlay.ToolCategories) > 0 {
		result.ToolCategories = make([]ToolCategoryDef, len(overlay.ToolCategories))
		copy(result.ToolCategories, overlay.ToolCategories)
	}

	return result
}
