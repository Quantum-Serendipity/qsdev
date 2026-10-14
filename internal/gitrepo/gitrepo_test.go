package gitrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/procexec"
)

// requireGit skips when git is absent, as the drift tests do.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := procexec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

// isolatedEnv is the environment test git commands run with: the process
// environment minus repository pins, plus a throwaway global config and no
// system config, so the developer's git setup cannot steer the result.
func isolatedEnv(t *testing.T) []string {
	t.Helper()
	return append(ScopedEnv(os.Environ()),
		"GIT_CONFIG_GLOBAL="+filepath.Join(t.TempDir(), "gitconfig"),
		"GIT_CONFIG_NOSYSTEM=1",
	)
}

// runGit runs git in dir with a fixed identity, failing the test on error.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	full := append([]string{"-c", "user.name=test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false"}, args...)
	cmd := procexec.Command("git", full...)
	cmd.Dir = dir
	cmd.Env = isolatedEnv(t)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

// writeFile writes content to dir/rel, creating parent directories.
func writeFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// newRepo creates a repository whose main branch tracks
// .claude/settings.json and src/x.go, and whose branch evil changes the
// settings file.
func newRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGit(t, repo, "init", "-q", "-b", "main")
	writeFile(t, repo, ".claude/settings.json", "{}\n")
	writeFile(t, repo, "src/x.go", "package x\n")
	runGit(t, repo, "add", "-A")
	runGit(t, repo, "commit", "-q", "-m", "init")
	runGit(t, repo, "checkout", "-q", "-b", "evil")
	writeFile(t, repo, ".claude/settings.json", "{\"permissions\":{}}\n")
	runGit(t, repo, "commit", "-q", "-am", "evil")
	runGit(t, repo, "checkout", "-q", "main")
	return repo
}

// client is a Client whose real runner uses the isolated environment.
func client(t *testing.T) Client {
	t.Helper()
	env := isolatedEnv(t)
	return Client{Runner: func(ctx context.Context, c Call) ([]byte, error) {
		c.Env = append(slices.Clone(env), c.Env...)
		return ExecRunner(ctx, c)
	}}
}

func TestScopedEnv(t *testing.T) {
	t.Parallel()
	in := []string{
		"PATH=/bin", "GIT_DIR=/x/.git", "GIT_WORK_TREE=/x", "GIT_COMMON_DIR=/x/.git",
		"GIT_INDEX_FILE=/x/.git/index", "GIT_AUTHOR_NAME=a", "HOME=/h",
	}
	want := []string{"PATH=/bin", "GIT_AUTHOR_NAME=a", "HOME=/h"}
	if got := ScopedEnv(in); !slices.Equal(got, want) {
		t.Errorf("ScopedEnv() = %q, want %q", got, want)
	}
}

func TestHooksDir(t *testing.T) {
	requireGit(t)
	t.Parallel()
	tests := []struct {
		name  string
		setup func(t *testing.T, repo string) (dir, want string)
	}{
		{"plain repository", func(t *testing.T, repo string) (string, string) {
			return repo, filepath.Join(repo, ".git", "hooks")
		}},
		{"core.hooksPath", func(t *testing.T, repo string) (string, string) {
			runGit(t, repo, "config", "core.hooksPath", ".githooks")
			hooks := filepath.Join(repo, ".githooks")
			if err := os.MkdirAll(hooks, 0o755); err != nil {
				t.Fatal(err)
			}
			return repo, hooks
		}},
		{"linked worktree uses the common hooks", func(t *testing.T, repo string) (string, string) {
			wt := filepath.Join(t.TempDir(), "feature")
			runGit(t, repo, "worktree", "add", "-q", wt, "evil")
			return wt, filepath.Join(repo, ".git", "hooks")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir, want := tt.setup(t, newRepo(t))
			got, err := client(t).HooksDir(context.Background(), dir)
			if err != nil {
				t.Fatalf("HooksDir: %v", err)
			}
			assertSamePath(t, got, want)
		})
	}

	t.Run("subdirectory of a repository is not a repository root", func(t *testing.T) {
		t.Parallel()
		repo := newRepo(t)
		_, err := client(t).HooksDir(context.Background(), filepath.Join(repo, "src"))
		if !errors.Is(err, ErrNotRepo) {
			t.Errorf("HooksDir(subdir) error = %v, want ErrNotRepo", err)
		}
	})
}

