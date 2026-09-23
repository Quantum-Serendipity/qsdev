//go:build !windows

package fileutil_test

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

func writeTestFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(path, mode); err != nil {
		t.Fatal(err)
	}
}

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	return string(data)
}

// TestWriteFileAtomic_WritesThroughSymlink guards the CLAUDE.md -> AGENTS.md
// layout: the link must survive and the target must receive the content.
func TestWriteFileAtomic_WritesThroughSymlink(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	target := filepath.Join(dir, "AGENTS.md")
	link := filepath.Join(dir, "CLAUDE.md")
	writeTestFile(t, target, "orig", 0o600)
	if err := os.Symlink("AGENTS.md", link); err != nil {
		t.Fatal(err)
	}

	if err := fileutil.WriteFileAtomic(link, []byte("new"), fileutil.ModeReadWrite); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Error("symlink was replaced by a regular file")
	}
	if got := readTestFile(t, target); got != "new" {
		t.Errorf("target content = %q, want %q", got, "new")
	}
}

// TestWriteFileAtomic_DoesNotWriteThroughEscapingSymlink verifies a link that
// points outside its directory is replaced rather than followed, so an
// unconfined write never modifies a file elsewhere on disk.
func TestWriteFileAtomic_DoesNotWriteThroughEscapingSymlink(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	outside := filepath.Join(base, "outside.md")
	writeTestFile(t, outside, "keep", 0o644)
	link := filepath.Join(base, "project", "CLAUDE.md")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, link); err != nil {
		t.Fatal(err)
	}

	if err := fileutil.WriteFileAtomic(link, []byte("new"), fileutil.ModeReadWrite); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}
	if got := readTestFile(t, outside); got != "keep" {
		t.Errorf("file outside the link's directory was modified: %q", got)
	}
	if got := readTestFile(t, link); got != "new" {
		t.Errorf("link path content = %q, want %q", got, "new")
	}
}

// TestWriteFileAtomic_DoesNotWriteThroughLinkIntoGitDir verifies a committed
// link cannot redirect a rewrite into repository metadata such as .git/config.
func TestWriteFileAtomic_DoesNotWriteThroughLinkIntoGitDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	gitConfig := filepath.Join(dir, ".git", "config")
	writeTestFile(t, gitConfig, "[core]\n", 0o644)
	link := filepath.Join(dir, "CLAUDE.md")
	if err := os.Symlink(filepath.Join(".git", "config"), link); err != nil {
		t.Fatal(err)
	}

	if err := fileutil.WriteFileAtomic(link, []byte("new"), fileutil.ModeReadWrite); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}
	if got := readTestFile(t, gitConfig); got != "[core]\n" {
		t.Errorf(".git/config was modified through a symlink: %q", got)
	}
	if got := readTestFile(t, link); got != "new" {
		t.Errorf("link path content = %q, want %q", got, "new")
	}
}

// TestWriteFileAtomic_DoesNotWidenExistingMode verifies a user-tightened file
// is not made group/world readable by a rewrite.
func TestWriteFileAtomic_DoesNotWidenExistingMode(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		existing  os.FileMode
		requested os.FileMode
		want      os.FileMode
	}{
		{"private stays private", 0o600, 0o644, 0o600},
		{"group-only read kept", 0o640, 0o644, 0o640},
		{"exec bit follows request", 0o644, 0o755, 0o755},
		{"narrowing honoured", 0o644, 0o600, 0o600},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(t.TempDir(), "f")
			writeTestFile(t, path, "old", tt.existing)
			if err := fileutil.WriteFileAtomic(path, []byte("new"), tt.requested); err != nil {
				t.Fatalf("WriteFileAtomic: %v", err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Mode().Perm(); got != tt.want {
				t.Errorf("mode = %o, want %o", got, tt.want)
			}
		})
	}
}

// TestWriteFileAtomic_RespectsUmask is not parallel: it changes the
// process-wide umask.
func TestWriteFileAtomic_RespectsUmask(t *testing.T) {
	old := syscall.Umask(0o077)
	defer syscall.Umask(old)

	path := filepath.Join(t.TempDir(), "new.yaml")
	if err := fileutil.WriteFileAtomic(path, []byte("x"), fileutil.ModeReadWrite); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("mode = %o, want 0600 under umask 077", got)
	}
}

