package repair

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestRepair_NilReport(t *testing.T) {
	result, updatedState, err := Repair("/tmp", RepairOptions{}, types.GeneratedState{}, nil, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result.Fixed)+len(result.Skipped)+len(result.Failed) != 0 {
		t.Error("expected empty result for nil report")
	}
	if updatedState == nil {
		t.Error("expected non-nil state")
	}
}

func TestRepair_DryRun(t *testing.T) {
	root := t.TempDir()

	// Create a file that will be "drifted".
	relPath := ".claude/settings.json"
	absPath := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(absPath, []byte("modified"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	originalContent := []byte("original content")
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			relPath: {
				Hash:     state.ComputeHash(originalContent),
				Strategy: types.Overwrite,
				Mode:     0o644,
			},
		},
	}

	driftReport := &posture.DriftReport{
		Categories: []posture.DriftCategory{
			{
				Name: "File Modification",
				Findings: []posture.DriftFinding{
					{
						Subject:     relPath,
						Description: "Machine-owned file \".claude/settings.json\" has been modified (strategy: overwrite)",
						Severity:    posture.DriftWarning,
					},
				},
			},
		},
	}

	freshFiles := map[string]types.GeneratedFile{
		relPath: {
			Path:     relPath,
			Content:  originalContent,
			Mode:     0o644,
			Strategy: types.Overwrite,
		},
	}

	result, _, err := Repair(root, RepairOptions{DryRun: true}, genState, freshFiles, driftReport)
	if err != nil {
		t.Fatalf("Repair: %v", err)
	}

	if len(result.Fixed) != 1 {
		t.Errorf("got %d fixed, want 1", len(result.Fixed))
	}

	// Verify the file was NOT actually written (dry-run).
	data, _ := os.ReadFile(absPath)
	if string(data) != "modified" {
		t.Error("dry-run should not modify the file")
	}

	// Verify no backup was created.
	backups := backupDir(root)
	if _, err := os.Stat(backups); !os.IsNotExist(err) {
		t.Error("dry-run should not create backups")
	}
}

func TestRepair_FixOverwriteFile(t *testing.T) {
	root := t.TempDir()

	relPath := ".claude/settings.json"
	absPath := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(absPath, []byte("modified by user"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	freshContent := []byte(`{"version": 2}`)
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			relPath: {
				Hash:     state.ComputeHash([]byte("original")),
				Strategy: types.Overwrite,
				Mode:     0o644,
			},
		},
	}

	driftReport := &posture.DriftReport{
		Categories: []posture.DriftCategory{
			{
				Name: "File Modification",
				Findings: []posture.DriftFinding{
					{
						Subject:     relPath,
						Description: "Machine-owned file \".claude/settings.json\" has been modified (strategy: overwrite)",
						Severity:    posture.DriftWarning,
					},
				},
			},
		},
	}

	freshFiles := map[string]types.GeneratedFile{
		relPath: {
			Path:     relPath,
			Content:  freshContent,
			Mode:     0o644,
			Strategy: types.Overwrite,
		},
	}

	result, updatedState, err := Repair(root, RepairOptions{}, genState, freshFiles, driftReport)
	if err != nil {
		t.Fatalf("Repair: %v", err)
	}

	// Verify the file was fixed.
	if len(result.Fixed) != 1 {
		t.Fatalf("got %d fixed, want 1", len(result.Fixed))
	}
	if result.Fixed[0].File != relPath {
		t.Errorf("Fixed[0].File = %q, want %q", result.Fixed[0].File, relPath)
	}

	// Verify file content was restored.
	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading restored file: %v", err)
	}
	if string(data) != string(freshContent) {
		t.Errorf("file content = %q, want %q", string(data), string(freshContent))
	}

	// Verify backup was created.
	if result.Fixed[0].BackupPath == "" {
		t.Error("expected non-empty backup path")
	}
	backupData, err := os.ReadFile(result.Fixed[0].BackupPath)
	if err != nil {
		t.Fatalf("reading backup: %v", err)
	}
	if string(backupData) != "modified by user" {
		t.Errorf("backup content = %q, want %q", string(backupData), "modified by user")
	}

	// Verify state was updated.
	fileState, ok := updatedState.Files[relPath]
	if !ok {
		t.Fatal("expected file in updated state")
	}
	if fileState.Hash != state.ComputeHash(freshContent) {
		t.Errorf("updated hash = %q, want %q", fileState.Hash, state.ComputeHash(freshContent))
	}
}

