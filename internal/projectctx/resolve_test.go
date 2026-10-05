package projectctx

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// mkdirs creates each slash-separated directory below root.
func mkdirs(t *testing.T, root string, dirs ...string) {
	t.Helper()
	for _, d := range dirs {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(d)), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", d, err)
		}
	}
}

// touch creates an empty regular file at the slash-separated path below root.
func touch(t *testing.T, root, file string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(file))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", file, err)
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		t.Fatalf("create %s: %v", file, err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("close %s: %v", file, err)
	}
}

// fixture returns a fresh directory holding a .git entry, so every walk in a
// test is bounded by it and never sees markers planted above the test's temp
// directory (a shared /tmp/.qsdev or /tmp/.devinit).
func fixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	mkdirs(t, root, ".git")
	return root
}

func TestResolve(t *testing.T) {
	t.Parallel()
	b := branding.Get()
	dataDir := "." + b.AppName

	tests := []struct {
		name      string
		setup     func(t *testing.T, root string)
		start     string // slash-separated, relative to the fixture root
		mode      Mode
		wantRoot  string // relative to the fixture root
		wantFound bool
		wantGit   string // relative to the fixture root; "-" for none
	}{
		{
			name: "config-file-in-parent",
			setup: func(t *testing.T, root string) {
				touch(t, root, "proj/"+b.ConfigFile)
				mkdirs(t, root, "proj/sub/dir")
			},
			start: "proj/sub/dir", wantRoot: "proj", wantFound: true, wantGit: "-",
		},
		{
			name: "config-dir-is-not-a-marker",
			setup: func(t *testing.T, root string) {
				mkdirs(t, root, "proj/"+b.ConfigFile, "proj/sub")
			},
			start: "proj/sub", wantRoot: "proj/sub", wantFound: false, wantGit: ".",
		},
		{
			name: "state-dir-in-parent",
			setup: func(t *testing.T, root string) {
				mkdirs(t, root, "proj/"+b.StateDir, "proj/sub")
			},
			start: "proj/sub", wantRoot: "proj", wantFound: true, wantGit: "-",
		},
		{
			name: "bare-data-dir-ignored",
			setup: func(t *testing.T, root string) {
				mkdirs(t, root, "proj/"+dataDir, "proj/sub")
			},
			start: "proj/sub", wantRoot: "proj/sub", wantFound: false, wantGit: ".",
		},
		{
			name: "home-data-dir-ignored",
			setup: func(t *testing.T, root string) {
				mkdirs(t, root, "home/user/"+dataDir+"/logs", "home/user/work")
			},
			start: "home/user/work", wantRoot: "home/user/work", wantFound: false, wantGit: ".",
		},
		{
			name: "stops-at-git-toplevel",
			setup: func(t *testing.T, root string) {
				touch(t, root, "outer/"+b.ConfigFile)
				mkdirs(t, root, "outer/repo/.git", "outer/repo/sub")
			},
			start: "outer/repo/sub", wantRoot: "outer/repo/sub", wantFound: false, wantGit: "outer/repo",
		},
		{
			name: "marker-at-git-toplevel",
			setup: func(t *testing.T, root string) {
				touch(t, root, "repo/"+b.ConfigFile)
				mkdirs(t, root, "repo/.git", "repo/a/b")
			},
			start: "repo/a/b", wantRoot: "repo", wantFound: true, wantGit: "repo",
		},
		{
			name: "git-file-worktree-ceiling",
			setup: func(t *testing.T, root string) {
				mkdirs(t, root, "main/"+b.StateDir, "main/wt/sub")
				touch(t, root, "main/wt/.git")
			},
			start: "main/wt/sub", wantRoot: "main/wt/sub", wantFound: false, wantGit: "main/wt",
		},
		{
			name: "here-mode-returns-start",
			setup: func(t *testing.T, root string) {
				touch(t, root, "proj/"+b.ConfigFile)
				mkdirs(t, root, "proj/sub")
			},
			start: "proj/sub", mode: Here, wantRoot: "proj/sub", wantFound: false, wantGit: "-",
		},
		{
			name: "here-mode-marker-at-start",
			setup: func(t *testing.T, root string) {
				mkdirs(t, root, "proj/"+b.StateDir)
			},
			start: "proj", mode: Here, wantRoot: "proj", wantFound: true, wantGit: "-",
		},
		{
			name: "not-found-falls-back-to-start",
			setup: func(t *testing.T, root string) {
				mkdirs(t, root, "a/b/c")
			},
			start: "a/b/c", wantRoot: "a/b/c", wantFound: false, wantGit: ".",
		},
		{
			name: "marker-at-start",
			setup: func(t *testing.T, root string) {
				touch(t, root, "proj/"+b.ConfigFile)
			},
			start: "proj", wantRoot: "proj", wantFound: true, wantGit: "-",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := fixture(t)
			tt.setup(t, root)
			start := filepath.Join(root, filepath.FromSlash(tt.start))
			got, err := Resolve(start, tt.mode)
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			wantRoot := filepath.Join(root, filepath.FromSlash(tt.wantRoot))
			if got.Start != start || got.Root != wantRoot || got.Found != tt.wantFound {
				t.Errorf("Resolve = {Start:%q Root:%q Found:%v}, want {Start:%q Root:%q Found:%v}",
					got.Start, got.Root, got.Found, start, wantRoot, tt.wantFound)
			}
			wantGit := ""
			if tt.wantGit != "-" {
				wantGit = filepath.Join(root, filepath.FromSlash(tt.wantGit))
			}
			if got.GitTop != wantGit {
				t.Errorf("GitTop = %q, want %q", got.GitTop, wantGit)
			}
			if len(got.Ignored) != 0 {
				t.Errorf("Ignored = %v, want none", got.Ignored)
			}
		})
	}
}

