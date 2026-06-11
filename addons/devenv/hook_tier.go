package devenv

import "github.com/Quantum-Serendipity/qsdev/internal/catalog"

// FilterHooksByTier returns the subset of hooks that belong to the given tier
// or any tier below it. The "full" tier returns all hooks unchanged.
func FilterHooksByTier(hooks []string, tier string) []string {
	if tier == "" || tier == "full" {
		return hooks
	}

	allowed := allowedHooksForTier(tier)

	var result []string
	for _, h := range hooks {
		if allowed[h] {
			result = append(result, h)
		}
	}
	return result
}

// allowedHooksForTier builds the set of hooks allowed at the given tier level.
func allowedHooksForTier(tier string) map[string]bool {
	cat, err := catalog.Default()
	if err != nil {
		return nil
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

	return allowed
}
