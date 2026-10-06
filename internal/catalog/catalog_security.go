package catalog

import "slices"

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

// SecurityHookIDs returns, sorted and without duplicates, the hook ids the
// catalog classifies as security hooks for the compliance level: the
// always-on security_hooks, the custom_hooks, and the level's
// required_pre_commit_hooks (none for an unknown or empty level). Hook tier
// members that are none of these (formatters, linters) are not security
// hooks.
func (c *Catalog) SecurityHookIDs(level string) []string {
	ids := slices.Clone(c.security.Hooks.Default)
	for _, h := range c.security.CustomHooks {
		ids = append(ids, h.ID)
	}
	ids = append(ids, c.compliance.Levels[level].RequiredPreCommitHooks...)
	slices.Sort(ids)
	return slices.Compact(ids)
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