func TestToplevel(t *testing.T) {
	requireGit(t)
	t.Parallel()
	repo := newRepo(t)
	c := client(t)
	for _, dir := range []string{repo, filepath.Join(repo, "src")} {
		got, err := c.Toplevel(context.Background(), dir)
		if err != nil {
			t.Fatalf("Toplevel(%s): %v", dir, err)
		}
		assertSamePath(t, got, repo)
	}
	if _, err := c.Toplevel(context.Background(), t.TempDir()); !errors.Is(err, ErrNotRepo) {
		t.Errorf("Toplevel(non-repo) error = %v, want ErrNotRepo", err)
	}
}

func TestChangedPaths(t *testing.T) {
	requireGit(t)
	t.Parallel()
	repo := newRepo(t)
	writeFile(t, repo, "src/x.go", "package x // edited\n")
	c := client(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		dir      string
		from, to string
		pathspec []string
		want     []string
		wantErr  error
	}{
		{name: "index vs worktree", dir: repo, want: []string{"src/x.go"}},
		{name: "rev vs worktree", dir: repo, from: "evil", want: []string{".claude/settings.json", "src/x.go"}},
		{name: "rev vs worktree with pathspec", dir: repo, from: "evil", pathspec: []string{".claude"}, want: []string{".claude/settings.json"}},
		{name: "rev vs rev", dir: repo, from: "HEAD", to: "evil", want: []string{".claude/settings.json"}},
		{name: "same tree", dir: repo, from: "HEAD", to: "main", want: nil},
		{name: "paths stay toplevel-relative from a subdir", dir: filepath.Join(repo, "src"), from: "evil", want: []string{".claude/settings.json", "src/x.go"}},
		{name: "option-shaped rev is a rev, not an option", dir: repo, from: "--output=" + filepath.Join(t.TempDir(), "o"), wantErr: ErrGit},
		{name: "unknown rev", dir: repo, from: "nosuch", wantErr: ErrGit},
		// A pathspec that git checkout accepts in a revision's place is not
		// a revision to diff: the caller rereads it as a pathspec.
		{name: "pathspec magic is not a rev", dir: repo, from: "HEAD", to: ":/", wantErr: ErrGit},
		{name: "directory is not a rev", dir: repo, from: "HEAD", to: "src", wantErr: ErrGit},
		{name: "top pathspec from a subdir", dir: filepath.Join(repo, "src"), pathspec: []string{":/"}, want: []string{"src/x.go"}},
		{name: "parent pathspec from a subdir", dir: filepath.Join(repo, "src"), pathspec: []string{".."}, want: []string{"src/x.go"}},
		{name: "not a repository", dir: t.TempDir(), wantErr: ErrNotRepo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := c.ChangedPaths(ctx, tt.dir, tt.from, tt.to, tt.pathspec)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("ChangedPaths error = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ChangedPaths: %v", err)
			}
			if !slices.Equal(got, tt.want) {
				t.Errorf("ChangedPaths = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestStashPaths(t *testing.T) {
	requireGit(t)
	t.Parallel()
	repo := newRepo(t)
	writeFile(t, repo, ".claude/settings.json", "{\"x\":1}\n")
	writeFile(t, repo, "untracked.txt", "u\n")
	runGit(t, repo, "stash", "push", "-q", "--include-untracked")
	c := client(t)
	ctx := context.Background()

	got, err := c.StashPaths(ctx, repo, "stash@{0}")
	if err != nil {
		t.Fatalf("StashPaths: %v", err)
	}
	if want := []string{".claude/settings.json", "untracked.txt"}; !slices.Equal(got, want) {
		t.Errorf("StashPaths = %q, want %q", got, want)
	}
	if _, err := c.StashPaths(ctx, repo, "stash@{5}"); !errors.Is(err, ErrGit) {
		t.Errorf("StashPaths(missing stash) error = %v, want ErrGit", err)
	}
	if _, err := c.StashPaths(ctx, t.TempDir(), "stash@{0}"); !errors.Is(err, ErrNotRepo) {
		t.Errorf("StashPaths(non-repo) error = %v, want ErrNotRepo", err)
	}

	// A stashed rename names the path it removes, not only its destination.
	runGit(t, repo, "mv", ".claude/settings.json", "moved.json")
	runGit(t, repo, "stash", "push", "-q")
	got, err = c.StashPaths(ctx, repo, "stash@{0}")
	if err != nil {
		t.Fatalf("StashPaths(rename): %v", err)
	}
	if want := []string{".claude/settings.json", "moved.json"}; !slices.Equal(got, want) {
		t.Errorf("StashPaths(rename) = %q, want %q", got, want)
	}
}

// TestTrackedPaths proves the index listing: tracked paths only (not an
// untracked file), relative to the directory, and ErrNotRepo outside a
// repository.
func TestTrackedPaths(t *testing.T) {
	requireGit(t)
	t.Parallel()
	repo := newRepo(t)
	writeFile(t, repo, "untracked.txt", "u\n")
	c := client(t)
	ctx := context.Background()

	got, err := c.TrackedPaths(ctx, repo)
	if err != nil {
		t.Fatalf("TrackedPaths: %v", err)
	}
	if !slices.Contains(got, ".claude/settings.json") || slices.Contains(got, "untracked.txt") {
		t.Errorf("TrackedPaths = %q, want the tracked .claude/settings.json and not untracked.txt", got)
	}
	if _, err := c.TrackedPaths(ctx, t.TempDir()); !errors.Is(err, ErrNotRepo) {
		t.Errorf("TrackedPaths(non-repo) error = %v, want ErrNotRepo", err)
	}
}

// TestCleanPaths proves the clean listing against git: it names what git
// clean with the same options would remove, a directory as a whole with -d,
// ignored files only with -x or -X, and it removes nothing itself. After the
// index stops tracking a file, the file is untracked and listed.
func TestCleanPaths(t *testing.T) {
	requireGit(t)
	t.Parallel()
	repo := newRepo(t)
	writeFile(t, repo, ".gitignore", "*.log\n")
	writeFile(t, repo, "loose.txt", "u\n")
	writeFile(t, repo, "newdir/a.txt", "u\n")
	writeFile(t, repo, "build.log", "u\n")
	c := client(t)
	ctx := context.Background()

	tests := []struct {
		name     string
		opts     CleanOptions
		pathspec []string
		want     []string
	}{
		{"files", CleanOptions{Force: 1}, nil, []string{".gitignore", "loose.txt"}},
		{"directories", CleanOptions{Force: 1, Dirs: true}, nil, []string{".gitignore", "loose.txt", "newdir/"}},
		{"ignored too", CleanOptions{Force: 1, Ignored: true}, nil, []string{".gitignore", "build.log", "loose.txt"}},
		{"only ignored", CleanOptions{Force: 1, OnlyIgnored: true}, nil, []string{"build.log"}},
		{"excluded", CleanOptions{Force: 1, Excludes: []string{"loose.*"}}, nil, []string{".gitignore"}},
		{"pathspec", CleanOptions{Force: 1, Dirs: true}, []string{"newdir"}, []string{"newdir/"}},
	}
	for _, tt := range tests {
		got, err := c.CleanPaths(ctx, repo, tt.opts, tt.pathspec)
		if err != nil {
			t.Fatalf("%s: CleanPaths: %v", tt.name, err)
		}
		slices.Sort(got)
		if !slices.Equal(got, tt.want) {
			t.Errorf("%s: CleanPaths = %q, want %q", tt.name, got, tt.want)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, "loose.txt")); err != nil {
		t.Errorf("the listing removed loose.txt: %v", err)
	}

	runGit(t, repo, "rm", "-r", "-q", "--cached", ".")
	got, err := c.CleanPaths(ctx, repo, CleanOptions{Force: 1, Dirs: true}, nil)
	if err != nil {
		t.Fatalf("CleanPaths after rm --cached: %v", err)
	}
	if !slices.Contains(got, ".claude/") {
		t.Errorf("CleanPaths after rm --cached = %q, want .claude/", got)
	}
	if _, err := c.CleanPaths(ctx, t.TempDir(), CleanOptions{Force: 1}, nil); err == nil {
		t.Error("CleanPaths(non-repo) succeeded, want an error")
	}
}

// TestCleanOutputPaths pins the parse of git clean's listing: C-quoted paths
// are unquoted, a skipped nested repository is not removed, and a quoted path
// that cannot be read is an error rather than a path the check never sees.
func TestCleanOutputPaths(t *testing.T) {
	t.Parallel()
	out := "Would remove a.txt\nWould remove \"we\\\"ird\\tname\"\nWould skip repository nested/\r\nWould remove dir/\n"
	got, err := cleanOutputPaths([]byte(out))
	if err != nil {
		t.Fatalf("cleanOutputPaths: %v", err)
	}
	if want := []string{"a.txt", "we\"ird\tname", "dir/"}; !slices.Equal(got, want) {
		t.Errorf("cleanOutputPaths = %q, want %q", got, want)
	}
	if _, err := cleanOutputPaths([]byte("Would remove \"bad\n")); !errors.Is(err, ErrGit) {
		t.Errorf("cleanOutputPaths(unterminated quote) error = %v, want ErrGit", err)
	}
}

// branchRepo is newRepo after main moved on: main changes the settings file
// and branch feature, forked before that, changes only src/x.go. Branch
// moved renames the settings file away.
func branchRepo(t *testing.T) string {
	t.Helper()
	repo := newRepo(t)
	runGit(t, repo, "checkout", "-q", "-b", "feature")
	writeFile(t, repo, "src/x.go", "package x // feature\n")
	runGit(t, repo, "commit", "-q", "-am", "feature")
	runGit(t, repo, "checkout", "-q", "-b", "moved", "main")
	runGit(t, repo, "mv", ".claude/settings.json", "moved.json")
	runGit(t, repo, "commit", "-q", "-m", "moved")
	runGit(t, repo, "checkout", "-q", "main")
	writeFile(t, repo, ".claude/settings.json", "{\"main\":1}\n")
	runGit(t, repo, "commit", "-q", "-am", "main")
	return repo
}

// TestChangedPaths_MergeBaseRange proves the range form: "HEAD...rev" lists
// only what rev changed since it forked, which is what merging or rebasing
// onto it writes, while the two-dot form also lists what HEAD changed since.
// A rename lists its source as well as its destination.
func TestChangedPaths_MergeBaseRange(t *testing.T) {
	requireGit(t)
	t.Parallel()
	repo := branchRepo(t)
	c := client(t)
	ctx := context.Background()
	tests := []struct {
		from string
		want []string
	}{
		{"HEAD...feature", []string{"src/x.go"}},
		{"HEAD..feature", []string{".claude/settings.json", "src/x.go"}},
		{"HEAD...evil", []string{".claude/settings.json"}},
		{"feature...HEAD", []string{".claude/settings.json"}},
		{"HEAD...moved", []string{".claude/settings.json", "moved.json"}},
	}
	for _, tt := range tests {
		got, err := c.ChangedPaths(ctx, repo, tt.from, "", nil)
		if err != nil {
			t.Fatalf("ChangedPaths(%s): %v", tt.from, err)
		}
		if !slices.Equal(got, tt.want) {
			t.Errorf("ChangedPaths(%s) = %q, want %q", tt.from, got, tt.want)
		}
	}
	if _, err := c.ChangedPaths(ctx, repo, "HEAD...nosuch", "", nil); !errors.Is(err, ErrGit) {
		t.Errorf("ChangedPaths(unknown range end) error = %v, want ErrGit", err)
	}
}

// TestCommitPaths proves the commit listing: one commit's paths, a range's
// commits, a rename's two names, every commit of several, and errors for an
// unknown or option-shaped revision and outside a repository.
func TestCommitPaths(t *testing.T) {
	requireGit(t)
	t.Parallel()
	repo := branchRepo(t)
	c := client(t)
	ctx := context.Background()
	tests := []struct {
		revs []string
		want []string
	}{
		{[]string{"evil"}, []string{".claude/settings.json"}},
		{[]string{"feature"}, []string{"src/x.go"}},
		{[]string{"evil~1..evil"}, []string{".claude/settings.json"}},
		{[]string{"moved"}, []string{".claude/settings.json", "moved.json"}},
		{[]string{"feature", "evil"}, []string{"src/x.go", ".claude/settings.json"}},
	}
	for _, tt := range tests {
		got, err := c.CommitPaths(ctx, repo, tt.revs)
		if err != nil {
			t.Fatalf("CommitPaths(%q): %v", tt.revs, err)
		}
		slices.Sort(got)
		want := slices.Sorted(slices.Values(tt.want))
		if !slices.Equal(got, want) {
			t.Errorf("CommitPaths(%q) = %q, want %q", tt.revs, got, want)
		}
	}
	for _, revs := range [][]string{{"nosuch"}, {"--output=" + filepath.Join(t.TempDir(), "o")}} {
		if _, err := c.CommitPaths(ctx, repo, revs); !errors.Is(err, ErrGit) {
			t.Errorf("CommitPaths(%q) error = %v, want ErrGit", revs, err)
		}
	}
	if _, err := c.CommitPaths(ctx, t.TempDir(), []string{"HEAD"}); !errors.Is(err, ErrNotRepo) {
		t.Errorf("CommitPaths(non-repo) error = %v, want ErrNotRepo", err)
	}
}

// TestPatchPaths proves the patch listing: a mailbox from format-patch, a
// rename's source and destination, a plain diff read from a subdirectory
// (named from the top level), -p and --directory, a file named like an
// option, outside a repository, and an error for a Maildir (a directory) and
// a missing file. Nothing is applied.
func TestPatchPaths(t *testing.T) {
	requireGit(t)
	t.Parallel()
	repo := branchRepo(t)
	patches := t.TempDir()
	runGit(t, repo, "format-patch", "-q", "-o", filepath.Join(patches, "evil"), "-1", "evil")
	runGit(t, repo, "format-patch", "-q", "-o", filepath.Join(patches, "moved"), "-1", "moved")
	mbox := func(name string) string {
		t.Helper()
		files, err := filepath.Glob(filepath.Join(patches, name, "*.patch"))
		if err != nil || len(files) != 1 {
			t.Fatalf("format-patch %s wrote %q (%v)", name, files, err)
		}
		return files[0]
	}
	plain := filepath.Join(patches, "-plain.patch")
	writeFile(t, patches, "-plain.patch", "--- a/y.txt\n+++ b/y.txt\n@@ -0,0 +1 @@\n+hi\n")
	c := client(t)
	ctx := context.Background()

	tests := []struct {
		name             string
		dir              string
		files            []string
		strip, directory string
		want             []string
	}{
		{"mailbox", repo, []string{mbox("evil")}, "", "", []string{".claude/settings.json"}},
		{"rename", repo, []string{mbox("moved")}, "", "", []string{".claude/settings.json", "moved.json"}},
		{"plain diff from a subdirectory", filepath.Join(repo, "src"), []string{plain}, "", "", []string{"src/y.txt"}},
		{"strip and directory", repo, []string{plain}, "0", "sub", []string{"sub/b/y.txt"}},
		{"outside a repository", t.TempDir(), []string{mbox("evil")}, "", "", []string{".claude/settings.json"}},
	}
	for _, tt := range tests {
		got, err := c.PatchPaths(ctx, tt.dir, tt.files, tt.strip, tt.directory)
		if err != nil {
			t.Fatalf("%s: PatchPaths: %v", tt.name, err)
		}
		got = slices.Compact(slices.Sorted(slices.Values(got)))
		if !slices.Equal(got, tt.want) {
			t.Errorf("%s: PatchPaths = %q, want %q", tt.name, got, tt.want)
		}
	}
	for _, files := range [][]string{{patches}, {filepath.Join(patches, "missing.patch")}} {
		if _, err := c.PatchPaths(ctx, repo, files, "", ""); !errors.Is(err, ErrGit) {
			t.Errorf("PatchPaths(%q) error = %v, want ErrGit", files, err)
		}
	}
	// Nothing was applied.
	if b, err := os.ReadFile(filepath.Join(repo, ".claude", "settings.json")); err != nil || string(b) != "{\"main\":1}\n" {
		t.Errorf("settings.json = %q (%v) after PatchPaths, want it untouched", b, err)
	}
}

// TestQueryTimeout proves a runner that outlives a query's timeout yields
// ErrTimeout, through the injected runner (no git needed).
func TestQueryTimeout(t *testing.T) {
	t.Parallel()
	sleeper := func(ctx context.Context, _ Call) ([]byte, error) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Minute):
			return nil, errors.New("runner was not cancelled")
		}
	}
	c := Client{Runner: sleeper, revParseTimeout: 10 * time.Millisecond, queryTimeout: 10 * time.Millisecond}
	ctx := context.Background()
	queries := map[string]func() error{
		"Toplevel":     func() error { _, err := c.Toplevel(ctx, "/r"); return err },
		"HooksDir":     func() error { _, err := c.HooksDir(ctx, "/r"); return err },
		"ChangedPaths": func() error { _, err := c.ChangedPaths(ctx, "/r", "HEAD", "", nil); return err },
		"StashPaths":   func() error { _, err := c.StashPaths(ctx, "/r", "stash@{0}"); return err },
		"TrackedPaths": func() error { _, err := c.TrackedPaths(ctx, "/r"); return err },
		"CommitPaths":  func() error { _, err := c.CommitPaths(ctx, "/r", []string{"evil"}); return err },
		"PatchPaths":   func() error { _, err := c.PatchPaths(ctx, "/r", []string{"x.patch"}, "", ""); return err },
	}
	for name, q := range queries {
		if err := q(); !errors.Is(err, ErrTimeout) {
			t.Errorf("%s error = %v, want ErrTimeout", name, err)
		}
	}
}

