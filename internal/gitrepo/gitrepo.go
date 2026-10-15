// Package gitrepo answers read-only questions about a git repository: where
// its top level and hooks directory are, which paths differ between two
// trees, which paths a stash, a commit or a patch file changes, which
// paths the index tracks, which untracked paths git clean would remove and
// which heads FETCH_HEAD records for merging. Every query runs git with an argv array, in the
// caller's directory, with repository-pinning variables scrubbed and a short
// timeout, and none of them writes to the repository (--no-optional-locks,
// no fsmonitor, no external diff or textconv). A rename is listed as both
// its source and its destination (--no-renames), so a path a change removes
// is never hidden behind the name it moves to. Each argv is one of the
// procexec local-probe shapes.
package gitrepo

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
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

// FetchHeadMerges returns the object names of the heads the last fetch in
// the repository at dir recorded for merging: the FETCH_HEAD lines not marked
// not-for-merge. `git merge FETCH_HEAD` merges every one of them (an octopus
// when there are several), not only the first, which is all a revision
// lookup of FETCH_HEAD resolves. A repository without a FETCH_HEAD, or one
// holding a line git could not merge, is an ErrGit: git merge fails on it
// too.
func (c Client) FetchHeadMerges(ctx context.Context, dir string) ([]string, error) {
	out, err := c.run(ctx, c.revParseTimeoutOrDefault(), Call{
		Dir: dir, Args: []string{"rev-parse", "--path-format=absolute", "--git-path", "FETCH_HEAD"},
	})
	if err != nil {
		return nil, err
	}
	file, err := absGitPath(out)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("%w: reading %s: %w", ErrGit, file, err)
	}
	return fetchHeadMerges(string(data))
}

// fetchHeadMerges parses FETCH_HEAD, whose lines are
// "<object>\t[not-for-merge]\t<description>", into the objects of the lines
// recorded for merging.
func fetchHeadMerges(data string) ([]string, error) {
	var heads []string
	for line := range strings.Lines(data) {
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			continue
		}
		oid, rest, ok := strings.Cut(line, "\t")
		if !ok || oid == "" || strings.Trim(oid, "0123456789abcdef") != "" {
			return nil, fmt.Errorf("%w: malformed FETCH_HEAD line %q", ErrGit, line)
		}
		if marker, _, _ := strings.Cut(rest, "\t"); marker == "" {
			heads = append(heads, oid)
		}
	}
	return heads, nil
}

// ChangedPaths lists the paths, relative to the top level, that differ
// between from and to, limited to pathspec. An empty to compares from with
// the working tree; empty from and to compare the index with the working
// tree. from may instead be a range with an empty to: "A..B" compares the
// two trees, and "A...B" compares B with the merge base of A and B, which is
// what merging B into A writes. Outside a work tree git diff would compare
// filesystem paths instead (no-index mode); the explicit "--" makes that form
// a usage error, so a caller outside a repository gets an error, never a
// list.
func (c Client) ChangedPaths(ctx context.Context, dir, from, to string, pathspec []string) ([]string, error) {
	args := []string{
		"--no-optional-locks", "-c", "core.fsmonitor=false", "diff",
		"--name-only", "-z", "--no-relative", "--no-renames", "--no-ext-diff", "--no-textconv", "--end-of-options",
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
		"--no-optional-locks", "stash", "show", "--name-only", "-z", "--no-renames", "--include-untracked",
		"--end-of-options", stash,
	}})
	if err != nil {
		return nil, err
	}
	return splitNUL(out), nil
}

// CommitPaths lists the paths, relative to the top level, that the commits
// revs name change against their parents (against each parent of a merge),
// which is what cherry-pick and revert replay. A single revision names one
// commit; a range ("a..b", "^a b") names the commits it walks, as
// cherry-pick's own revision walk does (--no-walk has no effect on a range).
func (c Client) CommitPaths(ctx context.Context, dir string, revs []string) ([]string, error) {
	if len(revs) == 0 {
		return nil, fmt.Errorf("gitrepo: no commits to list")
	}
	args := slices.Concat([]string{
		"--no-optional-locks", "log", "--format=", "--name-only", "-z", "--no-relative", "--no-renames",
		"--no-show-signature", "--no-walk", "--diff-merges=separate", "--end-of-options",
	}, revs, []string{"--"})
	out, err := c.run(ctx, c.queryTimeoutOrDefault(), Call{Dir: dir, Args: args})
	if err != nil {
		return nil, err
	}
	return splitNUL(out), nil
}

