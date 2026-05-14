package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestApplyAutoFixes_DenyRules(t *testing.T) {
	dir := t.TempDir()

	// Create .claude/settings.json with some rules but missing others.
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	settings := map[string]any{
		"permissions": map[string]any{
			"deny": []string{`Bash(rm -rf *)`},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create results with a missing deny rule.
	results := []CheckResult{
		{
			Name:        "deny_rule_missing",
			Status:      StatusFail,
			Severity:    SeverityMedium,
			Message:     "Required deny rule missing: Bash(git push --force *)",
			AutoFixable: true,
		},
		{
			Name:     "other_check",
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  "ok",
		},
	}

	updated := ApplyAutoFixes(results, dir)

	// The deny rule should be fixed.
	if updated[0].Status != StatusPass {
		t.Errorf("updated[0].Status = %s, want %s", updated[0].Status, StatusPass)
	}
	if updated[0].AutoFixable {
		t.Error("fixed result should have AutoFixable=false")
	}

	// Verify the rule was added to the file.
	newData, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}

	var parsed struct {
		Permissions struct {
			Deny []string `json:"deny"`
		} `json:"permissions"`
	}
	if err := json.Unmarshal(newData, &parsed); err != nil {
		t.Fatal(err)
	}

	found := false
	for _, rule := range parsed.Permissions.Deny {
		if rule == `Bash(git push --force *)` {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected deny rule to be added to settings.json")
	}

	// The existing rule should still be there.
	existingFound := false
	for _, rule := range parsed.Permissions.Deny {
		if rule == `Bash(rm -rf *)` {
			existingFound = true
			break
		}
	}
	if !existingFound {
		t.Error("existing deny rule should be preserved")
	}
}

func TestApplyAutoFixes_NoAutoFixable(t *testing.T) {
	results := []CheckResult{
		{
			Name:        "not_fixable",
			Status:      StatusFail,
			Severity:    SeverityHigh,
			Message:     "something bad",
			AutoFixable: false,
		},
	}

	updated := ApplyAutoFixes(results, t.TempDir())

	if updated[0].Status != StatusFail {
		t.Errorf("non-fixable result should remain failed; got %s", updated[0].Status)
	}
}
