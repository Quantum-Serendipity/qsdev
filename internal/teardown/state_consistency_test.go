package teardown

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestStateFilesForTeardown_CoversStateFilePaths guards against the teardown
// list drifting from the canonical state file locations.
func TestStateFilesForTeardown_CoversStateFilePaths(t *testing.T) {
	t.Parallel()

	got := stateFilesForTeardown()
	for _, want := range state.StateFilePaths() {
		if !slices.Contains(got, want) {
			t.Errorf("stateFilesForTeardown() missing state file %q; got %v", want, got)
		}
	}
	if cfg := branding.Get().ConfigFile; !slices.Contains(got, cfg) {
		t.Errorf("stateFilesForTeardown() missing config file %q; got %v", cfg, got)
	}
	// W154: an older release's tracked .claude answers copy (origin URL,
	// username, absolute path) must not survive teardown.
	if legacy := answers.LegacyClaudeCopyFile(); !slices.Contains(got, legacy) {
		t.Errorf("stateFilesForTeardown() missing legacy Claude answers %q; got %v", legacy, got)
	}
	// F349: the committed generated-file manifest describes files teardown
	// removes, so it must go with them.
	if manifest := state.ManifestFile(); !slices.Contains(got, manifest) {
		t.Errorf("stateFilesForTeardown() missing generated-file manifest %q; got %v", manifest, got)
	}
}

// TestClassifyFiles_ModeChangeIsModified verifies teardown agrees with
// state.CheckModified: a chmod-only change marks the file modified, so an
// exclusive file is preserved instead of deleted.
func TestClassifyFiles_ModeChangeIsModified(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("file modes are not compared on Windows")
	}

	tests := []struct {
		name         string
		recordedMode os.FileMode
		diskMode     os.FileMode
		wantModified bool
	}{
		{name: "same mode", recordedMode: 0o644, diskMode: 0o644, wantModified: false},
		{name: "chmod only", recordedMode: 0o644, diskMode: 0o600, wantModified: true},
		{name: "mode not recorded", recordedMode: 0, diskMode: 0o600, wantModified: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			dir := t.TempDir()
			content := []byte("original content")
			absPath := filepath.Join(dir, "exclusive.txt")
			if err := os.WriteFile(absPath, content, tt.diskMode); err != nil {
				t.Fatal(err)
			}
			if err := os.Chmod(absPath, tt.diskMode); err != nil {
				t.Fatal(err)
			}

			genState := types.GeneratedState{
				Files: map[string]types.FileState{
					"exclusive.txt": {
						Hash:  state.ComputeHash(content),
						Mode:  tt.recordedMode,
						Owner: "test-tool",
					},
				},
			}

			classified := ClassifyFiles(genState, dir, testRegistry())
			if len(classified) != 1 {
				t.Fatalf("expected 1 classified file, got %d", len(classified))
			}
			if got := classified[0].Modified; got != tt.wantModified {
				t.Errorf("Modified = %v, want %v", got, tt.wantModified)
			}
		})
	}
}
