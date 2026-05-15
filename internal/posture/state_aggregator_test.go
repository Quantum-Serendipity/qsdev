package posture

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
	"gopkg.in/yaml.v3"
)

func TestLoadAllStates_AllPresent(t *testing.T) {
	root := t.TempDir()

	// Create all three state files with distinct content.
	states := map[string]types.GeneratedState{
		".devinit/.qsdev-init-state.yaml": {
			QsdevVersion: "1.0.0",
			LastRun:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
			Files: map[string]types.FileState{
				"devenv.nix": {Hash: "abc123"},
			},
		},
		".devenv/.qsdev-state.yaml": {
			QsdevVersion: "1.1.0",
			LastRun:     time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC),
			Files: map[string]types.FileState{
				".envrc":    {Hash: "def456"},
				"devenv.nix": {Hash: "ghi789"}, // overrides devinit's entry
			},
			EnabledTools: map[string]bool{
				"golangci-lint": true,
			},
		},
		".claude/.qsdev-claude-state.yaml": {
			QsdevVersion: "1.2.0",
			LastRun:     time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC),
			Files: map[string]types.FileState{
				".claude/settings.json": {Hash: "jkl012"},
			},
			EnabledTools: map[string]bool{
				"claude-code":   true,
				"golangci-lint": false, // overrides devenv's entry
			},
		},
	}

	for relPath, st := range states {
		writeState(t, root, relPath, st)
	}

	merged := LoadAllStates(root)

	// Should have loaded all three sources.
	if len(merged.Sources) != 3 {
		t.Fatalf("expected 3 sources, got %d: %v", len(merged.Sources), merged.Sources)
	}

	// Should have no errors.
	if len(merged.Errors) != 0 {
		t.Fatalf("expected 0 errors, got %d: %v", len(merged.Errors), merged.Errors)
	}

	// QsdevVersion should be from the last loaded file.
	if merged.QsdevVersion != "1.2.0" {
		t.Errorf("QsdevVersion = %q, want %q", merged.QsdevVersion, "1.2.0")
	}

	// Files should be merged with later entries winning.
	if len(merged.Files) != 3 {
		t.Errorf("expected 3 merged files, got %d", len(merged.Files))
	}
	if fs, ok := merged.Files["devenv.nix"]; !ok || fs.Hash != "ghi789" {
		t.Errorf("devenv.nix hash = %q, want %q (from .devenv state)", fs.Hash, "ghi789")
	}
	if _, ok := merged.Files[".envrc"]; !ok {
		t.Error("expected .envrc in merged files")
	}
	if _, ok := merged.Files[".claude/settings.json"]; !ok {
		t.Error("expected .claude/settings.json in merged files")
	}

	// EnabledTools should be merged with later entries winning.
	if !merged.EnabledTools["claude-code"] {
		t.Error("expected claude-code to be enabled")
	}
	if merged.EnabledTools["golangci-lint"] {
		t.Error("expected golangci-lint to be disabled (overridden by claude state)")
	}

	if !merged.HasAnyState() {
		t.Error("HasAnyState() should be true")
	}
}

func TestLoadAllStates_OneMissing(t *testing.T) {
	root := t.TempDir()

	// Only create devinit and devenv state files — claude is missing.
	writeState(t, root, ".devinit/.qsdev-init-state.yaml", types.GeneratedState{
		QsdevVersion: "1.0.0",
		LastRun:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Files: map[string]types.FileState{
			"devenv.nix": {Hash: "aaa"},
		},
	})
	writeState(t, root, ".devenv/.qsdev-state.yaml", types.GeneratedState{
		QsdevVersion: "1.0.0",
		LastRun:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Files: map[string]types.FileState{
			".envrc": {Hash: "bbb"},
		},
	})

	merged := LoadAllStates(root)

	if len(merged.Sources) != 2 {
		t.Errorf("expected 2 sources, got %d: %v", len(merged.Sources), merged.Sources)
	}
	if len(merged.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(merged.Errors))
	}
	if len(merged.Files) != 2 {
		t.Errorf("expected 2 files, got %d", len(merged.Files))
	}
	if !merged.HasAnyState() {
		t.Error("HasAnyState() should be true")
	}
}

func TestLoadAllStates_OneCorrupt(t *testing.T) {
	root := t.TempDir()

	// Create a valid state file.
	writeState(t, root, ".devinit/.qsdev-init-state.yaml", types.GeneratedState{
		QsdevVersion: "1.0.0",
		LastRun:     time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
		Files: map[string]types.FileState{
			"devenv.nix": {Hash: "valid"},
		},
	})

	// Create a corrupt YAML file.
	corruptPath := filepath.Join(root, ".devenv")
	if err := os.MkdirAll(corruptPath, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(corruptPath, ".qsdev-state.yaml"), []byte("{{not: valid: yaml: ["), 0o644); err != nil {
		t.Fatal(err)
	}

	merged := LoadAllStates(root)

	// Should have loaded the valid file.
	if len(merged.Sources) != 1 {
		t.Errorf("expected 1 source, got %d: %v", len(merged.Sources), merged.Sources)
	}

	// Should have recorded the corruption error.
	if len(merged.Errors) != 1 {
		t.Fatalf("expected 1 error, got %d", len(merged.Errors))
	}
	loadErr := merged.Errors[0]
	if loadErr.Path != ".devenv/.qsdev-state.yaml" {
		t.Errorf("error path = %q, want %q", loadErr.Path, ".devenv/.qsdev-state.yaml")
	}

	// The error message from StateLoadError should include the path.
	errStr := loadErr.Error()
	if errStr == "" {
		t.Error("StateLoadError.Error() should not be empty")
	}

	// Valid state should still be present.
	if fs, ok := merged.Files["devenv.nix"]; !ok || fs.Hash != "valid" {
		t.Error("expected valid devenv.nix file in merged state")
	}

	if !merged.HasAnyState() {
		t.Error("HasAnyState() should be true despite corruption")
	}
}

func TestLoadAllStates_NonePresent(t *testing.T) {
	root := t.TempDir()

	merged := LoadAllStates(root)

	if len(merged.Sources) != 0 {
		t.Errorf("expected 0 sources, got %d", len(merged.Sources))
	}
	if len(merged.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(merged.Errors))
	}
	if len(merged.Files) != 0 {
		t.Errorf("expected 0 files, got %d", len(merged.Files))
	}
	if merged.QsdevVersion != "" {
		t.Errorf("QsdevVersion = %q, want empty", merged.QsdevVersion)
	}
	if merged.HasAnyState() {
		t.Error("HasAnyState() should be false when no state files exist")
	}
}

// writeState is a test helper that marshals a GeneratedState to YAML and
// writes it to the given relative path under root.
func writeState(t *testing.T, root, relPath string, st types.GeneratedState) {
	t.Helper()
	absPath := filepath.Join(root, relPath)
	dir := filepath.Dir(absPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("creating directory %s: %v", dir, err)
	}
	data, err := yaml.Marshal(&st)
	if err != nil {
		t.Fatalf("marshaling state: %v", err)
	}
	if err := os.WriteFile(absPath, data, 0o644); err != nil {
		t.Fatalf("writing state file %s: %v", absPath, err)
	}
}
