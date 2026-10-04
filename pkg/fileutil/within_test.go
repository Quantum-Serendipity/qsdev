package fileutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestPathWithin pins that PathWithin compares resolved locations: a path
// through a symlink into root is within it, a sibling with a common prefix is
// not, and a path that does not exist yet is judged by its parent.
func TestPathWithin(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	root := filepath.Join(base, "root")
	if err := os.MkdirAll(filepath.Join(root, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name, path string
		want       bool
	}{
		{"root itself", root, true},
		{"below", filepath.Join(root, "sub", "x.yaml"), true},
		{"missing below", filepath.Join(root, "new", "x.yaml"), true},
		{"dot-dot out", filepath.Join(root, "..", "other"), false},
		{"common prefix sibling", root + "x", false},
	}
	if runtime.GOOS != "windows" {
		link := filepath.Join(base, "link")
		if err := os.Symlink(root, link); err != nil {
			t.Fatal(err)
		}
		tests = append(tests, struct {
			name, path string
			want       bool
		}{"through symlink", filepath.Join(link, "x.yaml"), true})
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := PathWithin(root, tt.path); got != tt.want {
				t.Errorf("PathWithin(%q, %q) = %v, want %v", root, tt.path, got, tt.want)
			}
		})
	}
}
