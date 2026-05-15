package toolreg

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ValidateEnable checks that enabling toolName is valid given the current
// set of enabled tools. Returns nil if valid, or an error describing the
// unmet prerequisite or conflict.
func ValidateEnable(registry *Registry, toolName string, enabledTools map[string]bool) error {
	tool, ok := registry.ByName(toolName)
	if !ok {
		return fmt.Errorf("unknown tool %q; use 'qsdev list' to see available tools", toolName)
	}

	for _, prereq := range tool.Prerequisites {
		if !enabledTools[prereq] {
			return fmt.Errorf("cannot enable %q: prerequisite %q is not enabled", toolName, prereq)
		}
	}

	for _, conflict := range tool.Conflicts {
		if enabledTools[conflict] {
			return fmt.Errorf("cannot enable %q: conflicts with enabled tool %q", toolName, conflict)
		}
	}

	return nil
}

// ValidateDisable checks that disabling toolName is valid — no other
// enabled tool lists it as a prerequisite.
func ValidateDisable(registry *Registry, toolName string, enabledTools map[string]bool) error {
	if _, ok := registry.ByName(toolName); !ok {
		return fmt.Errorf("unknown tool %q; use 'qsdev list' to see available tools", toolName)
	}

	var dependents []string
	for _, tool := range registry.All() {
		if !enabledTools[tool.Name] {
			continue
		}
		for _, prereq := range tool.Prerequisites {
			if prereq == toolName {
				dependents = append(dependents, tool.Name)
			}
		}
	}

	if len(dependents) > 0 {
		return fmt.Errorf("cannot disable %q: required by %s", toolName, strings.Join(dependents, ", "))
	}

	return nil
}

// ComputeDefaults returns the set of tools that should be enabled by
// default for a project with the given detection results.
func ComputeDefaults(registry *Registry, detected types.DetectedProject) map[string]bool {
	enabled := make(map[string]bool)

	for _, tool := range registry.All() {
		switch tool.Default {
		case AlwaysOn:
			enabled[tool.Name] = true
		case OnWhenDetected:
			if tool.DetectFunc != nil && tool.DetectFunc(detected) {
				enabled[tool.Name] = true
			}
		case OptIn, AlwaysOff:
			// Not enabled by default.
		}
	}

	return enabled
}