func TestRepair_DeletedFileRegenerated(t *testing.T) {
	root := t.TempDir()

	relPath := ".envrc"
	freshContent := []byte("# generated .envrc\nuse devenv\n")

	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			relPath: {
				Hash:     state.ComputeHash(freshContent),
				Strategy: types.Overwrite,
				Mode:     0o644,
			},
		},
	}

	driftReport := &posture.DriftReport{
		Categories: []posture.DriftCategory{
			{
				Name: "File Modification",
				Findings: []posture.DriftFinding{
					{
						Subject:     relPath,
						Description: "Generated file \".envrc\" has been deleted",
						Severity:    posture.DriftError,
					},
				},
			},
		},
	}

	freshFiles := map[string]types.GeneratedFile{
		relPath: {
			Path:     relPath,
			Content:  freshContent,
			Mode:     0o644,
			Strategy: types.Overwrite,
		},
	}

	result, _, err := Repair(root, RepairOptions{}, genState, freshFiles, driftReport)
	if err != nil {
		t.Fatalf("Repair: %v", err)
	}

	if len(result.Fixed) != 1 {
		t.Fatalf("got %d fixed, want 1", len(result.Fixed))
	}

	// File should be created (no backup since it was deleted).
	data, err := os.ReadFile(filepath.Join(root, relPath))
	if err != nil {
		t.Fatalf("reading regenerated file: %v", err)
	}
	if string(data) != string(freshContent) {
		t.Errorf("file content = %q, want %q", string(data), string(freshContent))
	}

	// No backup for deleted files.
	if result.Fixed[0].BackupPath != "" {
		t.Errorf("expected no backup for deleted file, got %q", result.Fixed[0].BackupPath)
	}
}

func TestRepair_MissingFreshContent(t *testing.T) {
	root := t.TempDir()

	relPath := ".claude/settings.json"
	absPath := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(absPath, []byte("modified"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			relPath: {
				Hash:     "sha256:abc",
				Strategy: types.Overwrite,
				Mode:     0o644,
			},
		},
	}

	driftReport := &posture.DriftReport{
		Categories: []posture.DriftCategory{
			{
				Name: "File Modification",
				Findings: []posture.DriftFinding{
					{
						Subject:     relPath,
						Description: "Machine-owned file has been modified (strategy: overwrite)",
						Severity:    posture.DriftWarning,
					},
				},
			},
		},
	}

	// No fresh files provided.
	freshFiles := map[string]types.GeneratedFile{}

	result, _, err := Repair(root, RepairOptions{}, genState, freshFiles, driftReport)
	if err != nil {
		t.Fatalf("Repair: %v", err)
	}

	if len(result.Failed) != 1 {
		t.Fatalf("got %d failed, want 1", len(result.Failed))
	}
	if result.Failed[0].Error == nil {
		t.Error("expected non-nil error for missing fresh content")
	}
	if !strings.Contains(result.Failed[0].Error.Error(), "no fresh content") {
		t.Errorf("error = %q, want to contain 'no fresh content'", result.Failed[0].Error.Error())
	}
}

