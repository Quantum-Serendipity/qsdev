package config

import (
	"slices"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// mergePointerBool merges two *bool values. nil means "inherit from base";
// a non-nil overlay overrides the base value.
func mergePointerBool(base, overlay *bool) *bool {
	if overlay != nil {
		v := *overlay
		return &v
	}
	if base != nil {
		v := *base
		return &v
	}
	return nil
}

// mergeUnionStrings returns a deduplicated union of base and overlay slices,
// preserving the order in which items first appear (base items first).
func mergeUnionStrings(base, overlay []string) []string {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}

	seen := make(map[string]bool, max(len(base), len(overlay)))
	var result []string

	for _, s := range base {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range overlay {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}

// mergePackageGuard merges two hooks.package_guard blocks so that no layer
// loosens the gate: lists are unioned, min_age_days takes the larger value and
// new_dep_gate the stricter one (later in types.NewDepGates()). The result
// never aliases an input's pointer.
func mergePackageGuard(base, overlay types.PackageGuardConfig) types.PackageGuardConfig {
	out := types.PackageGuardConfig{
		Allowlist:  mergeUnionStrings(base.Allowlist, overlay.Allowlist),
		Denylist:   mergeUnionStrings(base.Denylist, overlay.Denylist),
		NewDepGate: base.NewDepGate,
	}
	if overlay.NewDepGate != "" && (base.NewDepGate == "" || newDepGateRank(overlay.NewDepGate) > newDepGateRank(base.NewDepGate)) {
		out.NewDepGate = overlay.NewDepGate
	}
	for _, days := range []*int{base.MinAgeDays, overlay.MinAgeDays} {
		if days != nil && (out.MinAgeDays == nil || *days > *out.MinAgeDays) {
			v := *days
			out.MinAgeDays = &v
		}
	}
	return out
}

// newDepGateRank orders new_dep_gate values from loosest to strictest; an
// unknown value (rejected by validation) ranks below every known one.
func newDepGateRank(gate string) int {
	return slices.Index(types.NewDepGates(), gate)
}

// mergeReplaceLanguages replaces the base languages with the overlay when the
// overlay is non-empty. An empty overlay means "use base".
func mergeReplaceLanguages(base, overlay []types.LanguageConfig) []types.LanguageConfig {
	if len(overlay) > 0 {
		out := make([]types.LanguageConfig, len(overlay))
		copy(out, overlay)
		return out
	}
	if len(base) > 0 {
		out := make([]types.LanguageConfig, len(base))
		copy(out, base)
		return out
	}
	return nil
}

// mergeReplaceServices replaces the base services with the overlay when the
// overlay is non-empty. An empty overlay means "use base".
func mergeReplaceServices(base, overlay []types.ServiceConfig) []types.ServiceConfig {
	if len(overlay) > 0 {
		out := make([]types.ServiceConfig, len(overlay))
		copy(out, overlay)
		return out
	}
	if len(base) > 0 {
		out := make([]types.ServiceConfig, len(base))
		copy(out, base)
		return out
	}
	return nil
}

// mergeMapStringAny performs a recursive key-level merge of two
// map[string]map[string]any structures. Later (overlay) keys win at the
// leaf level. Neither input is modified.
func mergeMapStringAny(base, overlay map[string]map[string]any) map[string]map[string]any {
	if len(base) == 0 && len(overlay) == 0 {
		return nil
	}

	result := make(map[string]map[string]any, max(len(base), len(overlay)))

	// Copy base entries.
	for k, v := range base {
		inner := make(map[string]any, len(v))
		for ik, iv := range v {
			inner[ik] = iv
		}
		result[k] = inner
	}

	// Merge overlay entries.
	for k, v := range overlay {
		existing, ok := result[k]
		if !ok {
			existing = make(map[string]any, len(v))
			result[k] = existing
		}
		for ik, iv := range v {
			existing[ik] = iv
		}
	}

	return result
}
