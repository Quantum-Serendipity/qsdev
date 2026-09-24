package tier

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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
	cat, err := catalog.Default()
	if err != nil {
		return 0, fmt.Errorf("loading catalog: %w", err)
	}
	def, ok := cat.TierDef(s)
	if !ok {
		return 0, fmt.Errorf("unknown tier %q; valid tiers: %s", s, strings.Join(cat.TierOrder(), ", "))
	}
	return Tier(def.Order), nil
}

// String returns the canonical name for the tier.
func (t Tier) String() string {
	cat, err := catalog.Default()
	if err != nil {
		return fmt.Sprintf("tier(%d)", int(t))
	}
	if name, _, ok := lookup(cat, t); ok {
		return name
	}
	return fmt.Sprintf("tier(%d)", int(t))
}

// lookup returns the catalog tier whose order equals t. It walks the
// deterministic TierOrder (catalog validation guarantees orders are unique)
// rather than ranging over the tier map.
func lookup(cat *catalog.Catalog, t Tier) (string, catalog.TierDef, bool) {
	for _, name := range cat.TierOrder() {
		if def, _ := cat.TierDef(name); Tier(def.Order) == t {
			return name, def, true
		}
	}
	return "", catalog.TierDef{}, false
}

// DefaultPermissionPreset returns the permission preset implied by this tier.
func (t Tier) DefaultPermissionPreset() string {
	cat, err := catalog.Default()
	if err == nil {
		if _, def, ok := lookup(cat, t); ok && def.DefaultPermissionPreset != "" {
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
	cat, err := catalog.Default()
	if err != nil {
		return nil
	}
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
	cat, err := catalog.Default()
	if err != nil {
		return "", false
	}
	order := cat.TierOrder()
	for i, name := range order {
		if name == current && i+1 < len(order) {
			return order[i+1], true
		}
	}
	return "", false
}

// PreviewCommand returns the command that previews regenerating an
// initialized project at tier name. --yes and --force are required: without
// them init leaves a set-up project untouched ("Nothing to do") and a
// non-terminal run cannot start the wizard, so nothing would be previewed.
func PreviewCommand(appName, name string) string {
	return fmt.Sprintf("%s init --yes --force --tier %s --dry-run", appName, name)
}

// Position returns the 1-based position of the named tier in the ordering,
// or 0 if the tier is not recognized.
func Position(name string) int {
	cat, err := catalog.Default()
	if err != nil {
		return 0
	}
	for i, t := range cat.TierOrder() {
		if t == name {
			return i + 1
		}
	}
	return 0
}

// Total returns the number of defined tiers.
func Total() int {
	cat, err := catalog.Default()
	if err != nil {
		return 0
	}
	return len(cat.TierOrder())
}

// Resolve determines the effective tier from an explicit tier string,
// falling back to inference from legacy fields when the explicit tier
// is not set or is invalid. An invalid explicit tier is logged rather than
// silently replaced; callers validate tiers at their input boundary
// (answers and .qsdev.yaml validation).
func Resolve(tierStr string, permissionLevel string, mcpServers []string) Tier {
	if tierStr != "" {
		t, err := ParseTier(tierStr)
		if err == nil {
			return t
		}
		inferred := Infer(permissionLevel, mcpServers)
		slog.Warn("ignoring invalid tier; falling back to inferred tier",
			"tier", tierStr, "inferred", int(inferred), "error", err)
		return inferred
	}
	return Infer(permissionLevel, mcpServers)
}

// Infer determines the most likely tier of a legacy .qsdev.yaml that predates
// the always-persisted tier field. A supply-chain-only permission level means
// that tier. The catalog's default MCP servers (and semble, provisioned by its
// agent tool) are written by every default init, so they never imply Full;
// only a server outside that set does. Anything else is the catalog's default
// tier.
func Infer(permissionLevel string, mcpServers []string) Tier {
	if permissionLevel == "supply-chain-only" {
		return SupplyChainOnly
	}
	cat, err := catalog.Default()
	if err != nil {
		return Standard
	}
	defaults := cat.DefaultMCPServers()
	for _, s := range mcpServers {
		if s != types.SembleMCPServer && !slices.Contains(defaults, s) {
			return Full
		}
	}
	return defaultTier(cat)
}

// Default returns the catalog's default tier: the tier a project gets when
// nothing selects one. It is Standard if the catalog cannot be loaded.
func Default() Tier {
	cat, err := catalog.Default()
	if err != nil {
		return Standard
	}
	return defaultTier(cat)
}

func defaultTier(cat *catalog.Catalog) Tier {
	if def, ok := cat.TierDef(cat.DefaultTier()); ok {
		return Tier(def.Order)
	}
	return Standard
}
