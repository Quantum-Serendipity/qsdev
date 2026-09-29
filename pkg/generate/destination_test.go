package generate

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidateDestination(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires elevated privileges on Windows")
	}

	root := t.TempDir()
	outside := t.TempDir()
	mustMkdir := func(p string) {
		t.Helper()
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	mustMkdir(filepath.Join(root, "inside", "dir"))
	if err := os.WriteFile(filepath.Join(outside, "victim.md"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	// .github -> directory outside the project.
	if err := os.Symlink(outside, filepath.Join(root, ".github")); err != nil {
		t.Fatal(err)
	}
	// A file symlink pointing outside.
	if err := os.Symlink(filepath.Join(outside, "victim.md"), filepath.Join(root, "linked.md")); err != nil {
		t.Fatal(err)
	}
	// Dangling symlinks whose targets would be created outside the project.
	if err := os.Symlink(filepath.Join(outside, "missing.md"), filepath.Join(root, "dangling.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(outside, "missing-dir"), filepath.Join(root, ".claude")); err != nil {
		t.Fatal(err)
	}
	// An in-project symlinked directory is fine.
	if err := os.Symlink(filepath.Join(root, "inside"), filepath.Join(root, "alias")); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		rel     string
		wantErr string
	}{
		{"plain new file", "new.txt", ""},
		{"new file in new nested dirs", "a/b/c/new.txt", ""},
		{"existing dir", "inside/dir/x.txt", ""},
		{"in-project symlinked dir", "alias/dir/x.txt", ""},
		{"absolute path", filepath.Join(root, "x"), "must be relative"},
		{"traversal", "../x", "path traversal"},
		{"symlinked dir escapes", ".github/pull_request_template.md", "escapes project root"},
		{"symlinked dir escapes for nested new dirs", ".github/workflows/new/x.yml", "escapes project root"},
		{"symlinked file escapes", "linked.md", "escapes project root"},
		{"dangling file symlink", "dangling.md", "dangling symlink"},
		{"dangling dir symlink", ".claude/hooks/guard.py", "dangling symlink"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateDestination(root, tt.rel)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("want error containing %q, got %v", tt.wantErr, err)
			}
		})
	}
}
