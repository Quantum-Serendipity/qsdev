package check

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
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

	updated := ApplyAutoFixes(results, dir, "", nil)

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

	updated := ApplyAutoFixes(results, t.TempDir(), "", nil)

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

	updated := ApplyAutoFixes(results, dir, "", regen)

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

	updated := ApplyAutoFixes(results, t.TempDir(), "", nil)

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

	updated := ApplyAutoFixes(results, t.TempDir(), "", regen)

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

	updated := ApplyAutoFixes(results, t.TempDir(), "", regen)

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

	updated := ApplyAutoFixes(results, dir, "", regen)

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

func TestAddDenyRules(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    string
		wantErr string
	}{
		{
			name:  "preserves key order and does not HTML-escape",
			input: `{"z":1,"permissions":{"allow":["Bash(a > b && c)"],"deny":["Bash(x)"]},"a":"<tag>"}`,
			want: "{\n  \"z\": 1,\n  \"permissions\": {\n    \"allow\": [\n      \"Bash(a > b && c)\"\n    ],\n" +
				"    \"deny\": [\n      \"Bash(x)\",\n      \"Bash(curl * | sh)\"\n    ]\n  },\n  \"a\": \"<tag>\"\n}\n",
		},
		{
			name:  "creates permissions and deny",
			input: `{"hooks":{}}`,
			want:  "{\n  \"hooks\": {},\n  \"permissions\": {\n    \"deny\": [\n      \"Bash(curl * | sh)\"\n    ]\n  }\n}\n",
		},
		{
			name:  "does not duplicate an existing rule",
			input: `{"permissions":{"deny":["Bash(curl * | sh)"]}}`,
			want:  "{\n  \"permissions\": {\n    \"deny\": [\n      \"Bash(curl * | sh)\"\n    ]\n  }\n}\n",
		},
		{name: "deny is not an array", input: `{"permissions":{"deny":"Bash(x)"}}`, wantErr: "not an array"},
		{name: "permissions is not an object", input: `{"permissions":[]}`, wantErr: "not an object"},
		{name: "not an object", input: `[]`, wantErr: "expected a JSON object"},
		{name: "trailing data", input: `{} {}`, wantErr: "unexpected data"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := addDenyRules([]byte(tt.input), []string{"Bash(curl * | sh)"})
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestApplyAutoFixes_ReportsFixFailures(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".claude"), 0o755); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"permissions":{"deny":"Bash(x)"}}`)
	settingsPath := filepath.Join(dir, ".claude", "settings.json")
	if err := os.WriteFile(settingsPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	results := []CheckResult{
		{
			Name:        "deny_rule_missing",
			Status:      StatusFail,
			Message:     "Required deny rule missing: Bash(rm -rf /)",
			AutoFixable: true,
			Metadata:    map[string]string{"rule": "Bash(rm -rf /)"},
		},
		{
			Name:        "file_exists_gone.txt",
			Status:      StatusFail,
			Message:     "Generated file gone.txt has been deleted",
			AutoFixable: true,
			Metadata:    map[string]string{"file": "gone.txt"},
		},
	}
	regen := func(_ string) (map[string]types.GeneratedFile, error) {
		return nil, fmt.Errorf("generation failed")
	}

	updated := ApplyAutoFixes(results, dir, "", regen)

	for _, r := range updated {
		if r.Status != StatusFail || !strings.Contains(r.Message, "auto-fix failed") {
			t.Errorf("%s: status %s, message %q; want failed with reason", r.Name, r.Status, r.Message)
		}
	}
	after, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(original) {
		t.Errorf("malformed settings.json must be left untouched, got %s", after)
	}
}

func TestApplyAutoFixes_RecordsRestoredFilesInState(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	stateFile := filepath.Join(dir, ".qsdev", "state.yaml")
	old := state.RecordFiles([]types.GeneratedFile{
		{Path: "kept.txt", Content: []byte("kept")},
		{Path: "restored.json", Content: []byte("old"), Strategy: types.ThreeWayMerge},
	})
	if err := state.SaveStateToFile(stateFile, old); err != nil {
		t.Fatal(err)
	}

	results := []CheckResult{{
		Name:        "file_exists_restored.json",
		Status:      StatusFail,
		AutoFixable: true,
		Metadata:    map[string]string{"file": "restored.json"},
	}}
	fresh := []byte(`{"fresh":true}`)
	regen := func(_ string) (map[string]types.GeneratedFile, error) {
		return map[string]types.GeneratedFile{
			"restored.json": {Path: "restored.json", Content: fresh, Strategy: types.ThreeWayMerge},
		}, nil
	}

	updated := ApplyAutoFixes(results, dir, stateFile, regen)
	if updated[0].Status != StatusPass {
		t.Fatalf("restore failed: %s", updated[0].Message)
	}

	got, err := state.LoadStateFromFile(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	fs := got.Files["restored.json"]
	if fs.Hash != state.ComputeHash(fresh) {
		t.Errorf("state hash not updated for restored file")
	}
	if string(fs.BaseContent) != string(fresh) {
		t.Errorf("three-way-merge base not recorded: %q", fs.BaseContent)
	}
	if got.Files["kept.txt"].Hash != old.Files["kept.txt"].Hash {
		t.Errorf("unrelated state entry changed")
	}
}

// TestApplyAutoFixes_RefusesSymlinkEscape verifies auto-fix never writes
// through a committed symlink that leaves the project: a .claude directory
// linked to the user's global Claude config keeps its settings.json, and a
// deleted file restored into a symlinked directory is not written outside.
func TestApplyAutoFixes_RefusesSymlinkEscape(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on Windows")
	}

	const globalSettings = `{"permissions": {"deny": []}}`
	dir := t.TempDir()
	outside := t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "settings.json"), []byte(globalSettings), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(dir, ".claude")); err != nil {
		t.Fatal(err)
	}

	results := []CheckResult{
		{
			Name: "deny_rule_missing", Status: StatusFail, AutoFixable: true,
			Metadata: map[string]string{"rule": `Bash(git push --force *)`},
		},
		{
			Name: "file_exists_hook", Status: StatusFail, AutoFixable: true,
			Metadata: map[string]string{"file": ".claude/hooks/guard.sh"},
		},
	}
	regen := func(_ string) (map[string]types.GeneratedFile, error) {
		return map[string]types.GeneratedFile{
			".claude/hooks/guard.sh": {Path: ".claude/hooks/guard.sh", Content: []byte("#!/bin/sh\n"), SkipValidation: true},
		}, nil
	}

	updated := ApplyAutoFixes(results, dir, "", regen)
	for _, r := range updated {
		if r.Status == StatusPass {
			t.Errorf("%s: fix through a symlink leaving the project reported success", r.Name)
		}
	}
	if data, err := os.ReadFile(filepath.Join(outside, "settings.json")); err != nil || string(data) != globalSettings {
		t.Errorf("global settings.json = %q (err %v), want it untouched", data, err)
	}
	if _, err := os.Stat(filepath.Join(outside, "hooks")); !os.IsNotExist(err) {
		t.Errorf("restored hook was written outside the project (stat err %v)", err)
	}
}
