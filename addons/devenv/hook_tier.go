package devenv

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
)

// Hook tiers come from the catalog's hook_tier_order and hook_tiers, one tier
// per security level. A project runs the hooks of its security level's tier
// and of every tier below it. Only the hooks the catalog places above that
// tier are dropped: the lowest tier (where every security hook lives) and any
// hook no tier lists run at every level, so an uncatalogued hook, such as a
// new ecosystem security scanner, is never silently removed.

// FilterHooksByTier returns hooks without the ones the catalog assigns to a
// tier above the given one, keeping their order. An empty tier keeps every
// hook. It fails closed: an unknown tier, or a catalog that cannot be loaded,
// is an error rather than a guess at which hooks to keep.
func FilterHooksByTier(hooks []string, tier string) ([]string, error) {
	keep, err := hookTierFilter(tier)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, h := range hooks {
		if keep(h) {
			result = append(result, h)
		}
	}
	return result, nil
}

// hookTierFilter returns a predicate that reports whether the hook with the
// given ID runs at the strictest of levels. Empty levels are ignored, and
// with none left every hook runs. BuildDevenvNixData passes both the answers'
// HookTier and ComplianceLevel: HookTier is set from the effective security
// level when answers come from .qsdev.yaml, ComplianceLevel is the floor on
// every path, and the stricter of the two can only keep more hooks.
func hookTierFilter(levels ...string) (func(id string) bool, error) {
	levels = slices.DeleteFunc(slices.Clone(levels), func(l string) bool { return l == "" })
	if len(levels) == 0 {
		return func(string) bool { return true }, nil
	}
	cat, err := catalog.Default()
	if err != nil {
		return nil, fmt.Errorf("loading hook tiers from catalog: %w", err)
	}
	above, err := hooksAboveTier(cat.HookTierOrder(), cat.HookTiers(), levels)
	if err != nil {
		return nil, err
	}
	return func(id string) bool { return !above[id] }, nil
}

// hooksAboveTier returns the hooks listed by the tiers above the strictest of
// levels, each of which must name a tier in order.
func hooksAboveTier(order []string, tiers map[string][]string, levels []string) (map[string]bool, error) {
	top := -1
	for _, level := range levels {
		i := slices.Index(order, level)
		if i < 0 {
			return nil, fmt.Errorf("unknown hook tier %q (want one of: %s)", level, strings.Join(order, ", "))
		}
		top = max(top, i)
	}
	above := make(map[string]bool)
	for _, t := range order[top+1:] {
		for _, h := range tiers[t] {
			above[h] = true
		}
	}
	return above, nil
}
