package catalog

// --- DefaultsProvider implementation (satisfies types.DefaultsProvider) ---

// DefaultPostmortem returns the default postmortem enabled setting.
func (c *Catalog) DefaultPostmortem() bool {
	return c.derivations.DefaultAgentTools.PostmortemEnabled
}

// DefaultVersionSentinel returns the default version sentinel enabled setting.
func (c *Catalog) DefaultVersionSentinel() bool {
	return c.derivations.DefaultAgentTools.VersionSentinel
}

// DefaultVersionSentinelHours returns the default version sentinel hours.
func (c *Catalog) DefaultVersionSentinelHours() int {
	return c.derivations.DefaultAgentTools.VersionSentinelHours
}

// DefaultSembleEnabled returns the default semble enabled setting.
func (c *Catalog) DefaultSembleEnabled() bool {
	return c.derivations.DefaultAgentTools.SembleEnabled
}

// DefaultSembleMode returns the default semble mode.
func (c *Catalog) DefaultSembleMode() string {
	return c.derivations.DefaultAgentTools.SembleMode
}

// DefaultAgentToolConfig returns the default agent tool settings.
func (c *Catalog) DefaultAgentToolConfig() DefaultAgentTools {
	return c.derivations.DefaultAgentTools
}

// TierCompliance returns the compliance level for a given tier, or empty string.
func (c *Catalog) TierCompliance(tier string) string {
	return c.derivations.TierToCompliance[tier]
}

// TierEnabledTools returns the enabled tools list for a given tier.
func (c *Catalog) TierEnabledTools(tier string) []string {
	tools := c.derivations.TierToEnabledTools[tier]
	if len(tools) == 0 {
		return nil
	}
	out := make([]string, len(tools))
	copy(out, tools)
	return out
}
