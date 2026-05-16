package devenv

// hookTiers defines which hooks belong to each tier.
// Tier hierarchy: baseline < enhanced < specialized < full.
// Higher tiers include all hooks from lower tiers.
var hookTiers = map[string][]string{
	"baseline": {
		"ripsecrets",
		"gitleaks",
		"check-added-large-files",
		"no-commit-to-branch",
		"check-merge-conflicts",
	},
	"enhanced": {
		"semgrep",
		"shellcheck",
		"formatters",
	},
	"specialized": {
		"lock-file-audit",
		"nix-secrets-check",
		"statix",
	},
}

// tierOrder defines the cumulative ordering of tiers.
var tierOrder = []string{"baseline", "enhanced", "specialized"}

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
