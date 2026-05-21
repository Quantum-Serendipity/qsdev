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
		LastRun:      time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
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

func TestAssess_TierInfoDefaultStandard(t *testing.T) {
	root := t.TempDir()

	// Create .qsdev.yaml without a security level — should default to standard.
	if err := os.WriteFile(filepath.Join(root, ".qsdev.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	report, err := Assess(root, AssessOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if report.Tier.Current != "standard" {
		t.Errorf("Tier.Current = %q, want %q", report.Tier.Current, "standard")
	}
	if report.Tier.Position != 2 {
		t.Errorf("Tier.Position = %d, want 2", report.Tier.Position)
	}
	if report.Tier.Total != 3 {
		t.Errorf("Tier.Total = %d, want 3", report.Tier.Total)
	}
	if report.Tier.NextTier != "full" {
		t.Errorf("Tier.NextTier = %q, want %q", report.Tier.NextTier, "full")
	}
}

func TestAssess_TierInfoFromConfig(t *testing.T) {
	tests := []struct {
		name         string
		yaml         string
		wantCurrent  string
		wantPosition int
		wantNext     string
	}{
		{
			name:         "explicit tier supply-chain-only",
			yaml:         "version: 1\ntier: supply-chain-only\n",
			wantCurrent:  "supply-chain-only",
			wantPosition: 1,
			wantNext:     "standard",
		},
		{
			name:         "explicit tier standard",
			yaml:         "version: 1\ntier: standard\n",
			wantCurrent:  "standard",
			wantPosition: 2,
			wantNext:     "full",
		},
		{
			name:         "explicit tier full",
			yaml:         "version: 1\ntier: full\n",
			wantCurrent:  "full",
			wantPosition: 3,
			wantNext:     "",
		},
		{
			name:         "inferred from supply-chain-only permission level",
			yaml:         "version: 1\nclaude_code:\n  permission_level: supply-chain-only\n",
			wantCurrent:  "supply-chain-only",
			wantPosition: 1,
			wantNext:     "standard",
		},
		{
			name:         "inferred from MCP servers present",
			yaml:         "version: 1\nclaude_code:\n  mcp_servers:\n    - github\n",
			wantCurrent:  "full",
			wantPosition: 3,
			wantNext:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, ".qsdev.yaml"), []byte(tt.yaml), 0o644); err != nil {
				t.Fatal(err)
			}

			report, err := Assess(root, AssessOptions{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if report.Tier.Current != tt.wantCurrent {
				t.Errorf("Tier.Current = %q, want %q", report.Tier.Current, tt.wantCurrent)
			}
			if report.Tier.Position != tt.wantPosition {
				t.Errorf("Tier.Position = %d, want %d", report.Tier.Position, tt.wantPosition)
			}
			if report.Tier.Total != 3 {
				t.Errorf("Tier.Total = %d, want 3", report.Tier.Total)
			}
			if report.Tier.NextTier != tt.wantNext {
				t.Errorf("Tier.NextTier = %q, want %q", report.Tier.NextTier, tt.wantNext)
			}
		})
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
