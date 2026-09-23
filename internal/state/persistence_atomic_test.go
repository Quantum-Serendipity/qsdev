package state

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestSaveStateToFile_AtomicReplace verifies the state file is written via the
// shared atomic writer: parent directories are created, an existing file is
// replaced, the result has the standard mode, and no temp files are left over.
func TestSaveStateToFile_AtomicReplace(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "state", "state.yaml")

	first := types.GeneratedState{Files: map[string]types.FileState{"a": {Hash: "sha256:01"}}}
	second := types.GeneratedState{Files: map[string]types.FileState{"b": {Hash: "sha256:02"}}}

	if err := SaveStateToFile(path, first); err != nil {
		t.Fatalf("first save: %v", err)
	}
	if err := SaveStateToFile(path, second); err != nil {
		t.Fatalf("second save: %v", err)
	}

	loaded, err := LoadStateFromFile(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if _, ok := loaded.Files["b"]; !ok || len(loaded.Files) != 1 {
		t.Fatalf("expected replaced state with only %q, got %v", "b", loaded.Files)
	}

	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if got := info.Mode().Perm(); got != fileutil.ModeReadWrite {
			t.Errorf("state file mode = %o, want %o", got, fileutil.ModeReadWrite)
		}
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "state.yaml" {
			t.Errorf("unexpected leftover file %q in state directory", e.Name())
		}
	}
}

// TestLoadProjectStates_CorruptFileDoesNotHideOthers verifies one unreadable
// state file is reported without discarding the states that did load.
func TestLoadProjectStates_CorruptFileDoesNotHideOthers(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	corrupt := filepath.Join(root, InitStateFile())
	if err := os.MkdirAll(filepath.Dir(corrupt), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(corrupt, []byte("files: [not: a map"), 0o644); err != nil {
		t.Fatal(err)
	}
	valid := types.GeneratedState{Files: map[string]types.FileState{"devenv.nix": {Hash: "sha256:01"}}}
	if err := SaveStateToFile(filepath.Join(root, DevenvStateFile()), valid); err != nil {
		t.Fatal(err)
	}

	states, err := LoadProjectStates(root)
	if err == nil {
		t.Error("expected an error for the corrupt state file")
	}
	if len(states) != 1 {
		t.Fatalf("loaded %d states, want 1 (the valid devenv state)", len(states))
	}
	if _, ok := states[0].Files["devenv.nix"]; !ok {
		t.Errorf("valid state not loaded: %v", states[0].Files)
	}
}
