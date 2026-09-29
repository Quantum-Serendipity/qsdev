package catalog

import "maps"

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
