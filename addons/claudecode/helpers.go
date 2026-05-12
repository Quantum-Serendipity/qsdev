package claudecode

import (
	"strings"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// toModuleConfig converts a LanguageChoice from wizard answers into an
// ecosystem.ModuleConfig suitable for passing to EcosystemModule methods.
func toModuleConfig(lang types.LanguageChoice) ecosystem.ModuleConfig {
	return ecosystem.ModuleConfig{
		Version:        lang.Version,
		PackageManager: lang.PackageManager,
		Extras:         extrasMap(lang.Extras),
	}
}

// extrasMap converts the []string extras from LanguageChoice into a
// map[string]string for ModuleConfig.Extras.
func extrasMap(extras []string) map[string]string {
	if len(extras) == 0 {
		return nil
	}
	m := make(map[string]string, len(extras))
	for _, e := range extras {
		if k, v, ok := strings.Cut(e, "="); ok {
			m[k] = v
		} else {
			m[e] = "true"
		}
	}
	return m
}

// contains checks whether a string slice includes the given value.
func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

// dedup returns a new slice with duplicates removed, preserving order.
func dedup(items []string) []string {
	seen := make(map[string]bool, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}
	return result
}