// TestDefaultTimeouts pins the per-query budgets, which must stay well inside
// the hook's evaluation deadline.
func TestDefaultTimeouts(t *testing.T) {
	t.Parallel()
	var c Client
	if got := c.revParseTimeoutOrDefault(); got != time.Second {
		t.Errorf("rev-parse timeout = %v, want 1s", got)
	}
	if got := c.queryTimeoutOrDefault(); got != 2*time.Second {
		t.Errorf("diff/stash timeout = %v, want 2s", got)
	}
}

// TestCallShapes pins the argv, directory and environment each query hands
// the runner: cmd.Dir is the caller's directory and the argv is exactly the
// procexec local-probe shape.
func TestCallShapes(t *testing.T) {
	t.Parallel()
	var calls []Call // the queries run one at a time, on this goroutine
	c := Client{Runner: func(_ context.Context, call Call) ([]byte, error) {
		calls = append(calls, call)
		return []byte("/r\n"), nil
	}}
	ctx := context.Background()
	dir := filepath.Join(string(filepath.Separator)+"r", "sub")
	diff := []string{"--no-optional-locks", "-c", "core.fsmonitor=false", "diff", "--name-only", "-z",
		"--no-relative", "--no-renames", "--no-ext-diff", "--no-textconv", "--end-of-options"}
	log := []string{"--no-optional-locks", "log", "--format=", "--name-only", "-z", "--no-relative", "--no-renames",
		"--no-show-signature", "--no-walk", "--diff-merges=separate", "--end-of-options"}
	numstat := []string{"--no-optional-locks", "apply", "--numstat", "-z"}

	tests := []struct {
		name string
		run  func()
		want [][]string // the argv of each git spawned, in order
	}{
		{"Toplevel", func() { _, _ = c.Toplevel(ctx, dir) }, [][]string{{"rev-parse", "--show-toplevel"}}},
		{"FetchHeadMerges", func() { _, _ = c.FetchHeadMerges(ctx, dir) },
			[][]string{{"rev-parse", "--path-format=absolute", "--git-path", "FETCH_HEAD"}}},
		{"ChangedPaths", func() { _, _ = c.ChangedPaths(ctx, dir, "evil", "", []string{"."}) },
			[][]string{slices.Concat(diff, []string{"evil", "--", "."})}},
		// A range is one revision argument, after --end-of-options.
		{"ChangedPaths merge-base range", func() { _, _ = c.ChangedPaths(ctx, dir, "HEAD...-evil", "", nil) },
			[][]string{slices.Concat(diff, []string{"HEAD...-evil", "--"})}},
		{"StashPaths", func() { _, _ = c.StashPaths(ctx, dir, "stash@{1}") },
			[][]string{{"--no-optional-locks", "stash", "show", "--name-only", "-z", "--no-renames", "--include-untracked",
				"--end-of-options", "stash@{1}"}}},
		{"TrackedPaths", func() { _, _ = c.TrackedPaths(ctx, dir) },
			[][]string{{"--no-optional-locks", "-c", "core.fsmonitor=false", "ls-files", "-z"}}},
		// The options follow --dry-run, an exclude is bound to its option and
		// the pathspec follows "--"; -f is passed at most twice.
		{"CleanPaths", func() {
			_, _ = c.CleanPaths(ctx, dir, CleanOptions{Force: 3, Dirs: true, Ignored: true, Excludes: []string{"-n"}}, []string{"-x"})
		}, [][]string{{"--no-optional-locks", "-c", "core.fsmonitor=false", "-c", "core.quotePath=false", "clean", "--dry-run",
			"-d", "-x", "-f", "-f", "--exclude=-n", "--", "-x"}}},
		{"CleanPaths only ignored", func() { _, _ = c.CleanPaths(ctx, dir, CleanOptions{Ignored: true, OnlyIgnored: true}, nil) },
			[][]string{{"--no-optional-locks", "-c", "core.fsmonitor=false", "-c", "core.quotePath=false", "clean", "--dry-run",
				"-X", "--"}}},
		// A revision that starts with "-" stays a revision.
		{"CommitPaths", func() { _, _ = c.CommitPaths(ctx, dir, []string{"main..evil", "--all"}) },
			[][]string{slices.Concat(log, []string{"main..evil", "--all", "--"})}},
		// The patch files follow "--", so one named like an option is a file;
		// the second read is in reverse.
		{"PatchPaths", func() { _, _ = c.PatchPaths(ctx, dir, []string{"-x.patch", "b.mbox"}, "", "") },
			[][]string{
				slices.Concat(numstat, []string{"--", "-x.patch", "b.mbox"}),
				slices.Concat(numstat, []string{"-R", "--", "-x.patch", "b.mbox"}),
			}},
		{"PatchPaths strip and directory", func() { _, _ = c.PatchPaths(ctx, dir, []string{"x.patch"}, "2", "sub") },
			[][]string{
				slices.Concat(numstat, []string{"-p2", "--directory=sub", "--", "x.patch"}),
				slices.Concat(numstat, []string{"-R", "-p2", "--directory=sub", "--", "x.patch"}),
			}},
	}
	for _, tt := range tests {
		calls = nil
		tt.run()
		if len(calls) != len(tt.want) {
			t.Errorf("%s spawned %d git processes, want %d", tt.name, len(calls), len(tt.want))
			continue
		}
		for i, call := range calls {
			if call.Dir != dir {
				t.Errorf("%s ran in %q, want %q", tt.name, call.Dir, dir)
			}
			if !slices.Equal(call.Args, tt.want[i]) {
				t.Errorf("%s argv = %q, want %q", tt.name, call.Args, tt.want[i])
			}
		}
	}

	// Refused before spawning anything.
	calls = nil
	refusals := map[string]func() error{
		"StashPaths(option-shaped stash)": func() error { _, err := c.StashPaths(ctx, dir, "--output=x"); return err },
		"CommitPaths(no revisions)":       func() error { _, err := c.CommitPaths(ctx, dir, nil); return err },
		"PatchPaths(no files)":            func() error { _, err := c.PatchPaths(ctx, dir, nil, "", ""); return err },
		"PatchPaths(option in strip)":     func() error { _, err := c.PatchPaths(ctx, dir, []string{"x"}, "1 --apply", ""); return err },
	}
	for name, q := range refusals {
		if err := q(); err == nil {
			t.Errorf("%s succeeded, want a refusal", name)
		}
	}
	if len(calls) != 0 {
		t.Errorf("refused queries spawned git: %v", calls)
	}
}

