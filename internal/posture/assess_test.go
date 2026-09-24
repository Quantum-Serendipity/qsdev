package posture

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
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
			// Every default init writes the catalog's default MCP servers,
			// so they never imply the full tier.
			name:         "default MCP servers keep the default tier",
			yaml:         "version: 1\nclaude_code:\n  mcp_servers:\n    - context7\n    - github\n    - socket\n",
			wantCurrent:  "standard",
			wantPosition: 2,
			wantNext:     "full",
		},
		{
			name:         "inferred from a non-default MCP server",
			yaml:         "version: 1\nclaude_code:\n  mcp_servers:\n    - github\n    - custom-db\n",
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

	if report.Drift.TotalFindings < 1 {
		t.Errorf("expected at least 1 drift finding, got %d", report.Drift.TotalFindings)
	}
	found := false
	for _, cat := range report.Drift.Categories {
		if cat.Name == "state-files" {
			found = true
			if len(cat.Findings) != 1 {
				t.Errorf("expected 1 finding in state-files category, got %d", len(cat.Findings))
			}
			break
		}
	}
	if !found {
		t.Error("expected state-files drift category to be present")
	}
}

// TestAssess_MalformedConfigRecordedAsDrift guards against a config with a
// syntax error silently grading the project against the default tier.
func TestAssess_MalformedConfigRecordedAsDrift(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		yaml        string
		wantFinding bool
	}{
		{name: "valid config", yaml: "version: 1\ntier: full\n", wantFinding: false},
		{name: "yaml syntax error", yaml: "version: 1\ntier: [full\n", wantFinding: true},
		{name: "missing version", yaml: "tier: full\n", wantFinding: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if err := os.WriteFile(filepath.Join(root, ".qsdev.yaml"), []byte(tt.yaml), 0o644); err != nil {
				t.Fatal(err)
			}

			report, err := Assess(root, AssessOptions{})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var got *drift.Finding
			for _, cat := range report.Drift.Categories {
				for i, f := range cat.Findings {
					if cat.Name == StateFilesCategory && f.Subject == ".qsdev.yaml" {
						got = &cat.Findings[i]
					}
				}
			}
			if (got != nil) != tt.wantFinding {
				t.Fatalf("config drift finding = %+v, want present=%v", got, tt.wantFinding)
			}
			if got != nil && got.Severity != drift.Error {
				t.Errorf("config drift severity = %q, want %q", got.Severity, drift.Error)
			}
		})
	}
}

// TestBuildConfigFileInfos_DeletedFileIsMissing guards the wrapped-error bug:
// ComputeFileHash wraps the not-exist error, so a deleted file must still be
// classified missing, not corrupt.
func TestBuildConfigFileInfos_DeletedFileIsMissing(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "present.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(root, "dir.txt"), 0o755); err != nil {
		t.Fatal(err)
	}

	infos := buildConfigFileInfos(root, map[string]types.FileState{
		"gone.txt":    {Hash: "abc"},
		"present.txt": {Hash: "stale"},
		"dir.txt":     {Hash: "abc"}, // unreadable as a file: corrupt
	})

	want := map[string]string{"gone.txt": "missing", "present.txt": "modified", "dir.txt": "corrupt"}
	for _, info := range infos {
		if info.State != want[info.Path] {
			t.Errorf("%s: State = %q, want %q", info.Path, info.State, want[info.Path])
		}
	}
}

// TestAssess_ConfigCountersSumToTotal guards the config health counters: a
// deleted file counts as missing and every state has a counter.
func TestAssess_ConfigCountersSumToTotal(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	writeState(t, root, ".claude/.qsdev-claude-state.yaml", types.GeneratedState{
		QsdevVersion: "1.0.0",
		LastRun:      time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Files: map[string]types.FileState{
			".claude/settings.json": {Hash: "abc"}, // deleted
		},
		EnabledTools: map[string]bool{"attach-guard": true},
	})

	report, err := Assess(root, AssessOptions{})
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}

	cfg := report.Config
	if cfg.Missing != 1 {
		t.Errorf("Missing = %d, want 1", cfg.Missing)
	}
	if sum := cfg.Current + cfg.Modified + cfg.Outdated + cfg.Missing + cfg.Corrupt; sum != cfg.Total {
		t.Errorf("counters sum to %d, want Total %d: %+v", sum, cfg.Total, cfg)
	}
}

// TestBuildEcosystemStatuses_PrefersDedicatedLockfile guards lockfile choice:
// a real lockfile must win over a manifest that doubles as a pin source, so a
// uv project with a loose requirements.txt is scanned through uv.lock.
func TestBuildEcosystemStatuses_PrefersDedicatedLockfile(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		files []string
		want  string
	}{
		{name: "uv.lock over requirements.txt", files: []string{"requirements.txt", "uv.lock"}, want: "uv.lock"},
		{name: "poetry.lock over requirements.txt", files: []string{"requirements.txt", "poetry.lock"}, want: "poetry.lock"},
		{name: "requirements.txt alone", files: []string{"requirements.txt"}, want: "requirements.txt"},
		{name: "nothing", files: nil, want: "missing"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			for _, name := range tt.files {
				if err := os.WriteFile(filepath.Join(root, name), []byte("x"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			detected := types.DetectedProject{Ecosystems: map[string]bool{ecosystem.NamePython: true}}

			statuses := buildEcosystemStatuses(detected, root, nil)

			if len(statuses) != 1 || statuses[0].LockFile != tt.want {
				t.Errorf("statuses = %+v, want python lock file %q", statuses, tt.want)
			}
		})
	}
}
