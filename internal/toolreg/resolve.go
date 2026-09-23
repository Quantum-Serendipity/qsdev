package toolreg

import (
	"fmt"
	"slices"
	"strings"
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

	// Conflicts are symmetric: an enabled tool that lists toolName among its
	// conflicts blocks the enable even when toolName does not list it back.
	for _, other := range registry.All() {
		if other.Name == toolName || !enabledTools[other.Name] {
			continue
		}
		if slices.Contains(other.Conflicts, toolName) {
			return fmt.Errorf("cannot enable %q: conflicts with enabled tool %q", toolName, other.Name)
		}
	}

	return nil
}

// AlwaysOnError is returned when attempting to disable a tool that has
// always-on policy. The caller may bypass this with --force.
type AlwaysOnError struct {
	ToolName string
}

func (e *AlwaysOnError) Error() string {
	return fmt.Sprintf("cannot disable %q: tool has always-on policy; use --force to override", e.ToolName)
}

// ValidateDisable checks that disabling toolName is valid — the tool must
// not be always-on (unless forced) and no other enabled tool may depend on it.
func ValidateDisable(registry *Registry, toolName string, enabledTools map[string]bool) error {
	tool, ok := registry.ByName(toolName)
	if !ok {
		return fmt.Errorf("unknown tool %q; use 'qsdev list' to see available tools", toolName)
	}

	if tool.Default == AlwaysOn {
		return &AlwaysOnError{ToolName: toolName}
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
