package check

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// ApplyAutoFixes attempts to fix each AutoFixable result. It returns an
// updated copy of the results slice with fixed results changed to StatusPass.
func ApplyAutoFixes(results []CheckResult, projectRoot string) []CheckResult {
	updated := make([]CheckResult, len(results))
	copy(updated, results)

	// Collect missing deny rules.
	var missingRules []string
	var ruleIndices []int
	for i, r := range updated {
		if r.AutoFixable && r.Name == "deny_rule_missing" {
			if rule, ok := r.Metadata["rule"]; ok {
				missingRules = append(missingRules, rule)
				ruleIndices = append(ruleIndices, i)
			}
		}
	}

	if len(missingRules) > 0 {
		if err := fixDenyRules(projectRoot, missingRules); err == nil {
			for _, idx := range ruleIndices {
				updated[idx].Status = StatusPass
				updated[idx].Message = "Auto-fixed: " + updated[idx].Message
				updated[idx].AutoFixable = false
			}
		}
	}

	return updated
}

// fixDenyRules reads .claude/settings.json, adds missing deny rules, and
// writes it back.
func fixDenyRules(projectRoot string, missingRules []string) error {
	settingsPath := filepath.Join(projectRoot, ".claude", "settings.json")

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		return fmt.Errorf("reading settings.json: %w", err)
	}

	// Parse into a generic map to preserve unknown fields.
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("parsing settings.json: %w", err)
	}

	// Navigate to permissions.deny.
	perms, ok := settings["permissions"]
	if !ok {
		perms = map[string]any{}
		settings["permissions"] = perms
	}
	permsMap, ok := perms.(map[string]any)
	if !ok {
		return fmt.Errorf("permissions is not an object")
	}

	var existingDeny []any
	if d, ok := permsMap["deny"]; ok {
		existingDeny, _ = d.([]any)
	}

	// Build set of existing rules.
	existing := make(map[string]bool, len(existingDeny))
	for _, r := range existingDeny {
		if s, ok := r.(string); ok {
			existing[s] = true
		}
	}

	// Add missing rules.
	for _, rule := range missingRules {
		if !existing[rule] {
			existingDeny = append(existingDeny, rule)
		}
	}
	permsMap["deny"] = existingDeny

	// Write back.
	out, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings.json: %w", err)
	}
	out = append(out, '\n')

	if err := os.WriteFile(settingsPath, out, 0o644); err != nil {
		return fmt.Errorf("writing settings.json: %w", err)
	}

	return nil
}
