package ecosystem

import (
	"context"
	"maps"
	"slices"
)

// ToolchainWarnings returns the warnings each ecosystem detected in
// projectRoot reports about the toolchain on PATH (see ToolchainChecker),
// ordered by module name and each prefixed with the module's display name.
// results are the per-module detections for projectRoot (DetectAll); each
// module is checked with the configuration its own Detect suggested.
// Undetected ecosystems, and modules that do not implement ToolchainChecker,
// contribute nothing.
func (r *Registry) ToolchainWarnings(ctx context.Context, projectRoot string, results map[string]DetectionResult) []string {
	var warnings []string
	for _, name := range slices.Sorted(maps.Keys(results)) {
		if !results[name].Detected {
			continue
		}
		m, ok := r.ByName(name)
		if !ok {
			continue
		}
		c, ok := m.(ToolchainChecker)
		if !ok {
			continue
		}
		for _, msg := range c.ToolchainWarnings(ctx, projectRoot, results[name].SuggestedConfig) {
			warnings = append(warnings, m.DisplayName()+": "+msg)
		}
	}
	return warnings
}
