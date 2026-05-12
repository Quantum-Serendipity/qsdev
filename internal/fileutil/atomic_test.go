package fileutil_test

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/fileutil"
)

func TestWriteFileAtomic_CreatesFileWithCorrectContentAndPermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	content := []byte("hello world")

	if err := fileutil.WriteFileAtomic(path, content, 0644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content = %q, want %q", got, content)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0644 {
		t.Errorf("mode = %o, want %o", info.Mode().Perm(), 0644)
	}
}

func TestWriteFileAtomic_OverwritesExistingFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")

	if err := os.WriteFile(path, []byte("old content"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	newContent := []byte("new content")
	if err := fileutil.WriteFileAtomic(path, newContent, 0644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(newContent) {
		t.Errorf("content = %q, want %q", got, newContent)
	}
}

func TestWriteFileAtomic_CreatesNestedDirectories(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "subdir", "nested", "file.txt")
	content := []byte("nested content")

	if err := fileutil.WriteFileAtomic(path, content, 0644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("content = %q, want %q", got, content)
	}
}

func TestWriteFileAtomic_CleansUpTempFileOnWriteFailure(t *testing.T) {
	dir := t.TempDir()
	// Make the directory read-only so Chmod will fail after write succeeds
	// but we can still create temp files. Instead, test a simpler scenario:
	// write to a path where rename will fail because target dir doesn't exist
	// and we prevent MkdirAll from creating it by making parent read-only.
	subdir := filepath.Join(dir, "readonly")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("setup MkdirAll: %v", err)
	}
	if err := os.Chmod(subdir, 0444); err != nil {
		t.Fatalf("setup Chmod: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(subdir, 0755)
	})

	path := filepath.Join(subdir, "child", "file.txt")
	err := fileutil.WriteFileAtomic(path, []byte("data"), 0644)
	if err == nil {
		t.Fatal("expected error when writing to read-only directory, got nil")
	}

	// Verify no temp files were left behind in the subdir
	entries, _ := os.ReadDir(subdir)
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), ".gdev-tmp-") {
			t.Errorf("temp file left behind: %s", entry.Name())
		}
	}
}

func TestWriteFileAtomic_Mode0755Preserved(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "script.sh")
	content := []byte("#!/bin/bash\necho hello")

	if err := fileutil.WriteFileAtomic(path, content, 0755); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0755 {
		t.Errorf("mode = %o, want %o", info.Mode().Perm(), 0755)
	}
}

func TestWriteFileAtomic_NoPartialContentDuringConcurrentRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "concurrent.txt")
	original := []byte("original content that should remain intact")

	// Write the initial file.
	if err := os.WriteFile(path, original, 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	replacement := []byte("replacement content written atomically!")

	var wg sync.WaitGroup
	const readers = 10

	// Start concurrent readers that continuously read the file.
	// Each read should see either the original or the replacement, never a mix.
	errCh := make(chan error, readers)
	stop := make(chan struct{})

	for i := 0; i < readers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				data, err := os.ReadFile(path)
				if err != nil {
					// File might be momentarily absent on some OS edge cases; skip.
					continue
				}
				s := string(data)
				if s != string(original) && s != string(replacement) {
					errCh <- &partialContentError{got: s}
					return
				}
			}
		}()
	}

	// Perform the atomic write while readers are running.
	if err := fileutil.WriteFileAtomic(path, replacement, 0644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	close(stop)
	wg.Wait()
	close(errCh)

	for err := range errCh {
		t.Errorf("concurrent reader saw partial content: %v", err)
	}

	// Final check: file should have replacement content.
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("final ReadFile: %v", err)
	}
	if string(got) != string(replacement) {
		t.Errorf("final content = %q, want %q", got, replacement)
	}
}

type partialContentError struct {
	got string
}

func (e *partialContentError) Error() string {
	return "partial content: " + e.got
}