func TestWriteFileAtomicInRoot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		// setup builds the layout under root (with a sibling outside dir)
		// and returns the relative path to write.
		setup       func(t *testing.T, root, outside string) string
		wantErr     error
		wantContent map[string]string // path relative to root or outside -> content
	}{
		{
			name:        "plain nested write",
			setup:       func(_ *testing.T, _, _ string) string { return filepath.Join(".claude", "settings.json") },
			wantContent: map[string]string{"root:.claude/settings.json": "new"},
		},
		{
			name: "symlinked parent escaping root is rejected",
			setup: func(t *testing.T, root, outside string) string {
				t.Helper()
				if err := os.Symlink(outside, filepath.Join(root, ".claude")); err != nil {
					t.Fatal(err)
				}
				return filepath.Join(".claude", "settings.json")
			},
			wantErr: fileutil.ErrOutsideRoot,
		},
		{
			name: "symlinked file escaping root is rejected",
			setup: func(t *testing.T, root, outside string) string {
				t.Helper()
				writeTestFile(t, filepath.Join(outside, "global.md"), "keep", 0o644)
				if err := os.Symlink(filepath.Join(outside, "global.md"), filepath.Join(root, "CLAUDE.md")); err != nil {
					t.Fatal(err)
				}
				return "CLAUDE.md"
			},
			wantErr:     fileutil.ErrOutsideRoot,
			wantContent: map[string]string{"outside:global.md": "keep"},
		},
		{
			name: "symlinked file inside root is written through",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				writeTestFile(t, filepath.Join(root, "AGENTS.md"), "orig", 0o644)
				if err := os.Symlink("AGENTS.md", filepath.Join(root, "CLAUDE.md")); err != nil {
					t.Fatal(err)
				}
				return "CLAUDE.md"
			},
			wantContent: map[string]string{"root:AGENTS.md": "new"},
		},
		{
			name: "symlinked file into .git is rejected",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				writeTestFile(t, filepath.Join(root, ".git", "config"), "keep", 0o644)
				if err := os.Symlink(filepath.Join(".git", "config"), filepath.Join(root, "CLAUDE.md")); err != nil {
					t.Fatal(err)
				}
				return "CLAUDE.md"
			},
			wantErr:     fileutil.ErrOutsideRoot,
			wantContent: map[string]string{"root:.git/config": "keep"},
		},
		{
			name: "symlinked parent into .git is rejected",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				if err := os.MkdirAll(filepath.Join(root, ".git", "hooks"), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(filepath.Join(".git", "hooks"), filepath.Join(root, ".claude")); err != nil {
					t.Fatal(err)
				}
				return filepath.Join(".claude", "pre-commit")
			},
			wantErr: fileutil.ErrOutsideRoot,
		},
		{
			name: "explicit .git path is allowed",
			setup: func(t *testing.T, root, _ string) string {
				t.Helper()
				if err := os.MkdirAll(filepath.Join(root, ".git", "info"), 0o755); err != nil {
					t.Fatal(err)
				}
				return filepath.Join(".git", "info", "exclude")
			},
			wantContent: map[string]string{"root:.git/info/exclude": "new"},
		},
		{
			name:    "non-local path is rejected",
			setup:   func(_ *testing.T, _, _ string) string { return filepath.Join("..", "escape.txt") },
			wantErr: fileutil.ErrOutsideRoot,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			base := t.TempDir()
			root := filepath.Join(base, "project")
			outside := filepath.Join(base, "outside")
			for _, d := range []string{root, outside} {
				if err := os.MkdirAll(d, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			rel := tt.setup(t, root, outside)

			err := fileutil.WriteFileAtomicInRoot(root, rel, []byte("new"), fileutil.ModeReadWrite)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("WriteFileAtomicInRoot error = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr != nil {
				entries, _ := os.ReadDir(outside)
				for _, e := range entries {
					if e.Name() != "global.md" {
						t.Errorf("unexpected file written outside root: %s", e.Name())
					}
				}
			}
			for key, want := range tt.wantContent {
				dir, rel, _ := strings.Cut(key, ":")
				base := root
				if dir == "outside" {
					base = outside
				}
				if got := readTestFile(t, filepath.Join(base, rel)); got != want {
					t.Errorf("%s = %q, want %q", key, got, want)
				}
			}
		})
	}
}