// PatchPaths lists the paths the patch files would change if git apply (or
// git am, which applies each mail through it) applied them in dir. git apply
// reads a mailbox as am writes it. strip is the -p count ("" for git's
// default of 1) and directory the --directory prefix ("" for none). In a
// repository the paths are relative to the top level, outside one to dir.
// git apply --numstat applies nothing; it names a rename or copy by its
// destination only, so the patches are read a second time in reverse (-R),
// which names the source.
func (c Client) PatchPaths(ctx context.Context, dir string, files []string, strip, directory string) ([]string, error) {
	if len(files) == 0 {
		return nil, fmt.Errorf("gitrepo: no patch files to read")
	}
	if strings.Trim(strip, "0123456789") != "" {
		return nil, fmt.Errorf("gitrepo: invalid strip count %q", strip)
	}
	var opts []string
	if strip != "" {
		opts = append(opts, "-p"+strip)
	}
	if directory != "" {
		opts = append(opts, "--directory="+directory)
	}
	var paths []string
	for _, reverse := range [][]string{nil, {"-R"}} {
		args := slices.Concat([]string{"--no-optional-locks", "apply", "--numstat", "-z"}, reverse, opts, []string{"--"}, files)
		out, err := c.run(ctx, c.queryTimeoutOrDefault(), Call{Dir: dir, Args: args})
		if err != nil {
			return nil, err
		}
		paths = append(paths, numstatPaths(out)...)
	}
	return paths, nil
}

// TrackedPaths lists the paths, relative to the top level, that the index
// tracks and pathspec, read from dir, matches (what `git rm -r <pathspec>`
// removes); with no pathspec it lists those under dir, so run from the top
// level it lists the whole index. The explicit "--" keeps a pathspec named
// like an option a pathspec.
func (c Client) TrackedPaths(ctx context.Context, dir string, pathspec []string) ([]string, error) {
	out, err := c.run(ctx, c.queryTimeoutOrDefault(), Call{Dir: dir, Args: slices.Concat([]string{
		"--no-optional-locks", "-c", "core.fsmonitor=false", "ls-files", "-z", "--full-name", "--",
	}, pathspec)})
	if err != nil {
		return nil, err
	}
	return splitNUL(out), nil
}

// CleanOptions are the git clean options that decide which untracked files
// it removes.
type CleanOptions struct {
	// Force is how many times -f is given: twice also removes untracked
	// nested repositories.
	Force int
	// Dirs (-d) removes untracked directories as a whole.
	Dirs bool
	// Ignored (-x) also removes ignored files; OnlyIgnored (-X) removes only
	// those.
	Ignored, OnlyIgnored bool
	// Excludes are -e patterns, which keep matching files.
	Excludes []string
}

// cleanRemovePrefix and cleanSkipPrefix start the lines of `git clean
// --dry-run` output (LC_ALL=C).
const (
	cleanRemovePrefix = "Would remove "
	cleanSkipPrefix   = "Would skip repository "
)

// CleanPaths lists the paths, relative to dir, that git clean with opts and
// pathspec would remove when run in dir: `git clean --dry-run` with the same
// options, which removes nothing. A directory it would remove as a whole ends
// in "/". Paths are printed unquoted unless they hold a control character,
// '"' or '\', which git C-quotes.
func (c Client) CleanPaths(ctx context.Context, dir string, opts CleanOptions, pathspec []string) ([]string, error) {
	args := []string{"--no-optional-locks", "-c", "core.fsmonitor=false", "-c", "core.quotePath=false", "clean", "--dry-run"}
	if opts.Dirs {
		args = append(args, "-d")
	}
	switch {
	case opts.OnlyIgnored:
		args = append(args, "-X")
	case opts.Ignored:
		args = append(args, "-x")
	}
	for range min(opts.Force, 2) {
		args = append(args, "-f")
	}
	for _, e := range opts.Excludes {
		args = append(args, "--exclude="+e)
	}
	args = append(append(args, "--"), pathspec...)
	out, err := c.run(ctx, c.queryTimeoutOrDefault(), Call{Dir: dir, Args: args})
	if err != nil {
		return nil, err
	}
	return cleanOutputPaths(out)
}

// cleanOutputPaths parses `git clean --dry-run` output into the paths it
// would remove, C-unquoted; a skipped repository is not one.
func cleanOutputPaths(out []byte) ([]string, error) {
	var paths []string
	for line := range strings.Lines(string(out)) {
		line = strings.TrimRight(line, "\r\n")
		if strings.HasPrefix(line, cleanSkipPrefix) {
			continue
		}
		p, ok := strings.CutPrefix(line, cleanRemovePrefix)
		if !ok || p == "" {
			continue
		}
		if strings.HasPrefix(p, `"`) {
			unquoted, err := strconv.Unquote(p)
			if err != nil {
				return nil, fmt.Errorf("%w: unreadable clean path %s", ErrGit, p)
			}
			p = unquoted
		}
		paths = append(paths, p)
	}
	return paths, nil
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

// numstatPaths returns the paths of git's -z --numstat output, whose entries
// are "added<TAB>deleted<TAB>path". An entry with an empty path is followed
// by a rename's source and destination as entries of their own.
func numstatPaths(out []byte) []string {
	var paths []string
	for _, entry := range splitNUL(out) {
		_, rest, _ := strings.Cut(entry, "\t")
		if _, path, ok := strings.Cut(rest, "\t"); ok {
			entry = path
		}
		if entry != "" {
			paths = append(paths, entry)
		}
	}
	return paths
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
