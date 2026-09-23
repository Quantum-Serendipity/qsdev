package fileutil_test

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

func TestReadFirstLine(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"simple", "3.12.1\n", "3.12.1"},
		{"leading blank lines skipped", "\n  \n3.12\n", "3.12"},
		{"trimmed", "  v20  \n", "v20"},
		{"empty file", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "f")
			if err := os.WriteFile(path, []byte(tt.content), 0o644); err != nil {
				t.Fatal(err)
			}
			if got := fileutil.ReadFirstLine(path); got != tt.want {
				t.Errorf("ReadFirstLine = %q, want %q", got, tt.want)
			}
			got, err := fileutil.ReadFirstLineErr(path)
			if err != nil || got != tt.want {
				t.Errorf("ReadFirstLineErr = %q, %v; want %q, nil", got, err, tt.want)
			}
		})
	}
}

func TestReadFirstLineErr_ReportsErrors(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	if _, err := fileutil.ReadFirstLineErr(filepath.Join(dir, "missing")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("missing file error = %v, want ErrNotExist", err)
	}

	long := filepath.Join(dir, "long")
	if err := os.WriteFile(long, []byte(strings.Repeat("x", bufio.MaxScanTokenSize+1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := fileutil.ReadFirstLineErr(long); !errors.Is(err, bufio.ErrTooLong) {
		t.Errorf("over-long line error = %v, want bufio.ErrTooLong", err)
	}
	if got := fileutil.ReadFirstLine(long); got != "" {
		t.Errorf("ReadFirstLine over-long = %q, want empty", got)
	}
}

func TestFileAndDirExists(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name        string
		parts       []string
		file, isDir bool
	}{
		{"file", []string{dir, "f"}, true, false},
		{"dir", []string{dir}, false, true},
		{"missing", []string{dir, "nope"}, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := fileutil.FileExists(tt.parts...); got != tt.file {
				t.Errorf("FileExists = %v, want %v", got, tt.file)
			}
			if got := fileutil.DirExists(tt.parts...); got != tt.isDir {
				t.Errorf("DirExists = %v, want %v", got, tt.isDir)
			}
		})
	}
}
