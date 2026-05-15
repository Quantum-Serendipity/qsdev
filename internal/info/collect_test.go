package info

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestCollectInfo_FullProject(t *testing.T) {
	dir := t.TempDir()

	// Write .qsdev.yaml.
	qsdevYAML := `version: 1
languages:
  - name: go
    version: "1.26"
  - name: python
    version: "3.12"
security:
  level: enhanced
`
	if err := os.WriteFile(filepath.Join(dir, ".qsdev.yaml"), []byte(qsdevYAML), 0o644); err != nil {
		t.Fatalf("writing .qsdev.yaml: %v", err)
	}

	// Write state file.
	devinitDir := filepath.Join(dir, ".devinit")
	if err := os.MkdirAll(devinitDir, 0o755); err != nil {
		t.Fatalf("creating .devinit dir: %v", err)
	}

	genState := types.GeneratedState{
		LastRun:     time.Date(2026, 5, 14, 10, 0, 0, 0, time.UTC),
		QsdevVersion: "1.2.3",
		Files: map[string]types.FileState{
			"devenv.nix":   {Hash: "abc123"},
			"devenv.yaml":  {Hash: "def456"},
			".envrc":       {Hash: "ghi789"},
		},
		EnabledTools: map[string]bool{
			"attach-guard":      true,
			"agent-postmortem":  true,
			"changelog":         false,
		},
	}
	statePath := filepath.Join(devinitDir, ".qsdev-init-state.yaml")
	if err := state.SaveStateToFile(statePath, genState); err != nil {
		t.Fatalf("saving state: %v", err)
	}

	// Write answers file.
	answersYAML := `project_name: my-project
claude_code: true
compliance_level: strict
languages:
  - name: go
    version: "1.26"
`
	if err := os.WriteFile(filepath.Join(devinitDir, ".qsdev-init-answers.yaml"), []byte(answersYAML), 0o644); err != nil {
		t.Fatalf("writing answers: %v", err)
	}

	info, err := CollectInfo(dir)
	if err != nil {
		t.Fatalf("CollectInfo: %v", err)
	}

	// Verify fields.
	if info.ProjectName != "my-project" {
		t.Errorf("ProjectName = %q, want %q", info.ProjectName, "my-project")
	}
	if len(info.Ecosystems) != 2 {
		t.Errorf("len(Ecosystems) = %d, want 2", len(info.Ecosystems))
	} else {
		if info.Ecosystems[0] != "go" {
			t.Errorf("Ecosystems[0] = %q, want %q", info.Ecosystems[0], "go")
		}
		if info.Ecosystems[1] != "python" {
			t.Errorf("Ecosystems[1] = %q, want %q", info.Ecosystems[1], "python")
		}
	}
	if info.SecurityProfile != "enhanced" {
		t.Errorf("SecurityProfile = %q, want %q", info.SecurityProfile, "enhanced")
	}
	if info.QsdevVersion != "1.2.3" {
		t.Errorf("QsdevVersion = %q, want %q", info.QsdevVersion, "1.2.3")
	}
	if info.ConfigVersion != 1 {
		t.Errorf("ConfigVersion = %d, want 1", info.ConfigVersion)
	}
	if info.ManagedFileCount != 3 {
		t.Errorf("ManagedFileCount = %d, want 3", info.ManagedFileCount)
	}
	if info.ActiveToolCount != 2 {
		t.Errorf("ActiveToolCount = %d, want 2", info.ActiveToolCount)
	}
	if !info.ClaudeCodeEnabled {
		t.Error("ClaudeCodeEnabled = false, want true")
	}
	if info.LastUpdated.IsZero() {
		t.Error("LastUpdated is zero, expected non-zero")
	}
}

func TestCollectInfo_NoGdevYaml(t *testing.T) {
	dir := t.TempDir()

	_, err := CollectInfo(dir)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err != ErrNotQsdevProject {
		t.Errorf("error = %v, want ErrNotQsdevProject", err)
	}
}

