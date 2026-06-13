package catalog

import "maps"

// --- Profile accessors ---

// Profiles returns a copy of all tier-based profiles.
func (c *Catalog) Profiles() map[string]ProfileDef {
	out := make(map[string]ProfileDef, len(c.profiles.Profiles))
	maps.Copy(out, c.profiles.Profiles)
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
	maps.Copy(out, c.profiles.Aliases)
	return out
}

// --- Project profile accessors ---

// ProjectProfiles returns a copy of all project-type profiles.
func (c *Catalog) ProjectProfiles() map[string]ProjectProfileDef {
	out := make(map[string]ProjectProfileDef, len(c.projectProfiles.Profiles))
	maps.Copy(out, c.projectProfiles.Profiles)
	return out
}

// ProjectProfile returns a named project-type profile.
func (c *Catalog) ProjectProfile(name string) (ProjectProfileDef, bool) {
	d, ok := c.projectProfiles.Profiles[name]
	return d, ok
}
