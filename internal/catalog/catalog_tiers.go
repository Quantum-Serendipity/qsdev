package catalog

import (
	"cmp"
	"maps"
	"slices"
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
