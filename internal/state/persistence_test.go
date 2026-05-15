package state

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
	"gopkg.in/yaml.v3"
)

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.yaml")

	original := types.GeneratedState{
		LastRun:             time.Date(2025, 6, 15, 10, 30, 0, 0, time.UTC),
		TemplateVersion:     "v1.2.3",
		SkillLibraryVersion: "v0.5.0",
		Files: map[string]types.FileState{
			"devenv.nix": {
				Hash:     "sha256:abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789",
				Strategy: types.Overwrite,
				Mode:     0o644,
			},
			"script.sh": {
				Hash:     "sha256:1111111111111111111111111111111111111111111111111111111111111111",
				Strategy: types.Skip,
				Mode:     0o755,
			},
		},
	}

	if err := SaveStateToFile(path, original); err != nil {
		t.Fatalf("SaveStateToFile: %v", err)
	}

	loaded, err := LoadStateFromFile(path)
	if err != nil {
		t.Fatalf("LoadStateFromFile: %v", err)
	}

	if !loaded.LastRun.Equal(original.LastRun) {
		t.Errorf("LastRun: got %v, want %v", loaded.LastRun, original.LastRun)
	}
	if loaded.TemplateVersion != original.TemplateVersion {
		t.Errorf("TemplateVersion: got %q, want %q", loaded.TemplateVersion, original.TemplateVersion)
	}
	if loaded.SkillLibraryVersion != original.SkillLibraryVersion {
		t.Errorf("SkillLibraryVersion: got %q, want %q", loaded.SkillLibraryVersion, original.SkillLibraryVersion)
	}
	if len(loaded.Files) != len(original.Files) {
		t.Fatalf("Files count: got %d, want %d", len(loaded.Files), len(original.Files))
	}

	for path, origFS := range original.Files {
		loadedFS, ok := loaded.Files[path]
		if !ok {
			t.Errorf("missing file %q after round-trip", path)
			continue
		}
		if loadedFS.Hash != origFS.Hash {
			t.Errorf("file %q hash: got %q, want %q", path, loadedFS.Hash, origFS.Hash)
		}
		if loadedFS.Strategy != origFS.Strategy {
			t.Errorf("file %q strategy: got %v, want %v", path, loadedFS.Strategy, origFS.Strategy)
		}
		if loadedFS.Mode != origFS.Mode {
			t.Errorf("file %q mode: got %o, want %o", path, loadedFS.Mode, origFS.Mode)
		}
	}
}

func TestLoadNonExistentFile(t *testing.T) {
	state, err := LoadStateFromFile("/nonexistent/path/state.yaml")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if state.Files == nil {
		t.Fatal("expected initialized Files map, got nil")
	}
	if len(state.Files) != 0 {
		t.Fatalf("expected empty Files map, got %d entries", len(state.Files))
	}
}

func TestLoadCorruptedYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("{{{{not yaml at all!!!!"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadStateFromFile(path)
	if err == nil {
		t.Fatal("expected error for corrupted YAML, got nil")
	}
}

func TestLoadMissingFilesKey(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "partial.yaml")
	content := []byte("last_run: 2025-01-01T00:00:00Z\ntemplate_version: v1.0.0\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	state, err := LoadStateFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if state.Files == nil {
		t.Fatal("expected initialized Files map, got nil")
	}
	if len(state.Files) != 0 {
		t.Fatalf("expected empty Files map, got %d entries", len(state.Files))
	}
}

func TestSavedYAMLStructure(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.yaml")

	state := types.GeneratedState{
		LastRun:         time.Date(2025, 3, 1, 12, 0, 0, 0, time.UTC),
		TemplateVersion: "v2.0.0",
		Files: map[string]types.FileState{
			"config.yaml": {
				Hash:     "sha256:aabbccdd",
				Strategy: types.Merge,
				Mode:     0o644,
			},
		},
	}

	if err := SaveStateToFile(path, state); err != nil {
		t.Fatalf("SaveStateToFile: %v", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(raw)

	// Verify key names are present in the YAML.
	for _, key := range []string{"last_run:", "template_version:", "files:", "hash:", "strategy:", "mode:"} {
		if !strings.Contains(content, key) {
			t.Errorf("expected YAML to contain %q, got:\n%s", key, content)
		}
	}

	// Verify strategy is serialized as string, not integer.
	if strings.Contains(content, "strategy: 2") {
		t.Errorf("strategy should be a string, not an integer, in YAML:\n%s", content)
	}
	if !strings.Contains(content, "strategy: merge") {
		t.Errorf("expected 'strategy: merge' in YAML, got:\n%s", content)
	}

	// Verify it can be parsed as raw YAML with expected structure.
	var raw2 map[string]interface{}
	if err := yaml.Unmarshal(raw, &raw2); err != nil {
		t.Fatalf("re-parsing saved YAML: %v", err)
	}
	if _, ok := raw2["files"]; !ok {
		t.Error("expected top-level 'files' key in YAML")
	}
}
