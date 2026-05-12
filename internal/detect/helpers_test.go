package detect

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFileExists(t *testing.T) {
	dir := t.TempDir()

	// Create a regular file.
	path := filepath.Join(dir, "exists.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !fileExists(path) {
		t.Error("fileExists should return true for an existing file")
	}
	if fileExists(dir, "nope.txt") {
		t.Error("fileExists should return false for a missing file")
	}
	// A directory is not a file.
	if fileExists(dir) {
		t.Error("fileExists should return false for a directory")
	}
}

func TestDirExists(t *testing.T) {
	dir := t.TempDir()

	subdir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a regular file.
	path := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(path, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}

	if !dirExists(subdir) {
		t.Error("dirExists should return true for an existing directory")
	}
	if dirExists(path) {
		t.Error("dirExists should return false for a regular file")
	}
	if dirExists(dir, "missing") {
		t.Error("dirExists should return false for a missing path")
	}
}

func TestReadFirstLine(t *testing.T) {
	dir := t.TempDir()

	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"simple", "hello world\nsecond line\n", "hello world"},
		{"leading blank lines", "\n\n  content  \n", "content"},
		{"single line no newline", "v22.1.0", "v22.1.0"},
		{"empty file", "", ""},
		{"only whitespace", "  \n  \n", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(dir, tt.name+".txt")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			got := readFirstLine(path)
			if got != tt.want {
				t.Errorf("readFirstLine(%q) = %q, want %q", tt.content, got, tt.want)
			}
		})
	}

	// Missing file returns empty string.
	if got := readFirstLine(filepath.Join(dir, "missing")); got != "" {
		t.Errorf("readFirstLine(missing) = %q, want empty", got)
	}
}
