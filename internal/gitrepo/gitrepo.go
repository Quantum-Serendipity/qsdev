// Package gitrepo answers read-only questions about a git repository: where
// its top level and hooks directory are, which paths differ between two
// trees, which paths a stash holds and which paths the index tracks. Every query runs git with an argv
// array, in the caller's directory, with repository-pinning variables
// scrubbed and a short timeout, and none of them writes to the repository
// (--no-optional-locks, no fsmonitor, no external diff or textconv). Each
// argv is one of the procexec local-probe shapes.
package gitrepo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/procexec"
)

// Sentinel errors classify why a query failed.
var (
	// ErrTimeout means git did not answer within the query's timeout.
	ErrTimeout = errors.New("gitrepo: git query timed out")
	// ErrNotRepo means the directory is not inside a git repository.
	ErrNotRepo = errors.New("gitrepo: not a git repository")
	// ErrGit means git ran and refused the query, for example an unknown
	// revision or stash.
	ErrGit = errors.New("gitrepo: git query failed")
)

// Per-query timeouts. They sit well inside the hook evaluation deadline.
const (
	defaultRevParseTimeout = time.Second
	defaultQueryTimeout    = 2 * time.Second
	// waitDelay bounds how long a finished git may hold its output pipes
	// open through a lingering child.
	waitDelay = 500 * time.Millisecond
)

// Call is one git invocation: Args follow the git binary name, Dir is the
// working directory and Env holds variables added to the scrubbed process
// environment.
type Call struct {
	Dir  string
	Args []string
	Env  []string
}

// Runner runs a git Call and returns its standard output. A failure is
// reported wrapping ErrNotRepo or ErrGit when git itself refused; the Client
// turns a context deadline into ErrTimeout.
type Runner func(ctx context.Context, c Call) ([]byte, error)

// Client runs the read-only queries. The zero value runs real git through
// ExecRunner; tests inject a Runner to observe or fake the calls.
type Client struct {
	Runner Runner

	// revParseTimeout and queryTimeout override the default budgets in
	// tests; zero means the default.
	revParseTimeout time.Duration
	queryTimeout    time.Duration
}

// ScopedEnv drops the variables that pin git to a particular repository
// (set, for example, while qsdev runs inside a git hook), so git discovers
// the repository from its working directory instead.
func ScopedEnv(env []string) []string {
	out := make([]string, 0, len(env))
	for _, kv := range env {
		name, _, _ := strings.Cut(kv, "=")
		switch name {
		case "GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR", "GIT_INDEX_FILE":
			continue
		}
		out = append(out, kv)
	}
	return out
}

// ExecRunner runs git through procexec with the scrubbed environment and a
// C locale, so git's diagnostics classify the same way everywhere.
func ExecRunner(ctx context.Context, c Call) ([]byte, error) {
	cmd := procexec.CommandContext(ctx, "git", c.Args...)
	cmd.Dir = c.Dir
	cmd.Env = append(ScopedEnv(os.Environ()), "LC_ALL=C", "LANGUAGE=C")
	cmd.Env = append(cmd.Env, c.Env...)
	cmd.WaitDelay = waitDelay
	out, err := cmd.Output()
	if err == nil {
		return out, nil
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) || !exitErr.Exited() {
		return nil, fmt.Errorf("running git: %w", err)
	}
	stderr := strings.TrimSpace(string(exitErr.Stderr))
	if strings.Contains(strings.ToLower(stderr), "not a git repository") {
		return nil, fmt.Errorf("%w: %s", ErrNotRepo, stderr)
	}
	return nil, fmt.Errorf("%w (exit %d): %s", ErrGit, exitErr.ExitCode(), stderr)
}

// Toplevel returns the absolute top-level directory of the work tree that
// contains dir.
func (c Client) Toplevel(ctx context.Context, dir string) (string, error) {
	out, err := c.run(ctx, c.revParseTimeoutOrDefault(), Call{Dir: dir, Args: []string{"rev-parse", "--show-toplevel"}})
	if err != nil {
		return "", err
	}
	return absGitPath(out)
}

// HooksDir returns the directory git runs hooks from for the repository
// rooted exactly at dir. It follows worktree and submodule gitdir links and
// honours core.hooksPath. GIT_CEILING_DIRECTORIES stops git from walking up
// into an unrelated enclosing repository when dir's own .git is unusable.
func (c Client) HooksDir(ctx context.Context, dir string) (string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", dir, err)
	}
	out, err := c.run(ctx, c.revParseTimeoutOrDefault(), Call{
		Dir:  absDir,
		Args: []string{"rev-parse", "--path-format=absolute", "--git-path", "hooks"},
		Env:  []string{"GIT_CEILING_DIRECTORIES=" + filepath.Dir(absDir)},
	})
	if err != nil {
		return "", err
	}
	return absGitPath(out)
}

