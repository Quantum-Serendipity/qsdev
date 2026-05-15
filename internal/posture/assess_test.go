package posture

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
	"gopkg.in/yaml.v3"
)

func TestAssess_UninitializedProject(t *testing.T) {
	// An empty temp directory with no state files or .qsdev.yaml.
	root := t.TempDir()

	_, err := Assess(root, AssessOptions{})
	if err == nil {
		t.Fatal("expected error for uninitialized project, got nil")
	}
	if !errors.Is(err, ErrNotInitialized) {
		t.Errorf("expected ErrNotInitialized, got: %v", err)
	}
}

func TestAssess_WithGdevYAML(t *testing.T) {
	root := t.TempDir()

	// Create .qsdev.yaml to indicate the project is initialized.
	qsdevYAML := filepath.Join(root, ".qsdev.yaml")
	if err := os.WriteFile(qsdevYAML, []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := Assess(root, AssessOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.SchemaVersion != SchemaVersion {
		t.Errorf("SchemaVersion = %q, want %q", report.SchemaVersion, SchemaVersion)
	}
	if report.ProjectName != filepath.Base(root) {
		t.Errorf("ProjectName = %q, want %q", report.ProjectName, filepath.Base(root))
	}
	if report.ProjectPath != root {
		t.Errorf("ProjectPath = %q, want %q", report.ProjectPath, root)
	}
	if report.GeneratedAt.IsZero() {
		t.Error("GeneratedAt should not be zero")
	}
}

func TestAssess_WithStateFiles(t *testing.T) {
	root := t.TempDir()

	// Create a state file.
	st := types.GeneratedState{
		QsdevVersion: "2.0.0",
		LastRun:     time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
		Files: map[string]types.FileState{
			"devenv.nix": {Hash: "abc123"},
		},
	}
	dir := filepath.Join(root, ".devinit")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	data, err := yaml.Marshal(&st)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".qsdev-init-state.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := Assess(root, AssessOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.QsdevVersion != "2.0.0" {
		t.Errorf("QsdevVersion = %q, want %q", report.QsdevVersion, "2.0.0")
	}
}

func TestAssess_NonexistentPath(t *testing.T) {
	_, err := Assess("/nonexistent/path/that/should/not/exist", AssessOptions{})
	if err == nil {
		t.Fatal("expected error for nonexistent path, got nil")
	}
}

func TestAssess_FileNotDirectory(t *testing.T) {
	tmp := t.TempDir()
	filePath := filepath.Join(tmp, "afile")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Assess(filePath, AssessOptions{})
	if err == nil {
		t.Fatal("expected error for file path, got nil")
	}
}

func TestAssess_EmptySlicesNotNil(t *testing.T) {
	root := t.TempDir()

	// Create .qsdev.yaml.
	if err := os.WriteFile(filepath.Join(root, ".qsdev.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := Assess(root, AssessOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify all slices are non-nil (important for JSON serialization as [] not null).
	if report.Tools == nil {
		t.Error("Tools should be empty slice, not nil")
	}
	if report.Ecosystems == nil {
		t.Error("Ecosystems should be empty slice, not nil")
	}
	if report.Defense.Layers == nil {
		t.Error("Defense.Layers should be empty slice, not nil")
	}
	if report.Config.Files == nil {
		t.Error("Config.Files should be empty slice, not nil")
	}
	if report.Dependencies.Ecosystems == nil {
		t.Error("Dependencies.Ecosystems should be empty slice, not nil")
	}
	if report.Drift.Categories == nil {
		t.Error("Drift.Categories should be empty slice, not nil")
	}
	if report.Drift.BySeverity == nil {
		t.Error("Drift.BySeverity should be empty map, not nil")
	}
	if report.Conformance.Baseline.Checks == nil {
		t.Error("Conformance.Baseline.Checks should be empty slice, not nil")
	}
	if report.Conformance.Enhanced.Checks == nil {
		t.Error("Conformance.Enhanced.Checks should be empty slice, not nil")
	}
}

func TestAssess_CorruptStateRecordedAsDrift(t *testing.T) {
	root := t.TempDir()

	// Create .qsdev.yaml so project is considered initialized.
	if err := os.WriteFile(filepath.Join(root, ".qsdev.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a corrupt state file.
	dir := filepath.Join(root, ".devenv")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".qsdev-state.yaml"), []byte("{{corrupt"), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := Assess(root, AssessOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Drift.TotalFindings != 1 {
		t.Errorf("expected 1 drift finding, got %d", report.Drift.TotalFindings)
	}
	if len(report.Drift.Categories) != 1 {
		t.Fatalf("expected 1 drift category, got %d", len(report.Drift.Categories))
	}
	if report.Drift.Categories[0].Name != "state-files" {
		t.Errorf("drift category name = %q, want %q", report.Drift.Categories[0].Name, "state-files")
	}
}
