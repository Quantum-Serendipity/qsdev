package cmdutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCheckNotExists_FileExists(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "existing.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := CheckNotExists(dir, "existing.txt", false)
	if err == nil {
		t.Error("expected error when file exists and force=false")
	}
}

func TestCheckNotExists_FileDoesNotExist(t *testing.T) {
	dir := t.TempDir()

	err := CheckNotExists(dir, "nonexistent.txt", false)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestCheckNotExists_ForceOverride(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "existing.txt")
	if err := os.WriteFile(f, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := CheckNotExists(dir, "existing.txt", true)
	if err != nil {
		t.Errorf("expected no error when force=true, got: %v", err)
	}
}