func TestCollectInfo_MissingState(t *testing.T) {
	dir := t.TempDir()

	// Write minimal .qsdev.yaml.
	qsdevYAML := `version: 1
languages:
  - name: rust
security:
  level: strict
`
	if err := os.WriteFile(filepath.Join(dir, ".qsdev.yaml"), []byte(qsdevYAML), 0o644); err != nil {
		t.Fatalf("writing .qsdev.yaml: %v", err)
	}

	info, err := CollectInfo(dir)
	if err != nil {
		t.Fatalf("CollectInfo: %v", err)
	}

	// Should still populate from config.
	if info.SecurityProfile != "strict" {
		t.Errorf("SecurityProfile = %q, want %q", info.SecurityProfile, "strict")
	}
	if len(info.Ecosystems) != 1 || info.Ecosystems[0] != "rust" {
		t.Errorf("Ecosystems = %v, want [rust]", info.Ecosystems)
	}
	if info.ManagedFileCount != 0 {
		t.Errorf("ManagedFileCount = %d, want 0", info.ManagedFileCount)
	}
	if info.ActiveToolCount != 0 {
		t.Errorf("ActiveToolCount = %d, want 0", info.ActiveToolCount)
	}
	if info.LastUpdated.IsZero() == false {
		t.Error("LastUpdated should be zero when state is missing")
	}
}

func TestCollectInfo_MissingAnswers(t *testing.T) {
	dir := t.TempDir()

	// Write .qsdev.yaml.
	qsdevYAML := `version: 1
languages:
  - name: javascript
`
	if err := os.WriteFile(filepath.Join(dir, ".qsdev.yaml"), []byte(qsdevYAML), 0o644); err != nil {
		t.Fatalf("writing .qsdev.yaml: %v", err)
	}

	// Write state but no answers.
	devinitDir := filepath.Join(dir, ".devinit")
	if err := os.MkdirAll(devinitDir, 0o755); err != nil {
		t.Fatalf("creating .devinit dir: %v", err)
	}
	genState := types.GeneratedState{
		LastRun:     time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		QsdevVersion: "0.9.0",
		Files: map[string]types.FileState{
			"devenv.nix": {Hash: "aaa"},
		},
	}
	if err := state.SaveStateToFile(filepath.Join(devinitDir, ".qsdev-init-state.yaml"), genState); err != nil {
		t.Fatalf("saving state: %v", err)
	}

	info, err := CollectInfo(dir)
	if err != nil {
		t.Fatalf("CollectInfo: %v", err)
	}

	// Project name should fall back to directory basename.
	if info.ProjectName != filepath.Base(dir) {
		t.Errorf("ProjectName = %q, want %q", info.ProjectName, filepath.Base(dir))
	}
	if info.ClaudeCodeEnabled {
		t.Errorf("ClaudeCodeEnabled = %v, want false (no answers file)", info.ClaudeCodeEnabled)
	}
	if info.QsdevVersion != "0.9.0" {
		t.Errorf("QsdevVersion = %q, want %q", info.QsdevVersion, "0.9.0")
	}
	if info.ManagedFileCount != 1 {
		t.Errorf("ManagedFileCount = %d, want 1", info.ManagedFileCount)
	}
}

func TestCollectInfo_DefaultProjectName(t *testing.T) {
	dir := t.TempDir()

	// Write minimal .qsdev.yaml with no answers file at all.
	qsdevYAML := `version: 1
`
	if err := os.WriteFile(filepath.Join(dir, ".qsdev.yaml"), []byte(qsdevYAML), 0o644); err != nil {
		t.Fatalf("writing .qsdev.yaml: %v", err)
	}

	info, err := CollectInfo(dir)
	if err != nil {
		t.Fatalf("CollectInfo: %v", err)
	}

	expected := filepath.Base(dir)
	if info.ProjectName != expected {
		t.Errorf("ProjectName = %q, want %q (directory basename)", info.ProjectName, expected)
	}
	if info.SecurityProfile != "standard" {
		t.Errorf("SecurityProfile = %q, want %q (default)", info.SecurityProfile, "standard")
	}
}