func TestRepair_TargetFile(t *testing.T) {
	root := t.TempDir()

	// Create two drifted files.
	for _, rel := range []string{".npmrc", ".envrc"} {
		if err := os.WriteFile(filepath.Join(root, rel), []byte("modified"), 0o644); err != nil {
			t.Fatalf("write %s: %v", rel, err)
		}
	}

	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			".npmrc": {Hash: "sha256:old", Strategy: types.Overwrite, Mode: 0o644},
			".envrc": {Hash: "sha256:old", Strategy: types.Overwrite, Mode: 0o644},
		},
	}

	driftReport := &posture.DriftReport{
		Categories: []posture.DriftCategory{
			{
				Name: "File Modification",
				Findings: []posture.DriftFinding{
					{Subject: ".npmrc", Description: "Machine-owned file has been modified (strategy: overwrite)", Severity: posture.DriftWarning},
					{Subject: ".envrc", Description: "Machine-owned file has been modified (strategy: overwrite)", Severity: posture.DriftWarning},
				},
			},
		},
	}

	freshFiles := map[string]types.GeneratedFile{
		".npmrc": {Path: ".npmrc", Content: []byte("fresh-npmrc"), Mode: 0o644, Strategy: types.Overwrite},
		".envrc": {Path: ".envrc", Content: []byte("fresh-envrc"), Mode: 0o644, Strategy: types.Overwrite},
	}

	// Only target .npmrc.
	result, _, err := Repair(root, RepairOptions{TargetFile: ".npmrc"}, genState, freshFiles, driftReport)
	if err != nil {
		t.Fatalf("Repair: %v", err)
	}

	if len(result.Fixed) != 1 {
		t.Fatalf("got %d fixed, want 1", len(result.Fixed))
	}
	if result.Fixed[0].File != ".npmrc" {
		t.Errorf("Fixed[0].File = %q, want %q", result.Fixed[0].File, ".npmrc")
	}

	// .envrc should not have been touched.
	envrcData, _ := os.ReadFile(filepath.Join(root, ".envrc"))
	if string(envrcData) != "modified" {
		t.Error(".envrc should not have been modified when targeting .npmrc only")
	}
}

func TestRepair_SkippedActions(t *testing.T) {
	root := t.TempDir()

	driftReport := &posture.DriftReport{
		Categories: []posture.DriftCategory{
			{
				Name: "Version Drift",
				Findings: []posture.DriftFinding{
					{Subject: "qsdev version", Description: "Version mismatch", Severity: posture.DriftInfo},
				},
			},
			{
				Name: "Tool Availability",
				Findings: []posture.DriftFinding{
					{Subject: "semgrep", Description: "Binary missing", Severity: posture.DriftWarning},
				},
			},
		},
	}

	result, _, err := Repair(root, RepairOptions{}, types.GeneratedState{}, nil, driftReport)
	if err != nil {
		t.Fatalf("Repair: %v", err)
	}

	if len(result.Skipped) != 2 {
		t.Errorf("got %d skipped, want 2", len(result.Skipped))
	}
	if len(result.Fixed) != 0 {
		t.Errorf("got %d fixed, want 0", len(result.Fixed))
	}
}

func TestRepair_StateIsolation(t *testing.T) {
	// Verify that the original genState is not mutated.
	root := t.TempDir()

	relPath := ".npmrc"
	if err := os.WriteFile(filepath.Join(root, relPath), []byte("old"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	originalHash := "sha256:original"
	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			relPath: {Hash: originalHash, Strategy: types.Overwrite, Mode: 0o644},
		},
	}

	driftReport := &posture.DriftReport{
		Categories: []posture.DriftCategory{
			{
				Name: "File Modification",
				Findings: []posture.DriftFinding{
					{Subject: relPath, Description: "Machine-owned file has been modified (strategy: overwrite)", Severity: posture.DriftWarning},
				},
			},
		},
	}

	freshFiles := map[string]types.GeneratedFile{
		relPath: {Path: relPath, Content: []byte("fresh"), Mode: 0o644, Strategy: types.Overwrite},
	}

	_, updatedState, err := Repair(root, RepairOptions{}, genState, freshFiles, driftReport)
	if err != nil {
		t.Fatalf("Repair: %v", err)
	}

	// Original state should be unchanged.
	if genState.Files[relPath].Hash != originalHash {
		t.Error("original genState was mutated")
	}

	// Updated state should have new hash.
	if updatedState.Files[relPath].Hash == originalHash {
		t.Error("updatedState should have a new hash")
	}
}

