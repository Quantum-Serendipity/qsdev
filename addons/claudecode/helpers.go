package claudecode

import (
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// toModuleConfig converts a LanguageChoice into an ecosystem.ModuleConfig.
func toModuleConfig(lang types.LanguageChoice) ecosystem.ModuleConfig {
	return ecosystem.ToModuleConfig(lang)
}

// extrasMap converts extras to a map[string]string via the shared helper.
func extrasMap(extras []string) map[string]string {
	return ecosystem.ExtrasMap(extras)
}

// contains checks whether a string slice includes the given value.
func contains(slice []string, val string) bool {
	return ecosystem.ContainsStr(slice, val)
}

// dedup returns a new slice with duplicates removed, preserving order.
func dedup(items []string) []string {
	return ecosystem.DedupStrings(items)
}
