package catalog

import (
	"cmp"
	"maps"
	"slices"
	"time"
)

// --- Tier accessors ---

// TierOrder returns the tier names sorted by ascending order. Ties (which
// Validate rejects) are broken by name so the result is deterministic.
func (c *Catalog) TierOrder() []string {
	type kv struct {
		name  string
		order int
	}
	items := make([]kv, 0, len(c.tiers.Tiers))
	for name, def := range c.tiers.Tiers {
		items = append(items, kv{name, def.Order})
	}
	slices.SortFunc(items, func(a, b kv) int {
		return cmp.Or(cmp.Compare(a.order, b.order), cmp.Compare(a.name, b.name))
	})
	result := make([]string, len(items))
	for i, item := range items {
		result[i] = item.name
	}
	return result
}

// TierDefs returns a copy of all tier definitions.
func (c *Catalog) TierDefs() map[string]TierDef {
	out := make(map[string]TierDef, len(c.tiers.Tiers))
	maps.Copy(out, c.tiers.Tiers)
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
	maps.Copy(out, c.compliance.Levels)
	return out
}

// ComplianceLevel returns the definition for a named compliance level.
func (c *Catalog) ComplianceLevel(name string) (ComplianceLevelDef, bool) {
	d, ok := c.compliance.Levels[name]
	return d, ok
}

// AgeGate returns the release-age window (age_gating_threshold_hours) of the
// named compliance level: the one release-age policy every package manager
// config, the package guard and the dependency-update bots enforce (each
// raised to its own historical floor). An empty or unknown level gets the
// lowest-order level's window, so a missing or mistyped security.level never
// loosens the gate below baseline. A catalog without compliance levels
// returns 0, which consumers clamp to their floor.
func (c *Catalog) AgeGate(level string) time.Duration {
	def, ok := c.compliance.Levels[level]
	if !ok {
		lowest := ""
		for name, d := range c.compliance.Levels {
			if lowest == "" || d.Order < def.Order || (d.Order == def.Order && name < lowest) {
				lowest, def = name, d
			}
		}
	}
	return time.Duration(def.AgeGatingThresholdHours) * time.Hour
}

// EffectiveAgeGate returns the default catalog's AgeGate for the named
// compliance level, or 0 when the catalog cannot be loaded.
func EffectiveAgeGate(level string) time.Duration {
	cat, err := Default()
	if err != nil {
		return 0
	}
	return cat.AgeGate(level)
}
