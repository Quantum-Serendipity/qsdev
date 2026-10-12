package gitrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"sync/atomic"
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
	var calls atomic.Int32
	var last Call
	c := Client{Runner: func(_ context.Context, call Call) ([]byte, error) {
		calls.Add(1)
		last = call
		return []byte("/r\n"), nil
	}}
	ctx := context.Background()
	dir := filepath.Join(string(filepath.Separator)+"r", "sub")

	tests := []struct {
		name string
		run  func()
		want []string
	}{
		{"Toplevel", func() { _, _ = c.Toplevel(ctx, dir) }, []string{"rev-parse", "--show-toplevel"}},
		{"ChangedPaths", func() { _, _ = c.ChangedPaths(ctx, dir, "evil", "", []string{"."}) },
			[]string{"--no-optional-locks", "-c", "core.fsmonitor=false", "diff", "--name-only", "-z",
				"--no-relative", "--no-ext-diff", "--no-textconv", "--end-of-options", "evil", "--", "."}},
		{"StashPaths", func() { _, _ = c.StashPaths(ctx, dir, "stash@{1}") },
			[]string{"--no-optional-locks", "stash", "show", "--name-only", "-z", "--include-untracked",
				"--end-of-options", "stash@{1}"}},
		{"TrackedPaths", func() { _, _ = c.TrackedPaths(ctx, dir) },
			[]string{"--no-optional-locks", "-c", "core.fsmonitor=false", "ls-files", "-z"}},
	}
	for _, tt := range tests {
		before := calls.Load()
		tt.run()
		if n := calls.Load() - before; n != 1 {
			t.Errorf("%s spawned %d git processes, want 1", tt.name, n)
		}
		if last.Dir != dir {
			t.Errorf("%s ran in %q, want %q", tt.name, last.Dir, dir)
		}
		if !slices.Equal(last.Args, tt.want) {
			t.Errorf("%s argv = %q, want %q", tt.name, last.Args, tt.want)
		}
	}

	if _, err := c.StashPaths(ctx, dir, "--output=x"); err == nil || calls.Load() != int32(len(tests)) {
		t.Errorf("StashPaths(option-shaped stash) = %v after %d calls, want a refusal without spawning", err, calls.Load())
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
