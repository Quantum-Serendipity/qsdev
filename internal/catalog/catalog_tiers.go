package catalog

import (
	"cmp"
	"maps"
	"slices"
)

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
	slices.SortFunc(items, func(a, b kv) int {
		return cmp.Compare(a.order, b.order)
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

// --- Derivation accessors (tier/compliance mappings) ---

// TierToCompliance returns the tier->compliance level mapping.
func (c *Catalog) TierToCompliance() map[string]string {
	out := make(map[string]string, len(c.derivations.TierToCompliance))
	maps.Copy(out, c.derivations.TierToCompliance)
	return out
}

// TierToEnabledTools returns the tier->enabled tools mapping.
func (c *Catalog) TierToEnabledTools() map[string][]string {
	out := make(map[string][]string, len(c.derivations.TierToEnabledTools))
	for k, v := range c.derivations.TierToEnabledTools {
		cp := make([]string, len(v))
		copy(cp, v)
		out[k] = cp
	}
	return out
}
