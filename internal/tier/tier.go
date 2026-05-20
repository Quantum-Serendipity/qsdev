package tier

import "fmt"

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

var tiersByName = map[string]Tier{
	"supply-chain-only": SupplyChainOnly,
	"standard":          Standard,
	"full":              Full,
}

var tierNames = map[Tier]string{
	SupplyChainOnly: "supply-chain-only",
	Standard:        "standard",
	Full:            "full",
}

// Order is the canonical tier ordering from lowest to highest.
var Order = []string{"supply-chain-only", "standard", "full"}

// ParseTier converts a string to a Tier. Returns an error for unknown values.
func ParseTier(s string) (Tier, error) {
	t, ok := tiersByName[s]
	if !ok {
		return 0, fmt.Errorf("unknown tier %q; valid tiers: supply-chain-only, standard, full", s)
	}
	return t, nil
}

// String returns the canonical name for the tier.
func (t Tier) String() string {
	if name, ok := tierNames[t]; ok {
		return name
	}
	return fmt.Sprintf("tier(%d)", int(t))
}

// DefaultPermissionPreset returns the permission preset implied by this tier.
func (t Tier) DefaultPermissionPreset() string {
	if t <= SupplyChainOnly {
		return "supply-chain-only"
	}
	return "standard"
}

// AllTiers returns information about every tier in order.
func AllTiers() []TierInfo {
	return []TierInfo{
		{
			Name:        "supply-chain-only",
			Level:       SupplyChainOnly,
			Description: "Package supply chain security + devenv sandbox; no Claude Code restrictions",
		},
		{
			Name:        "standard",
			Level:       Standard,
			Description: "Supply chain deny rules + Claude Code governance + CLAUDE.md + gitleaks",
		},
		{
			Name:        "full",
			Level:       Full,
			Description: "Full tooling: MCP servers, agent tools, consulting workflows, AlwaysOn tools",
		},
	}
}

// NextTier returns the next tier above current, or false if already at max.
func NextTier(current string) (string, bool) {
	for i, name := range Order {
		if name == current && i+1 < len(Order) {
			return Order[i+1], true
		}
	}
	return "", false
}

// Position returns the 1-based position of the named tier in the ordering,
// or 0 if the tier is not recognized.
func Position(name string) int {
	for i, t := range Order {
		if t == name {
			return i + 1
		}
	}
	return 0
}

// Total returns the number of defined tiers.
func Total() int {
	return len(Order)
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
