package fileutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCopyFile(t *testing.T) {
	t.Parallel()

	src := filepath.Join(t.TempDir(), "src.txt")
	dst := filepath.Join(t.TempDir(), "dst.txt")

	content := []byte("hello, world\n")
	if err := os.WriteFile(src, content, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CopyFile(src, dst, 0o644); err != nil {
		t.Fatalf("CopyFile() error: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading destination: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("destination content = %q, want %q", got, content)
	}
}

func TestCopyFilePermissions(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("Windows does not support Unix file permission bits")
	}

	src := filepath.Join(t.TempDir(), "src.sh")
	dst := filepath.Join(t.TempDir(), "dst.sh")

	if err := os.WriteFile(src, []byte("#!/bin/sh\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CopyFile(src, dst, 0o755); err != nil {
		t.Fatalf("CopyFile() error: %v", err)
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("stat destination: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o755 {
		t.Errorf("destination permissions = %o, want 755", perm)
	}
}

func TestCopyFileMissingSource(t *testing.T) {
	t.Parallel()

	dst := filepath.Join(t.TempDir(), "dst.txt")

	err := CopyFile("/nonexistent/source/file", dst, 0o644)
	if err == nil {
		t.Fatal("expected error for missing source, got nil")
	}
}

func TestCopyFileBadDestinationDir(t *testing.T) {
	t.Parallel()

	src := filepath.Join(t.TempDir(), "src.txt")
	if err := os.WriteFile(src, []byte("data"), 0o644); err != nil {
		t.Fatal(err)
	}

	dst := filepath.Join(t.TempDir(), "nonexistent", "sub", "dst.txt")

	err := CopyFile(src, dst, 0o644)
	if err == nil {
		t.Fatal("expected error for bad destination directory, got nil")
	}
}

func TestCopyFileCleansUpOnFailure(t *testing.T) {
	t.Parallel()

	dst := filepath.Join(t.TempDir(), "dst.txt")

	err := CopyFile("/nonexistent/source", dst, 0o644)
	if err == nil {
		t.Fatal("expected error, got nil")
	}

	if _, statErr := os.Stat(dst); statErr == nil {
		t.Error("destination file should not exist after failed copy")
	}
}

func TestCopyFileOverwrite(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	if err := os.WriteFile(src, []byte("new content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, []byte("old content that is longer"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CopyFile(src, dst, 0o644); err != nil {
		t.Fatalf("CopyFile() error: %v", err)
	}

	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("reading destination: %v", err)
	}
	if string(got) != "new content" {
		t.Errorf("destination content = %q, want %q", got, "new content")
	}
}

func TestCopyFileEmptyFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	src := filepath.Join(dir, "empty.txt")
	dst := filepath.Join(dir, "dst.txt")

	if err := os.WriteFile(src, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := CopyFile(src, dst, 0o644); err != nil {
		t.Fatalf("CopyFile() error: %v", err)
	}

	info, err := os.Stat(dst)
	if err != nil {
		t.Fatalf("stat destination: %v", err)
	}
	if info.Size() != 0 {
		t.Errorf("destination size = %d, want 0", info.Size())
	}
}
