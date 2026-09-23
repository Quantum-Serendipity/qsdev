package devenv

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
)

// FilterHooksByTier returns the subset of hooks that belong to the given tier
// or any tier below it. The "full" tier returns all hooks unchanged. It fails
// closed: when the catalog tier definitions cannot be loaded it returns an
// error rather than an empty hook set, so a caller can never silently strip
// every security hook.
func FilterHooksByTier(hooks []string, tier string) ([]string, error) {
	if tier == "" || tier == "full" {
		return hooks, nil
	}

	allowed, err := allowedHooksForTier(tier)
	if err != nil {
		return nil, err
	}

	var result []string
	for _, h := range hooks {
		if allowed[h] {
			result = append(result, h)
		}
	}
	return result, nil
}

// allowedHooksForTier builds the set of hooks allowed at the given tier level.
func allowedHooksForTier(tier string) (map[string]bool, error) {
	cat, err := catalog.Default()
	if err != nil {
		return nil, fmt.Errorf("loading hook tiers from catalog: %w", err)
	}
	hookTiers := cat.HookTiers()
	tierOrder := cat.HookTierOrder()

	allowed := make(map[string]bool)
	for _, t := range tierOrder {
		for _, h := range hookTiers[t] {
			allowed[h] = true
		}
		if t == tier {
			break
		}
	}

	return allowed, nil
}
