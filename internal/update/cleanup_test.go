package update

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

func TestCleanupSidecar_Exists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "devenv.nix.new")

	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := CleanupSidecar(dir, "devenv.nix.new"); err != nil {
		t.Fatalf("CleanupSidecar returned error: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected file to be removed after CleanupSidecar")
	}
}

func TestCleanupSidecar_NotExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	if err := CleanupSidecar(dir, "nonexistent.nix.new"); err != nil {
		t.Fatalf("CleanupSidecar on nonexistent file returned error: %v", err)
	}
}

// TestCleanupSidecarRefusesGitRedirect verifies a sidecar is never removed
// through a directory symlinked into .git: the call fails with
// ErrRedirectsIntoGit and the .git file survives.
func TestCleanupSidecarRefusesGitRedirect(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	victim := filepath.Join(root, ".git", "nix", "devenv.nix.new")
	if err := os.MkdirAll(filepath.Dir(victim), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(victim, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	testutil.SymlinkOrSkip(t, filepath.Join(".git", "nix"), filepath.Join(root, "nix"))

	err := CleanupSidecar(root, filepath.Join("nix", "devenv.nix.new"))
	if !errors.Is(err, fileutil.ErrRedirectsIntoGit) {
		t.Errorf("CleanupSidecar error = %v, want ErrRedirectsIntoGit", err)
	}
	if _, err := os.Stat(victim); err != nil {
		t.Errorf("%s was removed: %v", victim, err)
	}
}
