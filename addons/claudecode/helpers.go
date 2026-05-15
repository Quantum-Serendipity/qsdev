package claudecode

import (
	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// toModuleConfig converts a LanguageChoice into an ecosystem.ModuleConfig.
func toModuleConfig(lang types.LanguageChoice) ecosystem.ModuleConfig {
	return ecosystem.ToModuleConfig(lang)
}

// contains checks whether a string slice includes the given value.
func contains(slice []string, val string) bool {
	return ecosystem.ContainsStr(slice, val)
}

// dedup returns a new slice with duplicates removed, preserving order.
func dedup(items []string) []string {
	return ecosystem.DedupStrings(items)
}
