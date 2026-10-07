package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// isolatedName is the directory IsolatedDir returns, below its git ceiling.
const isolatedName = "w"

// tmpReleaseWait bounds how long IsolateUserDirs' cleanup waits for its temp
// directory to become removable. Host probes run in os.TempDir
// (procexec.NeutralDir) and, after a probe timeout, a grandchild the kill did
// not reach (a docker CLI plugin) keeps running there. On Windows a process's
// working directory cannot be deleted, so the directory is busy until that
// grandchild finishes: briefly, but longer than testing's own two-second
// retry. A handle qsdev itself leaks never goes away and still fails.
const tmpReleaseWait = 30 * time.Second

// tmpReleasePoll is how often that cleanup retries.
const tmpReleasePoll = 100 * time.Millisecond

// IsolatedDir returns a fresh, empty, symlink-resolved directory that no
// upward project or repository walk can leave: its parent is a minimal git
// repository toplevel, so projectctx.Resolve and git discovery both stop
// there, before any host or planted ancestor. That holds wherever the temp
// directory lives, on every OS (on Windows it sits inside the user profile),
// so a test never needs to search for, or skip without, a clean ancestor.
//
// The returned directory has no .git of its own; tests may plant one, or a
// gitfile, to test ceilings. IsolatedDir changes no process-wide state, so
// parallel tests may call it.
func IsolatedDir(t testing.TB) string {
	t.Helper()
	base, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("resolving temp dir: %v", err)
	}
	writeMinimalGit(t, base)
	dir := filepath.Join(base, isolatedName)
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("creating isolated dir: %v", err)
	}
	return dir
}

// writeMinimalGit makes dir a git repository toplevel: a .git directory with
// the HEAD, objects/ and refs/ git requires to recognise one.
func writeMinimalGit(t testing.TB, dir string) {
	t.Helper()
	gitDir := filepath.Join(dir, ".git")
	for _, d := range []string{"objects", filepath.Join("refs", "heads")} {
		if err := os.MkdirAll(filepath.Join(gitDir, d), 0o755); err != nil {
			t.Fatalf("creating minimal git repository: %v", err)
		}
	}
	if err := fileutil.WriteFileAtomic(filepath.Join(gitDir, "HEAD"), []byte("ref: refs/heads/main\n"), 0o644); err != nil {
		t.Fatalf("creating minimal git repository: %v", err)
	}
}

// IsolateUserDirs points every per-user directory a command may read or
// write at fresh directories under one temporary root: HOME and USERPROFILE,
// the XDG base directories, APPDATA and LocalAppData, the temp directory
// (TMPDIR, TMP, TEMP) and git's global configuration, with the system
// configuration disabled. It returns the isolated home directory. It uses
// t.Setenv, so the calling test cannot be parallel.
func IsolateUserDirs(t testing.TB) string {
	t.Helper()
	// Created before TMPDIR moves, so the root lies in the original temp
	// directory and is removed with the test's other temp directories.
	root := t.TempDir()
	home := filepath.Join(root, "home")
	tmp := filepath.Join(root, "tmp")
	dirs := map[string]string{
		"HOME":            home,
		"USERPROFILE":     home,
		"XDG_CONFIG_HOME": filepath.Join(home, ".config"),
		"XDG_DATA_HOME":   filepath.Join(home, ".local", "share"),
		"XDG_STATE_HOME":  filepath.Join(home, ".local", "state"),
		"XDG_CACHE_HOME":  filepath.Join(home, ".cache"),
		"APPDATA":         filepath.Join(home, "AppData", "Roaming"),
		"LocalAppData":    filepath.Join(home, "AppData", "Local"),
		"TMPDIR":          tmp,
		"TMP":             tmp,
		"TEMP":            tmp,
	}
	for v, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("creating isolated %s: %v", v, err)
		}
		t.Setenv(v, d)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", filepath.Join(home, ".gitconfig"))
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	// Registered after root's t.TempDir cleanup, so it runs first and
	// root's removal finds tmp already gone.
	t.Cleanup(func() {
		if err := removeAllWithin(tmp, tmpReleaseWait, tmpReleasePoll, os.RemoveAll); err != nil {
			t.Errorf("removing isolated temp directory: %v", err)
		}
	})
	return home
}

// removeAllWithin calls remove on path until it succeeds or wait has passed,
// pausing poll between attempts, and returns the last error.
func removeAllWithin(path string, wait, poll time.Duration, remove func(string) error) error {
	deadline := time.Now().Add(wait)
	for {
		err := remove(path)
		if err == nil {
			return nil
		}
		if !time.Now().Before(deadline) {
			return fmt.Errorf("%s still busy after %v: %w", path, wait, err)
		}
		time.Sleep(poll)
	}
}

// ProjectOptions configures Project.
type ProjectOptions struct {
	// NoGit leaves the project directory without a .git of its own, for
	// tests of behaviour outside a repository. The isolation ceiling above
	// it still ends every walk.
	NoGit bool
}

// Project returns an isolated project directory (IsolatedDir) that is its
// own git repository toplevel unless opts.NoGit, isolates the per-user
// directories (IsolateUserDirs) and makes the directory the working
// directory for the rest of the test. The calling test cannot be parallel.
func Project(t testing.TB, opts ProjectOptions) string {
	t.Helper()
	dir := IsolatedDir(t)
	if !opts.NoGit {
		writeMinimalGit(t, dir)
	}
	IsolateUserDirs(t)
	t.Chdir(dir)
	return dir
}