func TestResolve_RelativeStartErrors(t *testing.T) {
	t.Parallel()
	for _, start := range []string{"", ".", "rel/dir"} {
		_, err := Resolve(start, Enclosing)
		if !errors.Is(err, ErrRelativeStart) {
			t.Errorf("Resolve(%q) error = %v, want ErrRelativeStart", start, err)
		}
	}
}

func TestResolve_MissingStartErrors(t *testing.T) {
	t.Parallel()
	start := filepath.Join(t.TempDir(), "missing")
	if _, err := Resolve(start, Enclosing); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("Resolve(missing) error = %v, want fs.ErrNotExist", err)
	}
}

func TestResolve_CleansStart(t *testing.T) {
	t.Parallel()
	root := fixture(t)
	touch(t, root, "proj/"+branding.Get().ConfigFile)
	mkdirs(t, root, "proj/sub")
	start := filepath.Join(root, "proj", "sub") + string(filepath.Separator) + "." + string(filepath.Separator)
	got, err := Resolve(start, Enclosing)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if want := filepath.Join(root, "proj", "sub"); got.Start != want {
		t.Errorf("Start = %q, want cleaned %q", got.Start, want)
	}
	if want := filepath.Join(root, "proj"); got.Root != want {
		t.Errorf("Root = %q, want %q", got.Root, want)
	}
}

// countSeams wraps the stat and lstat seams with call counters for the
// duration of the test. Tests using it must not run in parallel.
func countSeams(t *testing.T) *int {
	t.Helper()
	origStat, origLstat := stat, lstat
	t.Cleanup(func() { stat, lstat = origStat, origLstat })
	calls := new(int)
	stat = func(name string) (fs.FileInfo, error) { *calls++; return origStat(name) }
	lstat = func(name string) (fs.FileInfo, error) { *calls++; return origLstat(name) }
	return calls
}

// TestResolve_StatCallsPerLevel bounds the filesystem syscalls Resolve makes
// for each directory level it visits, on every OS: a walk from deep inside a
// tree must stay O(depth) with a small constant.
func TestResolve_StatCallsPerLevel(t *testing.T) {
	const maxPerLevel = 4
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b", "c", "d", "e", "f", "g", "h")
	mkdirs(t, root, "a/b/c/d/e/f/g/h")
	// Every level from deep up to the volume root may be visited.
	levels := 1
	for d := deep; filepath.Dir(d) != d; d = filepath.Dir(d) {
		levels++
	}

	for _, mode := range []Mode{Enclosing, Here} {
		calls := countSeams(t)
		if _, err := Resolve(deep, mode); err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		limit := maxPerLevel * levels
		if mode == Here {
			limit = maxPerLevel
		}
		if *calls == 0 || *calls > limit {
			t.Errorf("mode %v: %d stat/lstat calls, want 1..%d (%d per level)", mode, *calls, limit, maxPerLevel)
		}
	}

	// A walk bounded inside the fixture visits exactly levels start..gittop.
	fix := fixture(t)
	mkdirs(t, fix, "x/y/z")
	calls := countSeams(t)
	if _, err := Resolve(filepath.Join(fix, "x", "y", "z"), Enclosing); err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if limit := maxPerLevel * 4; *calls > limit {
		t.Errorf("bounded walk over 4 levels: %d calls, want <= %d", *calls, limit)
	}

	// Symlinked markers of the wrong kind are the worst case: each costs an
	// lstat and a stat, and neither ends the walk. Every level of x/y/z/w
	// holds a config-file link to a directory and a state-dir link to a file.
	const maxPerSymlinkLevel = maxPerLevel + 2
	b := branding.Get()
	sym := fixture(t)
	touch(t, sym, "target-file")
	mkdirs(t, sym, "target-dir", "x/y/z/w")
	for _, d := range []string{"x", "x/y", "x/y/z", "x/y/z/w"} {
		dir := filepath.Join(sym, filepath.FromSlash(d))
		if err := os.Symlink(filepath.Join(sym, "target-dir"), filepath.Join(dir, b.ConfigFile)); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		if err := os.Symlink(filepath.Join(sym, "target-file"), filepath.Join(dir, b.StateDir)); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
	}
	calls = countSeams(t)
	pc, err := Resolve(filepath.Join(sym, "x", "y", "z", "w"), Enclosing)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if pc.Found {
		t.Fatalf("symlinks to the wrong kind were taken as markers: %+v", pc)
	}
	if limit := maxPerSymlinkLevel * 5; *calls > limit {
		t.Errorf("walk over 5 levels with symlinked markers: %d calls, want <= %d", *calls, limit)
	}
}

