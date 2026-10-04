package logging

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// makeTree creates the given relative paths under root; a trailing "/" makes
// a directory, anything else a small file.
func makeTree(t *testing.T, root string, paths ...string) {
	t.Helper()
	for _, rel := range paths {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if rel[len(rel)-1] == '/' {
			if err := os.MkdirAll(p, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("version: 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// isAnyMarker reports whether dir holds any project marker, including a data
// directory that would only be ignored when dir is the (real) home directory.
func isAnyMarker(dir string) bool {
	b := branding.Get()
	return fileutil.FileExists(dir, b.ConfigFile) || fileutil.DirExists(dir, b.StateDir) ||
		fileutil.DirExists(dir, "."+b.AppName)
}

// walkRoot returns a temp directory no marker above which can leak into a
// walk-up, even when the default temp dir sits inside the real user profile.
func walkRoot(t *testing.T) string {
	t.Helper()
	return testutil.MarkerFreeTempDir(t, isAnyMarker)
}

// setHome points the user's home directory at home: HOME for Unix and
// USERPROFILE, which os.UserHomeDir reads instead on Windows.
func setHome(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

// TestFindProjectRoot pins the shared project marker set: config file, state
// directory, or project data directory — the last one never in the user's
// home, where the same name is the per-user global data directory — and the
// repository ceiling: the walk never climbs above the first directory holding
// a .git entry.
func TestFindProjectRoot(t *testing.T) {
	// Not parallel: t.Setenv changes HOME for the process.
	tests := []struct {
		name   string
		tree   []string
		home   string // relative to the temp root; "" = a directory outside it
		start  string
		want   string
		wantOK bool
	}{
		{name: "no marker", tree: []string{"a/b/"}, start: "a/b"},
		{name: "config file in ancestor", tree: []string{".qsdev.yaml", "src/pkg/"}, start: "src/pkg", want: ".", wantOK: true},
		{name: "config name as directory is not a marker", tree: []string{".qsdev.yaml/", "src/"}, start: "src"},
		{name: "state dir in ancestor", tree: []string{".devinit/", "src/"}, start: "src", want: ".", wantOK: true},
		{name: "data dir in ancestor", tree: []string{".qsdev/", "src/"}, start: "src", want: ".", wantOK: true},
		{name: "data dir in home is not a marker", tree: []string{".qsdev/logs/", "code/app/"}, home: ".", start: "code/app"},
		{name: "config file in home is a marker", tree: []string{".qsdev.yaml", "code/"}, home: ".", start: "code", want: ".", wantOK: true},
		{name: "project below home found before home", tree: []string{".qsdev/", "code/.qsdev/", "code/src/"}, home: ".", start: "code/src", want: "code", wantOK: true},
		{name: "nearest project wins", tree: []string{".qsdev.yaml", "sub/.devinit/", "sub/x/"}, start: "sub/x", want: "sub", wantOK: true},
		{name: "marker above .git dir not found", tree: []string{".qsdev.yaml", "repo/.git/", "repo/src/"}, start: "repo/src"},
		{name: "marker above .git file (worktree) not found", tree: []string{".qsdev/", "wt/.git", "wt/src/"}, start: "wt/src"},
		{name: "config file at git toplevel found from subdir", tree: []string{"repo/.git/", "repo/.qsdev.yaml", "repo/a/b/"}, start: "repo/a/b", want: "repo", wantOK: true},
		{name: "state dir between start and git toplevel found", tree: []string{".qsdev.yaml", "repo/.git/", "repo/pkg/.devinit/", "repo/pkg/x/"}, start: "repo/pkg/x", want: "repo/pkg", wantOK: true},
		{name: "nested repo is its own boundary", tree: []string{"repo/.git/", "repo/.qsdev.yaml", "repo/sub/.git", "repo/sub/src/"}, start: "repo/sub/src"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := walkRoot(t)
			makeTree(t, root, tt.tree...)
			home := t.TempDir()
			if tt.home != "" {
				home = filepath.Join(root, filepath.FromSlash(tt.home))
			}
			setHome(t, home)

			got, ok := FindProjectRoot(filepath.Join(root, filepath.FromSlash(tt.start)))
			want := ""
			if tt.wantOK {
				want = filepath.Join(root, filepath.FromSlash(tt.want))
			}
			if got != want || ok != tt.wantOK {
				t.Errorf("FindProjectRoot() = (%q, %v), want (%q, %v)", got, ok, want, tt.wantOK)
			}
		})
	}
}

// TestFindProjectRoot_UnreadableGitEntryStops covers the fail-closed ceiling:
// a directory whose .git entry cannot be inspected (Lstat fails with an error
// other than not-exist) stops the walk, so an ancestor marker is not adopted.
func TestFindProjectRoot_UnreadableGitEntryStops(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory search permission is a Unix concept")
	}
	if os.Geteuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}
	root := walkRoot(t)
	makeTree(t, root, ".qsdev.yaml", "locked/")
	locked := filepath.Join(root, "locked")
	// Read without search permission: Lstat(locked/.git) fails with EACCES.
	if err := os.Chmod(locked, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	setHome(t, t.TempDir())

	if _, err := os.Lstat(filepath.Join(locked, ".git")); err == nil || errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("precondition: Lstat(.git) error = %v, want a non-not-exist error", err)
	}
	if got, ok := FindProjectRoot(locked); ok {
		t.Errorf("FindProjectRoot() = %q, want no project: an unreadable .git entry is a ceiling", got)
	}
}

// TestFindProjectRoot_SymlinkedHome covers a home directory reached through a symlink
// while the start directory is the resolved path, as os.Getwd reports it.
func TestFindProjectRoot_SymlinkedHome(t *testing.T) {
	realHome := walkRoot(t)
	makeTree(t, realHome, ".qsdev/logs/", "code/")
	link := filepath.Join(t.TempDir(), "home")
	if err := os.Symlink(realHome, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	setHome(t, link)

	if got, ok := FindProjectRoot(filepath.Join(realHome, "code")); ok {
		t.Errorf("FindProjectRoot() = %q, want no project: the home data dir is not a marker", got)
	}
}

func TestDetectProjectRoot(t *testing.T) {
	root, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	makeTree(t, root, ".qsdev.yaml", "internal/pkg/")
	setHome(t, t.TempDir())
	t.Chdir(filepath.Join(root, "internal", "pkg"))

	if got := DetectProjectRoot(); got != root {
		t.Errorf("DetectProjectRoot() = %q, want %q", got, root)
	}
}
