package posture

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/state"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestDetectFileModification_Unmodified(t *testing.T) {
	dir := t.TempDir()
	content := []byte("original content\n")
	writeFile(t, filepath.Join(dir, "config.toml"), string(content))

	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"config.toml": {
				Hash:     state.ComputeHash(content),
				Strategy: types.Overwrite,
				Mode:     0o644,
			},
		},
	}

	cat := detectFileModification(dir, genState)

	if len(cat.Findings) != 0 {
		t.Errorf("expected zero findings for unmodified file, got %d: %+v", len(cat.Findings), cat.Findings)
	}
}

func TestDetectFileModification_MachineOwnedModified(t *testing.T) {
	strategies := []types.MergeStrategy{types.Overwrite, types.LibraryManaged}

	for _, strategy := range strategies {
		t.Run(strategy.String(), func(t *testing.T) {
			dir := t.TempDir()
			originalContent := []byte("original\n")
			writeFile(t, filepath.Join(dir, "managed.yml"), "modified content\n")

			genState := types.GeneratedState{
				Files: map[string]types.FileState{
					"managed.yml": {
						Hash:     state.ComputeHash(originalContent),
						Strategy: strategy,
						Mode:     0o644,
					},
				},
			}

			cat := detectFileModification(dir, genState)

			if len(cat.Findings) != 1 {
				t.Fatalf("expected 1 finding, got %d", len(cat.Findings))
			}

			f := cat.Findings[0]
			if f.Severity != DriftWarning {
				t.Errorf("expected severity %q, got %q", DriftWarning, f.Severity)
			}
			if !f.AutoFixable {
				t.Error("expected autoFixable to be true for machine-owned file")
			}
			if f.Subject != "managed.yml" {
				t.Errorf("expected subject %q, got %q", "managed.yml", f.Subject)
			}
		})
	}
}

func TestDetectFileModification_HumanEditedModified(t *testing.T) {
	strategies := []types.MergeStrategy{types.SectionMarker, types.ThreeWayMerge}

	for _, strategy := range strategies {
		t.Run(strategy.String(), func(t *testing.T) {
			dir := t.TempDir()
			originalContent := []byte("original\n")
			writeFile(t, filepath.Join(dir, "claude.md"), "user edited content\n")

			genState := types.GeneratedState{
				Files: map[string]types.FileState{
					"claude.md": {
						Hash:     state.ComputeHash(originalContent),
						Strategy: strategy,
						Mode:     0o644,
					},
				},
			}

			cat := detectFileModification(dir, genState)

			if len(cat.Findings) != 1 {
				t.Fatalf("expected 1 finding, got %d", len(cat.Findings))
			}

			f := cat.Findings[0]
			if f.Severity != DriftInfo {
				t.Errorf("expected severity %q, got %q", DriftInfo, f.Severity)
			}
			if f.AutoFixable {
				t.Error("expected autoFixable to be false for human-edited file")
			}
		})
	}
}

func TestDetectFileModification_DeletedFile(t *testing.T) {
	dir := t.TempDir()

	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"deleted.txt": {
				Hash:     "abc123",
				Strategy: types.Overwrite,
				Mode:     0o644,
			},
		},
	}

	cat := detectFileModification(dir, genState)

	if len(cat.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(cat.Findings))
	}

	f := cat.Findings[0]
	if f.Severity != DriftError {
		t.Errorf("expected severity %q, got %q", DriftError, f.Severity)
	}
	if f.Subject != "deleted.txt" {
		t.Errorf("expected subject %q, got %q", "deleted.txt", f.Subject)
	}
	if !f.AutoFixable {
		t.Error("expected autoFixable to be true for deleted file")
	}
}

func TestDetectFileModification_EmptyState(t *testing.T) {
	dir := t.TempDir()

	cat := detectFileModification(dir, types.GeneratedState{})

	if len(cat.Findings) != 0 {
		t.Errorf("expected zero findings for empty state, got %d", len(cat.Findings))
	}
}

func TestDetectFileModification_UnreadableFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not enforce POSIX directory permissions")
	}
	if os.Getuid() == 0 {
		t.Skip("test requires non-root to enforce file permission denial")
	}

	dir := t.TempDir()

	// Create a file and then make its parent directory unreadable.
	subDir := filepath.Join(dir, "restricted")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(subDir, "secret.txt"), "content")
	// Make directory unreadable to trigger an error.
	if err := os.Chmod(subDir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		os.Chmod(subDir, 0o755) //nolint:errcheck
	})

	genState := types.GeneratedState{
		Files: map[string]types.FileState{
			"restricted/secret.txt": {
				Hash:     "abc123",
				Strategy: types.Overwrite,
				Mode:     0o644,
			},
		},
	}

	cat := detectFileModification(dir, genState)

	if len(cat.Findings) != 1 {
		t.Fatalf("expected 1 finding, got %d: %+v", len(cat.Findings), cat.Findings)
	}

	f := cat.Findings[0]
	if f.Severity != DriftInfo {
		t.Errorf("expected severity %q for unknown status, got %q", DriftInfo, f.Severity)
	}
}
