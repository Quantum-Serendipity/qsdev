package update

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCleanupSidecar_Exists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "devenv.nix.new")

	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	if err := CleanupSidecar(path); err != nil {
		t.Fatalf("CleanupSidecar returned error: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("expected file to be removed after CleanupSidecar")
	}
}

func TestCleanupSidecar_NotExists(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent.nix.new")

	if err := CleanupSidecar(path); err != nil {
		t.Fatalf("CleanupSidecar on nonexistent file returned error: %v", err)
	}
}
