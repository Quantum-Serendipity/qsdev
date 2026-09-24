package state

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestSaveProjectState verifies project state is written under the project
// root and never through a committed symlink that points outside it.
func TestSaveProjectState(t *testing.T) {
	t.Parallel()
	st := types.GeneratedState{Files: map[string]types.FileState{"devenv.nix": {Hash: "sha256:00"}}}

	t.Run("writes under the root", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		if err := SaveProjectState(root, InitStateFile(), st); err != nil {
			t.Fatalf("SaveProjectState: %v", err)
		}
		loaded, err := LoadStateFromFile(filepath.Join(root, filepath.FromSlash(InitStateFile())))
		if err != nil {
			t.Fatalf("LoadStateFromFile: %v", err)
		}
		if _, ok := loaded.Files["devenv.nix"]; !ok {
			t.Errorf("loaded state = %+v, want the saved file entry", loaded)
		}
	})

	escapes := []struct {
		name string
		rel  string
	}{
		{"traversing path", "../state.yaml"},
		{"symlinked state directory", InitStateFile()},
	}
	for _, tt := range escapes {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if runtime.GOOS == "windows" {
				t.Skip("symlink creation requires privileges on Windows")
			}
			root := filepath.Join(t.TempDir(), "project")
			outside := t.TempDir()
			if err := os.MkdirAll(root, 0o755); err != nil {
				t.Fatal(err)
			}
			stateDir := filepath.Dir(filepath.Join(root, filepath.FromSlash(InitStateFile())))
			if err := os.MkdirAll(filepath.Dir(stateDir), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(outside, stateDir); err != nil {
				t.Fatal(err)
			}

			err := SaveProjectState(root, tt.rel, st)
			if !errors.Is(err, fileutil.ErrOutsideRoot) {
				t.Fatalf("err = %v, want ErrOutsideRoot", err)
			}
			entries, readErr := os.ReadDir(outside)
			if readErr != nil || len(entries) != 0 {
				t.Errorf("outside dir has %d entries (err %v), want none", len(entries), readErr)
			}
		})
	}
}