// TestFetchHeadMerges proves the FETCH_HEAD reading against git: after a
// fetch of two branches both heads are recorded for merging (git merge
// FETCH_HEAD merges both), after a fetch of a branch from a remote with a
// default refspec the others are not-for-merge, and a repository that never
// fetched has no FETCH_HEAD, which git merge fails on too.
func TestFetchHeadMerges(t *testing.T) {
	requireGit(t)
	t.Parallel()
	repo := branchRepo(t)
	c := client(t)
	ctx := context.Background()
	if _, err := c.FetchHeadMerges(ctx, repo); !errors.Is(err, ErrGit) {
		t.Fatalf("FetchHeadMerges(no fetch) error = %v, want ErrGit", err)
	}
	runGit(t, repo, "fetch", "-q", ".", "feature", "evil")
	got, err := c.FetchHeadMerges(ctx, filepath.Join(repo, "src"))
	if err != nil {
		t.Fatalf("FetchHeadMerges: %v", err)
	}
	want := []string{revParse(t, repo, "feature"), revParse(t, repo, "evil")}
	if !slices.Equal(got, want) {
		t.Errorf("FetchHeadMerges = %q, want %q (feature, evil)", got, want)
	}
}

// revParse returns the object name rev resolves to in repo.
func revParse(t *testing.T, repo, rev string) string {
	t.Helper()
	cmd := procexec.Command("git", "rev-parse", "--verify", rev)
	cmd.Dir = repo
	cmd.Env = isolatedEnv(t)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse %s: %v", rev, err)
	}
	return strings.TrimSpace(string(out))
}

