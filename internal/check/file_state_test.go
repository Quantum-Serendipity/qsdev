package check

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestCheckFileState_AllFilesMatch(t *testing.T) {
	dir := t.TempDir()

	// Create a file.
	content := []byte("hello world")
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, content, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create state tracking that file.
	genState := state.RecordFiles([]types.GeneratedFile{
		{
			Path:    "test.txt",
			Content: content,
			Mode:    0o644,
		},
	})

	stateFile := filepath.Join(dir, ".devinit", ".qsdev-init-state.yaml")
	if err := state.SaveStateToFile(stateFile, genState); err != nil {
		t.Fatal(err)
	}

	ctx := CheckContext{
		ProjectRoot: dir,
		StateFile:   stateFile,
	}

	results := checkGeneratedFiles(ctx)

	for _, r := range results {
		if r.Status == StatusFail {
			t.Errorf("unexpected failure: %s: %s", r.Name, r.Message)
		}
	}

	// Should have a passing result.
	hasPass := false
	for _, r := range results {
		if r.Status == StatusPass {
			hasPass = true
			break
		}
	}
	if !hasPass {
		t.Error("expected at least one passing result")
	}
}

func TestCheckFileState_ModifiedFile(t *testing.T) {
	dir := t.TempDir()

	originalContent := []byte("original")
	testFile := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(testFile, originalContent, 0o644); err != nil {
		t.Fatal(err)
	}

	genState := state.RecordFiles([]types.GeneratedFile{
		{
			Path:    "test.txt",
			Content: originalContent,
			Mode:    0o644,
		},
	})

	// Now modify the file.
	if err := os.WriteFile(testFile, []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}

	stateFile := filepath.Join(dir, ".devinit", ".qsdev-init-state.yaml")
	if err := state.SaveStateToFile(stateFile, genState); err != nil {
		t.Fatal(err)
	}

	ctx := CheckContext{
		ProjectRoot: dir,
		StateFile:   stateFile,
	}

	results := checkGeneratedFiles(ctx)

	hasFail := false
	for _, r := range results {
		if r.Status == StatusFail && r.Severity == SeverityMedium {
			hasFail = true
			break
		}
	}
	if !hasFail {
		t.Error("expected a medium-severity failure for modified file")
	}
}

func TestCheckFileState_ModifiedUserEditableStrategy(t *testing.T) {
	strategies := []types.MergeStrategy{
		types.ManualMerge,
		types.SectionMarker,
		types.ThreeWayMerge,
	}

	for _, strat := range strategies {
		t.Run(strat.String(), func(t *testing.T) {
			dir := t.TempDir()

			originalContent := []byte("original")
			testFile := filepath.Join(dir, "test.txt")
			if err := os.WriteFile(testFile, originalContent, 0o644); err != nil {
				t.Fatal(err)
			}

			genState := state.RecordFiles([]types.GeneratedFile{
				{Path: "test.txt", Content: originalContent, Mode: 0o644, Strategy: strat},
			})

			if err := os.WriteFile(testFile, []byte("modified by user"), 0o644); err != nil {
				t.Fatal(err)
			}

			stateFile := filepath.Join(dir, ".devinit", ".qsdev-init-state.yaml")
			if err := state.SaveStateToFile(stateFile, genState); err != nil {
				t.Fatal(err)
			}

			results := checkGeneratedFiles(CheckContext{ProjectRoot: dir, StateFile: stateFile})

			for _, r := range results {
				if r.Status == StatusFail {
					t.Errorf("strategy %s: should not fail for user-editable file, got: %s", strat, r.Message)
				}
			}
		})
	}
}

func TestCheckFileState_DeletedFile(t *testing.T) {
	dir := t.TempDir()

	originalContent := []byte("original")

	genState := state.RecordFiles([]types.GeneratedFile{
		{
			Path:    "deleted.txt",
			Content: originalContent,
			Mode:    0o644,
		},
	})

	// Don't create the file — simulate deletion.

	stateFile := filepath.Join(dir, ".devinit", ".qsdev-init-state.yaml")
	if err := state.SaveStateToFile(stateFile, genState); err != nil {
		t.Fatal(err)
	}

	ctx := CheckContext{
		ProjectRoot: dir,
		StateFile:   stateFile,
	}

	results := checkGeneratedFiles(ctx)

	hasFail := false
	for _, r := range results {
		if r.Status == StatusFail && r.Severity == SeverityHigh {
			hasFail = true
			break
		}
	}
	if !hasFail {
		t.Error("expected a high-severity failure for deleted file")
	}
}

func TestCheckFileState_NoStateFile(t *testing.T) {
	dir := t.TempDir()

	ctx := CheckContext{
		ProjectRoot: dir,
		StateFile:   "",
	}

	results := checkGeneratedFiles(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusSkip {
		t.Errorf("Status = %s, want %s", results[0].Status, StatusSkip)
	}
}

func TestCheckDenyRules_AllPresent(t *testing.T) {
	dir := t.TempDir()

	// Create .claude/settings.json with all required deny rules.
	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	settings := map[string]any{
		"permissions": map[string]any{
			"deny": []string{
				`Bash(rm -rf *)`,
				`Bash(git push --force *)`,
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := CheckContext{
		ProjectRoot: dir,
		RequiredDenyRules: []string{
			`Bash(rm -rf *)`,
			`Bash(git push --force *)`,
		},
	}

	results := checkDenyRules(ctx)

	for _, r := range results {
		if r.Status == StatusFail {
			t.Errorf("unexpected failure: %s: %s", r.Name, r.Message)
		}
	}
}

func TestCheckDenyRules_MissingRules(t *testing.T) {
	dir := t.TempDir()

	claudeDir := filepath.Join(dir, ".claude")
	if err := os.MkdirAll(claudeDir, 0o755); err != nil {
		t.Fatal(err)
	}

	settings := map[string]any{
		"permissions": map[string]any{
			"deny": []string{
				`Bash(rm -rf *)`,
			},
		},
	}
	data, _ := json.MarshalIndent(settings, "", "  ")
	settingsPath := filepath.Join(claudeDir, "settings.json")
	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	ctx := CheckContext{
		ProjectRoot: dir,
		RequiredDenyRules: []string{
			`Bash(rm -rf *)`,
			`Bash(git push --force *)`,
		},
	}

	results := checkDenyRules(ctx)

	hasMissingFail := false
	for _, r := range results {
		if r.Status == StatusFail && r.Name == "deny_rule_missing" && r.AutoFixable {
			hasMissingFail = true
			break
		}
	}
	if !hasMissingFail {
		t.Error("expected an auto-fixable failure for missing deny rule")
	}
}

func TestCheckDenyRules_NoSettingsFile(t *testing.T) {
	dir := t.TempDir()

	ctx := CheckContext{
		ProjectRoot:       dir,
		RequiredDenyRules: []string{`Bash(rm -rf *)`},
	}

	results := checkDenyRules(ctx)

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusWarn {
		t.Errorf("Status = %s, want %s", results[0].Status, StatusWarn)
	}
}

func TestCheckDenyRules_EmptyRequired(t *testing.T) {
	dir := t.TempDir()

	ctx := CheckContext{
		ProjectRoot:       dir,
		RequiredDenyRules: nil,
	}

	results := checkDenyRules(ctx)

	if len(results) != 0 {
		t.Errorf("expected 0 results for empty required rules, got %d", len(results))
	}
}
