package check

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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
			Metadata:    map[string]string{"rule": `Bash(git push --force *)`},
		},
		{
			Name:     "other_check",
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  "ok",
		},
	}

	updated := ApplyAutoFixes(results, dir, nil)

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

	updated := ApplyAutoFixes(results, t.TempDir(), nil)

	if updated[0].Status != StatusFail {
		t.Errorf("non-fixable result should remain failed; got %s", updated[0].Status)
	}
}

func TestApplyAutoFixes_DeletedFiles(t *testing.T) {
	dir := t.TempDir()

	results := []CheckResult{
		{
			Name:        "file_exists_test.txt",
			Status:      StatusFail,
			Severity:    SeverityHigh,
			Message:     "Generated file test.txt has been deleted",
			FilePath:    "test.txt",
			AutoFixable: true,
			Metadata:    map[string]string{"file": "test.txt"},
		},
		{
			Name:     "other_check",
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  "ok",
		},
	}

	freshContent := []byte("restored content")
	regen := func(_ string) (map[string]types.GeneratedFile, error) {
		return map[string]types.GeneratedFile{
			"test.txt": {
				Path:    "test.txt",
				Content: freshContent,
				Mode:    0o644,
			},
		}, nil
	}

	updated := ApplyAutoFixes(results, dir, regen)

	if updated[0].Status != StatusPass {
		t.Errorf("updated[0].Status = %s, want %s", updated[0].Status, StatusPass)
	}
	if updated[0].AutoFixable {
		t.Error("fixed result should have AutoFixable=false")
	}

	restored, err := os.ReadFile(filepath.Join(dir, "test.txt"))
	if err != nil {
		t.Fatalf("file should be restored: %v", err)
	}
	if string(restored) != "restored content" {
		t.Errorf("restored content = %q, want %q", string(restored), "restored content")
	}

	if updated[1].Status != StatusPass {
		t.Errorf("other check should remain pass, got %s", updated[1].Status)
	}
}

func TestApplyAutoFixes_DeletedFiles_NilRegenerate(t *testing.T) {
	results := []CheckResult{
		{
			Name:        "file_exists_test.txt",
			Status:      StatusFail,
			Severity:    SeverityHigh,
			Message:     "Generated file test.txt has been deleted",
			FilePath:    "test.txt",
			AutoFixable: true,
			Metadata:    map[string]string{"file": "test.txt"},
		},
	}

	updated := ApplyAutoFixes(results, t.TempDir(), nil)

	if updated[0].Status != StatusFail {
		t.Errorf("should remain failed without regenerate func, got %s", updated[0].Status)
	}
}

func TestApplyAutoFixes_DeletedFiles_RegenerateFails(t *testing.T) {
	results := []CheckResult{
		{
			Name:        "file_exists_test.txt",
			Status:      StatusFail,
			Severity:    SeverityHigh,
			Message:     "Generated file test.txt has been deleted",
			FilePath:    "test.txt",
			AutoFixable: true,
			Metadata:    map[string]string{"file": "test.txt"},
		},
	}

	regen := func(_ string) (map[string]types.GeneratedFile, error) {
		return nil, fmt.Errorf("generation failed")
	}

	updated := ApplyAutoFixes(results, t.TempDir(), regen)

	if updated[0].Status != StatusFail {
		t.Errorf("should remain failed on regen error, got %s", updated[0].Status)
	}
}

func TestApplyAutoFixes_DeletedFiles_FileNotInFreshSet(t *testing.T) {
	results := []CheckResult{
		{
			Name:        "file_exists_orphaned.txt",
			Status:      StatusFail,
			Severity:    SeverityHigh,
			Message:     "Generated file orphaned.txt has been deleted",
			FilePath:    "orphaned.txt",
			AutoFixable: true,
			Metadata:    map[string]string{"file": "orphaned.txt"},
		},
	}

	regen := func(_ string) (map[string]types.GeneratedFile, error) {
		return map[string]types.GeneratedFile{
			"other.txt": {Path: "other.txt", Content: []byte("other"), Mode: 0o644},
		}, nil
	}

	updated := ApplyAutoFixes(results, t.TempDir(), regen)

	if updated[0].Status != StatusFail {
		t.Errorf("should remain failed when file not in fresh set, got %s", updated[0].Status)
	}
}

func TestApplyAutoFixes_DeletedFiles_NestedDirectory(t *testing.T) {
	dir := t.TempDir()

	results := []CheckResult{
		{
			Name:        "file_exists_.claude/local.json",
			Status:      StatusFail,
			Severity:    SeverityHigh,
			Message:     "Generated file .claude/local.json has been deleted",
			FilePath:    ".claude/local.json",
			AutoFixable: true,
			Metadata:    map[string]string{"file": ".claude/local.json"},
		},
	}

	regen := func(_ string) (map[string]types.GeneratedFile, error) {
		return map[string]types.GeneratedFile{
			".claude/local.json": {
				Path:    ".claude/local.json",
				Content: []byte(`{"test":true}`),
				Mode:    0o644,
			},
		}, nil
	}

	updated := ApplyAutoFixes(results, dir, regen)

	if updated[0].Status != StatusPass {
		t.Errorf("should fix nested file, got %s", updated[0].Status)
	}

	restored, err := os.ReadFile(filepath.Join(dir, ".claude", "local.json"))
	if err != nil {
		t.Fatalf("nested file should be restored: %v", err)
	}
	if string(restored) != `{"test":true}` {
		t.Errorf("unexpected content: %s", string(restored))
	}
}