// TestFetchHeadMergesParse pins the FETCH_HEAD parse: the heads recorded for
// merging in order, not-for-merge lines skipped, and a line git could not
// merge refused.
func TestFetchHeadMergesParse(t *testing.T) {
	t.Parallel()
	a, b, c := strings.Repeat("a", 40), strings.Repeat("b", 64), strings.Repeat("c", 40)
	data := a + "\t\tbranch 'x' of ../r\n" + b + "\tnot-for-merge\tbranch 'y' of ../r\r\n\n" + c + "\t\t'z'\n"
	got, err := fetchHeadMerges(data)
	if err != nil || !slices.Equal(got, []string{a, c}) {
		t.Errorf("fetchHeadMerges = %q, %v; want %q", got, err, []string{a, c})
	}
	for _, bad := range []string{"no tab\n", "\t\tdesc\n", "XYZ\t\tdesc\n", "--output=x\t\tdesc\n"} {
		if _, err := fetchHeadMerges(bad); !errors.Is(err, ErrGit) {
			t.Errorf("fetchHeadMerges(%q) error = %v, want ErrGit", bad, err)
		}
	}
	if got, err := fetchHeadMerges(""); err != nil || len(got) != 0 {
		t.Errorf("fetchHeadMerges(empty) = %q, %v; want none", got, err)
	}
}

// TestNumstatPaths pins the parse of apply's -z --numstat output: the path
// after two tab-separated counts (a binary file's are "-"), a path holding a
// tab, and a rename's source and destination after an empty path.
func TestNumstatPaths(t *testing.T) {
	t.Parallel()
	out := "1\t2\tsrc/x.go\x00-\t-\tbin/a.png\x003\t0\tweird\tname\x000\t0\t\x00old.json\x00new.json\x00"
	want := []string{"src/x.go", "bin/a.png", "weird\tname", "old.json", "new.json"}
	if got := numstatPaths([]byte(out)); !slices.Equal(got, want) {
		t.Errorf("numstatPaths = %q, want %q", got, want)
	}
}

// assertSamePath compares paths by file identity, so symlinked temp roots
// (macOS /var -> /private/var) and Windows short names still match.
func assertSamePath(t *testing.T, got, want string) {
	t.Helper()
	gi, err := os.Stat(got)
	if err != nil {
		t.Fatalf("stat %q: %v", got, err)
	}
	wi, err := os.Stat(want)
	if err != nil {
		t.Fatalf("stat %q: %v", want, err)
	}
	if !os.SameFile(gi, wi) {
		t.Errorf("path = %q, want %q", got, want)
	}
}