func TestCopyState(t *testing.T) {
	original := types.GeneratedState{
		QsdevVersion: "1.0.0",
		Files: map[string]types.FileState{
			"a.txt": {Hash: "sha256:aaa"},
			"b.txt": {Hash: "sha256:bbb"},
		},
		EnabledTools: map[string]bool{
			"semgrep": true,
		},
	}

	cp := copyState(original)

	// Mutate the copy.
	cp.Files["c.txt"] = types.FileState{Hash: "sha256:ccc"}
	cp.EnabledTools["gitleaks"] = true

	// Original should be unchanged.
	if _, ok := original.Files["c.txt"]; ok {
		t.Error("original Files was mutated")
	}
	if original.EnabledTools["gitleaks"] {
		t.Error("original EnabledTools was mutated")
	}
}

func TestRepair_DevenvNixProtected_WithForce(t *testing.T) {
	root := t.TempDir()

	relPath := "devenv.nix"
	absPath := filepath.Join(root, relPath)
	modifiedContent := []byte("# user modified devenv.nix")
	if err := os.WriteFile(absPath, modifiedContent, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			relPath: {
				Hash:     state.ComputeHash([]byte("original devenv.nix")),
				Strategy: types.Overwrite,
				Mode:     0o644,
			},
		},
	}

	driftReport := &posture.DriftReport{
		Categories: []posture.DriftCategory{
			{
				Name: "File Modification",
				Findings: []posture.DriftFinding{
					{
						Subject:     relPath,
						Description: "Machine-owned file modified",
						Severity:    posture.DriftWarning,
					},
				},
			},
		},
	}

	freshFiles := map[string]types.GeneratedFile{
		relPath: {
			Path:     relPath,
			Content:  []byte("fresh devenv.nix"),
			Mode:     0o644,
			Strategy: types.Overwrite,
		},
	}

	result, _, err := Repair(root, RepairOptions{Force: true}, genState, freshFiles, driftReport)
	if err != nil {
		t.Fatalf("Repair: %v", err)
	}

	if len(result.Fixed) != 0 {
		t.Errorf("got %d fixed, want 0 (devenv.nix should never be auto-modified)", len(result.Fixed))
	}
	if len(result.Skipped) != 1 {
		t.Errorf("got %d skipped, want 1", len(result.Skipped))
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	if string(data) != string(modifiedContent) {
		t.Errorf("devenv.nix was modified; content = %q, want %q", string(data), string(modifiedContent))
	}
}

func TestRepair_DevenvNixProtected_WithReset(t *testing.T) {
	root := t.TempDir()

	relPath := "devenv.nix"
	absPath := filepath.Join(root, relPath)
	modifiedContent := []byte("# user modified devenv.nix")
	if err := os.WriteFile(absPath, modifiedContent, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			relPath: {
				Hash:     state.ComputeHash([]byte("original devenv.nix")),
				Strategy: types.Overwrite,
				Mode:     0o644,
			},
		},
	}

	driftReport := &posture.DriftReport{
		Categories: []posture.DriftCategory{
			{
				Name: "File Modification",
				Findings: []posture.DriftFinding{
					{
						Subject:     relPath,
						Description: "Machine-owned file modified",
						Severity:    posture.DriftWarning,
					},
				},
			},
		},
	}

	freshFiles := map[string]types.GeneratedFile{
		relPath: {
			Path:     relPath,
			Content:  []byte("fresh devenv.nix"),
			Mode:     0o644,
			Strategy: types.Overwrite,
		},
	}

	result, _, err := Repair(root, RepairOptions{Reset: true}, genState, freshFiles, driftReport)
	if err != nil {
		t.Fatalf("Repair: %v", err)
	}

	if len(result.Fixed) != 0 {
		t.Errorf("got %d fixed, want 0 (devenv.nix should never be auto-modified)", len(result.Fixed))
	}
	if len(result.Skipped) != 1 {
		t.Errorf("got %d skipped, want 1", len(result.Skipped))
	}

	data, err := os.ReadFile(absPath)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}
	if string(data) != string(modifiedContent) {
		t.Errorf("devenv.nix was modified; content = %q, want %q", string(data), string(modifiedContent))
	}
}