func TestContextRoundTrip(t *testing.T) {
	t.Parallel()
	if _, ok := FromContext(context.Background()); ok {
		t.Fatal("FromContext on a bare context reported a Context")
	}
	want := Context{Start: "/s", Root: "/r", GitTop: "/r", Found: true, Ignored: []string{"/x"}}
	got, ok := FromContext(WithContext(context.Background(), want))
	if !ok {
		t.Fatal("FromContext lost the stored Context")
	}
	if got.Start != want.Start || got.Root != want.Root || got.GitTop != want.GitTop ||
		got.Found != want.Found || strings.Join(got.Ignored, ",") != strings.Join(want.Ignored, ",") {
		t.Errorf("FromContext = %+v, want %+v", got, want)
	}
}

func TestResolveWorkingDir(t *testing.T) {
	// Not parallel: t.Chdir changes process-wide state.
	root := fixture(t)
	touch(t, root, "proj/"+branding.Get().ConfigFile)
	mkdirs(t, root, "proj/sub")
	t.Chdir(filepath.Join(root, "proj", "sub"))
	for _, tt := range []struct {
		mode Mode
		want string
	}{
		{Enclosing, filepath.Join(root, "proj")},
		{Here, filepath.Join(root, "proj", "sub")},
	} {
		got, err := ResolveWorkingDir(tt.mode)
		if err != nil {
			t.Fatalf("ResolveWorkingDir(%v): %v", tt.mode, err)
		}
		// The working directory may be reported through a symlinked temp dir.
		gotInfo, err1 := os.Stat(got.Root)
		wantInfo, err2 := os.Stat(tt.want)
		if err1 != nil || err2 != nil || !os.SameFile(gotInfo, wantInfo) {
			t.Errorf("ResolveWorkingDir(%v).Root = %q, want %q", tt.mode, got.Root, tt.want)
		}
	}
}

func TestWorkingDirAndHomeDir(t *testing.T) {
	wd, err := WorkingDir()
	if err != nil || !filepath.IsAbs(wd) {
		t.Errorf("WorkingDir() = %q, %v; want an absolute path", wd, err)
	}

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if got, err := HomeDir(); err != nil || got != home {
		t.Errorf("HomeDir() = %q, %v; want %q", got, err, home)
	}

	t.Setenv("HOME", "relative/home")
	t.Setenv("USERPROFILE", "relative/home")
	if got, err := HomeDir(); err == nil {
		t.Errorf("HomeDir() with a relative HOME = %q, want an error", got)
	}
}

// TestCheckTrusted_StatCalls bounds CheckTrusted's filesystem calls on every
// OS: an lstat of the entry and a stat of its parent, plus a stat of the
// target for a symlink.
func TestCheckTrusted_StatCalls(t *testing.T) {
	root := t.TempDir()
	touch(t, root, "f")
	calls := countSeams(t)
	if err := CheckTrusted(filepath.Join(root, "f")); err != nil {
		t.Fatalf("CheckTrusted: %v", err)
	}
	if *calls != 2 {
		t.Errorf("CheckTrusted of a regular file: %d stat/lstat calls, want 2", *calls)
	}
	if err := os.Symlink(filepath.Join(root, "f"), filepath.Join(root, "l")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	*calls = 0
	if err := CheckTrusted(filepath.Join(root, "l")); err != nil {
		t.Fatalf("CheckTrusted(symlink): %v", err)
	}
	if *calls != 3 {
		t.Errorf("CheckTrusted of a symlink: %d stat/lstat calls, want 3", *calls)
	}
}
