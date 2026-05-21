package tier

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
)

// Tier represents an ordered security onboarding level. Each tier is a strict
// superset of the previous: features enabled at tier N are always present at
// tier N+1.
type Tier int

const (
	SupplyChainOnly Tier = 1
	Standard        Tier = 2
	Full            Tier = 3
)

// TierInfo describes a tier for display purposes.
type TierInfo struct {
	Name        string
	Level       Tier
	Description string
}

// ParseTier converts a string to a Tier. Returns an error for unknown values.
func ParseTier(s string) (Tier, error) {
	defs := catalog.Default().TierDefs()
	def, ok := defs[s]
	if !ok {
		return 0, fmt.Errorf("unknown tier %q; valid tiers: supply-chain-only, standard, full", s)
	}
	return Tier(def.Order), nil
}

// String returns the canonical name for the tier.
func (t Tier) String() string {
	for name, def := range catalog.Default().TierDefs() {
		if Tier(def.Order) == t {
			return name
		}
	}
	return fmt.Sprintf("tier(%d)", int(t))
}

// DefaultPermissionPreset returns the permission preset implied by this tier.
func (t Tier) DefaultPermissionPreset() string {
	for _, def := range catalog.Default().TierDefs() {
		if Tier(def.Order) == t && def.DefaultPermissionPreset != "" {
			return def.DefaultPermissionPreset
		}
	}
	if t <= SupplyChainOnly {
		return "supply-chain-only"
	}
	return "standard"
}

// AllTiers returns information about every tier in order.
// Backed by internal/catalog/defaults/tiers.yaml.
func AllTiers() []TierInfo {
	cat := catalog.Default()
	order := cat.TierOrder()
	defs := cat.TierDefs()

	infos := make([]TierInfo, 0, len(order))
	for _, name := range order {
		def := defs[name]
		infos = append(infos, TierInfo{
			Name:        name,
			Level:       Tier(def.Order),
			Description: def.Description,
		})
	}
	return infos
}

// NextTier returns the next tier above current, or false if already at max.
func NextTier(current string) (string, bool) {
	order := catalog.Default().TierOrder()
	for i, name := range order {
		if name == current && i+1 < len(order) {
			return order[i+1], true
		}
	}
	return "", false
}

// Position returns the 1-based position of the named tier in the ordering,
// or 0 if the tier is not recognized.
func Position(name string) int {
	for i, t := range catalog.Default().TierOrder() {
		if t == name {
			return i + 1
		}
	}
	return 0
}

// Total returns the number of defined tiers.
func Total() int {
	return len(catalog.Default().TierOrder())
}

// Resolve determines the effective tier from an explicit tier string,
// falling back to inference from legacy fields when the explicit tier
// is not set or is invalid.
func Resolve(tierStr string, permissionLevel string, mcpServers []string) Tier {
	if tierStr != "" {
		if t, err := ParseTier(tierStr); err == nil {
			return t
		}
	}
	return Infer(permissionLevel, mcpServers)
}

// Infer determines the most likely tier from legacy config fields that predate
// the explicit tier field. Used for backward compatibility with existing
// .qsdev.yaml files.
func Infer(permissionLevel string, mcpServers []string) Tier {
	if permissionLevel == "supply-chain-only" {
		return SupplyChainOnly
	}
	if len(mcpServers) > 0 {
		return Full
	}
	return Standard
}