// ChangedPaths lists the paths, relative to the top level, that differ
// between from and to, limited to pathspec. An empty to compares from with
// the working tree; empty from and to compare the index with the working
// tree. Outside a work tree git diff would compare filesystem paths instead
// (no-index mode); the explicit "--" makes that form a usage error, so a
// caller outside a repository gets an error, never a list.
func (c Client) ChangedPaths(ctx context.Context, dir, from, to string, pathspec []string) ([]string, error) {
	args := []string{
		"--no-optional-locks", "-c", "core.fsmonitor=false", "diff",
		"--name-only", "-z", "--no-relative", "--no-ext-diff", "--no-textconv", "--end-of-options",
	}
	switch {
	case from != "" && to != "":
		args = append(args, from, to)
	case from != "":
		args = append(args, from)
	case to != "":
		return nil, fmt.Errorf("gitrepo: diff target %q needs a source revision", to)
	}
	args = append(append(args, "--"), pathspec...)
	out, err := c.run(ctx, c.queryTimeoutOrDefault(), Call{Dir: dir, Args: args})
	if err != nil {
		return nil, err
	}
	return splitNUL(out), nil
}

// StashPaths lists the paths, tracked and untracked, that the stash entry
// holds changes for.
func (c Client) StashPaths(ctx context.Context, dir, stash string) ([]string, error) {
	if stash == "" || strings.HasPrefix(stash, "-") {
		return nil, fmt.Errorf("gitrepo: invalid stash reference %q", stash)
	}
	out, err := c.run(ctx, c.queryTimeoutOrDefault(), Call{Dir: dir, Args: []string{
		"--no-optional-locks", "stash", "show", "--name-only", "-z", "--include-untracked",
		"--end-of-options", stash,
	}})
	if err != nil {
		return nil, err
	}
	return splitNUL(out), nil
}

// TrackedPaths lists the paths the index tracks under dir, relative to dir;
// run from the top level it lists the whole index.
func (c Client) TrackedPaths(ctx context.Context, dir string) ([]string, error) {
	out, err := c.run(ctx, c.queryTimeoutOrDefault(), Call{Dir: dir, Args: []string{
		"--no-optional-locks", "-c", "core.fsmonitor=false", "ls-files", "-z",
	}})
	if err != nil {
		return nil, err
	}
	return splitNUL(out), nil
}

// run executes call under timeout and reports a missed deadline as
// ErrTimeout, whatever error the runner returned for it.
func (c Client) run(ctx context.Context, timeout time.Duration, call Call) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	runner := c.Runner
	if runner == nil {
		runner = ExecRunner
	}
	out, err := runner(ctx, call)
	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return nil, fmt.Errorf("%w: git %s after %v", ErrTimeout, subcommand(call.Args), timeout)
	}
	if err != nil {
		return nil, fmt.Errorf("git %s in %s: %w", subcommand(call.Args), call.Dir, err)
	}
	return out, nil
}

func (c Client) revParseTimeoutOrDefault() time.Duration {
	if c.revParseTimeout > 0 {
		return c.revParseTimeout
	}
	return defaultRevParseTimeout
}

func (c Client) queryTimeoutOrDefault() time.Duration {
	if c.queryTimeout > 0 {
		return c.queryTimeout
	}
	return defaultQueryTimeout
}

// subcommand names the git subcommand in args for error messages, skipping
// global options and their values.
func subcommand(args []string) string {
	for i := 0; i < len(args); i++ {
		switch {
		case args[i] == "-c" || args[i] == "-C":
			i++
		case !strings.HasPrefix(args[i], "-"):
			return args[i]
		}
	}
	return "query"
}

// absGitPath parses a single absolute path git printed (forward slashes on
// every OS) into a clean native path.
func absGitPath(out []byte) (string, error) {
	p := filepath.FromSlash(strings.TrimSpace(string(out)))
	if p == "" || !filepath.IsAbs(p) {
		return "", fmt.Errorf("%w: unusable path %q", ErrGit, p)
	}
	return filepath.Clean(p), nil
}

// splitNUL splits git's -z output into its non-empty entries.
func splitNUL(out []byte) []string {
	var paths []string
	for p := range bytes.SplitSeq(out, []byte{0}) {
		if len(p) > 0 {
			paths = append(paths, string(p))
		}
	}
	return paths
}
