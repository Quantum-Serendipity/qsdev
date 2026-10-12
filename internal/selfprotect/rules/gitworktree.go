package rules

import (
	"context"
	"errors"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/gitrepo"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// This file holds the worktree-rewrite check of SP-007 (U18-02c). A git
// command such as `git checkout evil -- .` or `git reset --hard evil` writes
// tracked files into the working tree without naming them, so a protected
// file (.claude/settings.json) it overwrites never appears on the command
// line. The check parses each rewriting git command into the comparison git
// would apply and asks git, read-only, which paths that comparison changes
// (internal/gitrepo); a protected path among them denies. Commands that do
// not rewrite the working tree spawn nothing.

// rewriteSource is what a rewriting git command copies into the working
// tree.
type rewriteSource int

const (
	// fromIndex copies index entries over working-tree files.
	fromIndex rewriteSource = iota + 1
	// fromRev copies a revision's tree over working-tree files.
	fromRev
	// betweenRevs moves the work tree from HEAD to a revision, carrying
	// local changes along: only the paths that differ between the two
	// trees are written.
	betweenRevs
	// fromStash applies a stash entry.
	fromStash
)

// emptyTree is git's (SHA-1) empty tree, which git knows without storing it:
// moving to it removes every tracked file.
const emptyTree = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

// worktreeRewrite is the comparison a rewriting git command applies to the
// working tree.
type worktreeRewrite struct {
	source rewriteSource
	// rev is the revision (fromRev, betweenRevs) or stash entry (fromStash).
	rev string
	// paths limits the rewrite (a pathspec relative to the command's
	// directory); none means the whole tree.
	paths []string
	// orPaths, when set, is the other reading of an ambiguous `git checkout
	// <operand>...` without "--": when the first operand does not name a
	// revision, git restores all the operands as paths from the index.
	orPaths []string
	// alsoIndex, when set, adds the paths where the work tree differs from
	// the index: the command writes both comparisons (`git stash -k`).
	alsoIndex bool
	// unchecked, when set, is why the command writes files this check
	// cannot predict (an output prefix), so it is denied outright.
	unchecked string
}

// gitOptSpec describes a git subcommand's options for parseGitOpts.
type gitOptSpec struct {
	long       []string        // known long options, without --
	longValue  map[string]bool // long options taking the next word as value
	shortValue string          // short options taking a value
}

// gitOpts is a git subcommand's arguments split into options and operands.
type gitOpts struct {
	set      map[string]string // canonical option ("-f", "--force") -> value
	operands []string          // before "--"
	paths    []string          // after "--"
	dashdash bool
}

func (o gitOpts) has(names ...string) bool {
	for _, n := range names {
		if _, ok := o.set[n]; ok {
			return true
		}
	}
	return false
}

func (o gitOpts) value(names ...string) string {
	for _, n := range names {
		if v, ok := o.set[n]; ok {
			return v
		}
	}
	return ""
}

// parseGitOpts splits args per spec. A long option is canonicalised to the
// known option it unambiguously abbreviates, as git's option parser does
// (`--har` is --hard); a short cluster (`-qf`, `-bname`) is split into its
// options, the rest of the cluster after a value option being its value.
func parseGitOpts(args []string, spec gitOptSpec) gitOpts {
	o := gitOpts{set: map[string]string{}}
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case a == "--":
			o.dashdash = true
			o.paths = args[i+1:]
			return o
		case strings.HasPrefix(a, "--"):
			name, val, hasVal := strings.Cut(a[2:], "=")
			name = abbreviatedOption(name, spec.long)
			if !hasVal && spec.longValue[name] && i+1 < len(args) {
				i++
				val = args[i]
			}
			o.set["--"+name] = val
		case len(a) > 1 && a[0] == '-':
			for k := 1; k < len(a); k++ {
				opt := "-" + a[k:k+1]
				if strings.IndexByte(spec.shortValue, a[k]) < 0 {
					o.set[opt] = ""
					continue
				}
				val := a[k+1:]
				if val == "" && i+1 < len(args) {
					i++
					val = args[i]
				}
				o.set[opt] = val
				break
			}
		default:
			o.operands = append(o.operands, a)
		}
	}
	return o
}

// abbreviatedOption returns the known long option name abbreviates
// unambiguously, or name.
func abbreviatedOption(name string, known []string) string {
	match := ""
	for _, k := range known {
		if k == name {
			return k
		}
		if strings.HasPrefix(k, name) {
			if match != "" {
				return name
			}
			match = k
		}
	}
	if match == "" {
		return name
	}
	return match
}

var (
	checkoutSpec = gitOptSpec{
		long: []string{
			"force", "ours", "theirs", "track", "no-track", "guess", "no-guess", "detach", "orphan",
			"merge", "conflict", "patch", "quiet", "progress", "no-progress", "overwrite-ignore",
			"no-overwrite-ignore", "ignore-other-worktrees", "recurse-submodules", "no-recurse-submodules",
			"pathspec-from-file", "pathspec-file-nul", "overlay", "no-overlay", "ignore-skip-worktree-bits",
		},
		longValue:  map[string]bool{"orphan": true, "conflict": true, "pathspec-from-file": true},
		shortValue: "bB",
	}
	switchSpec = gitOptSpec{
		long: []string{
			"create", "force-create", "orphan", "detach", "guess", "no-guess", "force", "discard-changes",
			"merge", "conflict", "quiet", "progress", "no-progress", "track", "no-track",
			"ignore-other-worktrees", "recurse-submodules", "no-recurse-submodules",
		},
		longValue:  map[string]bool{"create": true, "force-create": true, "orphan": true, "conflict": true},
		shortValue: "cC",
	}
	restoreSpec = gitOptSpec{
		long: []string{
			"source", "staged", "worktree", "patch", "ours", "theirs", "merge", "conflict", "ignore-unmerged",
			"ignore-skip-worktree-bits", "overlay", "no-overlay", "quiet", "progress", "no-progress",
			"recurse-submodules", "no-recurse-submodules", "pathspec-from-file", "pathspec-file-nul",
		},
		longValue:  map[string]bool{"source": true, "conflict": true, "pathspec-from-file": true},
		shortValue: "s",
	}
	resetSpec = gitOptSpec{
		long: []string{
			"hard", "soft", "mixed", "keep", "merge", "quiet", "no-quiet", "refresh", "no-refresh",
			"recurse-submodules", "no-recurse-submodules", "intent-to-add", "patch",
			"pathspec-from-file", "pathspec-file-nul",
		},
		longValue: map[string]bool{"pathspec-from-file": true},
	}
	readTreeSpec = gitOptSpec{
		long: []string{
			"reset", "prefix", "index-output", "empty", "no-sparse-checkout", "trivial", "aggressive",
			"recurse-submodules", "no-recurse-submodules", "quiet", "dry-run", "exclude-per-directory",
		},
	}
	checkoutIndexSpec = gitOptSpec{
		long: []string{
			"all", "force", "quiet", "no-create", "prefix", "stage", "temp", "stdin",
			"ignore-skip-worktree-bits",
		},
	}
	stashSpec     = gitOptSpec{long: []string{"index", "quiet"}}
	stashPushSpec = gitOptSpec{
		long: []string{
			"patch", "staged", "keep-index", "no-keep-index", "include-untracked", "all", "quiet",
			"message", "pathspec-from-file", "pathspec-file-nul",
		},
		longValue:  map[string]bool{"message": true, "pathspec-from-file": true},
		shortValue: "m",
	}
)

// parseWorktreeRewrite returns the comparison git subcommand sub with
// arguments rest applies to the working tree, and false when it leaves the
// working tree alone (a new branch from HEAD, restore --staged, reset
// --soft/--mixed, stash list, any other subcommand).
func parseWorktreeRewrite(sub string, rest []string) (worktreeRewrite, bool) {
	switch sub {
	case "checkout-index":
		return parseCheckoutIndex(parseGitOpts(rest, checkoutIndexSpec))
	case "checkout":
		return parseCheckout(parseGitOpts(rest, checkoutSpec))
	case "switch":
		return parseSwitch(parseGitOpts(rest, switchSpec))
	case "restore":
		return parseRestore(parseGitOpts(rest, restoreSpec))
	case "reset":
		o := parseGitOpts(rest, resetSpec)
		if !o.has("--hard", "--keep", "--merge") {
			return worktreeRewrite{}, false
		}
		return worktreeRewrite{source: fromRev, rev: firstOr(o.operands, "HEAD")}, true
	case "read-tree":
		return parseReadTree(parseGitOpts(rest, readTreeSpec))
	case "stash":
		return parseStash(rest)
	}
	return worktreeRewrite{}, false
}

// prefixUnchecked is why a rewrite under an output prefix is denied.
const prefixUnchecked = "writes the files under a --prefix of the caller's choosing, which cannot be checked"

func parseCheckoutIndex(o gitOpts) (worktreeRewrite, bool) {
	paths := append(slices.Clone(o.operands), o.paths...)
	all := o.has("-a", "--all", "--stdin")
	if !all && len(paths) == 0 {
		return worktreeRewrite{}, false
	}
	if o.has("--prefix") {
		return worktreeRewrite{source: fromIndex, unchecked: prefixUnchecked}, true
	}
	if all {
		paths = nil
	}
	return worktreeRewrite{source: fromIndex, paths: paths}, true
}

func parseCheckout(o gitOpts) (worktreeRewrite, bool) {
	force := o.has("-f", "--force")
	previousBranch(o.operands)
	if o.dashdash || o.has("--pathspec-from-file") {
		paths := o.paths
		if o.has("--pathspec-from-file") {
			paths = nil
		}
		if len(o.operands) == 0 {
			return worktreeRewrite{source: fromIndex, paths: paths}, true
		}
		return worktreeRewrite{source: fromRev, rev: o.operands[0], paths: paths}, true
	}
	if o.has("-b", "-B", "--orphan") {
		// A new branch: the work tree moves only to an explicit start point
		// (an unborn --orphan branch too: git adjusts the index and work tree
		// to it).
		if len(o.operands) == 0 {
			return worktreeRewrite{}, false
		}
		return switchTo(o.operands[0], force), true
	}
	patch := o.has("-p", "--patch")
	if len(o.operands) == 0 {
		switch {
		case force:
			return worktreeRewrite{source: fromRev, rev: "HEAD"}, true
		case patch:
			// `checkout -p` copies hunks of the index over the whole tree.
			return worktreeRewrite{source: fromIndex}, true
		}
		return worktreeRewrite{}, false
	}
	if patch {
		// `checkout -p [<tree-ish>] [<pathspec>...]` copies hunks of the
		// tree-ish, or of the index when the first operand names none.
		return worktreeRewrite{source: fromRev, rev: o.operands[0], paths: o.operands[1:], orPaths: o.operands}, true
	}
	// No "--": git takes the first operand as a revision when it names one,
	// and every operand as a path otherwise (`:/`, `pkg`, `src/..`). A ref
	// name cannot start with "." nor hold a glob, so such an operand starts
	// the paths; any other is judged as a revision first and, when it names
	// none, as the paths (orPaths).
	if first := o.operands[0]; strings.HasPrefix(first, ".") || hasGlobMeta(first) {
		return worktreeRewrite{source: fromIndex, paths: o.operands}, true
	}
	rw := switchTo(o.operands[0], force)
	if len(o.operands) > 1 {
		rw = worktreeRewrite{source: fromRev, rev: o.operands[0], paths: o.operands[1:]}
	}
	rw.orPaths = o.operands
	return rw, true
}

// previousBranch rewrites a leading "-" operand, git's shorthand for the
// previous branch, to the revision it names.
func previousBranch(operands []string) {
	if len(operands) > 0 && operands[0] == "-" {
		operands[0] = "@{-1}"
	}
}

// switchTo is the comparison of moving the work tree to rev: with force,
// every file is reset to rev; without, only the files that differ between
// HEAD and rev are written.
func switchTo(rev string, force bool) worktreeRewrite {
	if force {
		return worktreeRewrite{source: fromRev, rev: rev}
	}
	return worktreeRewrite{source: betweenRevs, rev: rev}
}

func parseSwitch(o gitOpts) (worktreeRewrite, bool) {
	if o.has("--orphan") {
		// switch --orphan removes every tracked file.
		return worktreeRewrite{source: betweenRevs, rev: emptyTree}, true
	}
	if len(o.operands) == 0 {
		return worktreeRewrite{}, false
	}
	previousBranch(o.operands)
	return switchTo(o.operands[0], o.has("-f", "--force", "--discard-changes")), true
}

func parseRestore(o gitOpts) (worktreeRewrite, bool) {
	staged := o.has("-S", "--staged")
	worktree := o.has("-W", "--worktree")
	if staged && !worktree {
		return worktreeRewrite{}, false
	}
	paths := append(slices.Clone(o.operands), o.paths...)
	if o.has("--pathspec-from-file") {
		paths = nil
	} else if len(paths) == 0 && !o.has("-p", "--patch") {
		return worktreeRewrite{}, false // git requires a pathspec, except with --patch
	}
	rev := o.value("-s", "--source")
	if rev == "" && staged {
		rev = "HEAD"
	}
	if rev == "" {
		return worktreeRewrite{source: fromIndex, paths: paths}, true
	}
	return worktreeRewrite{source: fromRev, rev: rev, paths: paths}, true
}

func parseReadTree(o gitOpts) (worktreeRewrite, bool) {
	if !o.has("-u") || o.has("-n", "--dry-run") {
		return worktreeRewrite{}, false
	}
	tree := ""
	switch {
	case len(o.operands) > 0:
		tree = o.operands[len(o.operands)-1]
	case o.has("--empty"):
		tree = emptyTree
	default:
		return worktreeRewrite{}, false
	}
	if o.has("--prefix") {
		return worktreeRewrite{source: fromRev, rev: tree, unchecked: prefixUnchecked}, true
	}
	return worktreeRewrite{source: fromRev, rev: tree}, true
}

func parseStash(rest []string) (worktreeRewrite, bool) {
	if len(rest) == 0 || isFlag(rest[0]) {
		return parseStashPush(parseGitOpts(rest, stashPushSpec), true)
	}
	switch rest[0] {
	case "push":
		return parseStashPush(parseGitOpts(rest[1:], stashPushSpec), true)
	case "save":
		// save's operands are its message, not a pathspec.
		return parseStashPush(parseGitOpts(rest[1:], stashPushSpec), false)
	}
	o := parseGitOpts(rest[1:], stashSpec)
	switch rest[0] {
	case "pop", "apply":
		return worktreeRewrite{source: fromStash, rev: firstOr(o.operands, "stash@{0}")}, true
	case "branch":
		// stash branch <name> [<stash>]
		ref := "stash@{0}"
		if len(o.operands) > 1 {
			ref = o.operands[1]
		}
		return worktreeRewrite{source: fromStash, rev: ref}, true
	}
	return worktreeRewrite{}, false
}

// parseStashPush returns the comparison of `git stash [push|save]`: it resets
// the work tree to HEAD, so every path where the work tree differs from HEAD
// is written (after a `git reset --soft` moved HEAD, the protected file). With
// --keep-index, --staged or --patch (which keeps the index) the work tree is
// then set to the index, so the paths where it differs from the index are
// written too (alsoIndex). withPaths reports whether the operands are a
// pathspec (push) rather than a message (save).
func parseStashPush(o gitOpts, withPaths bool) (worktreeRewrite, bool) {
	rw := worktreeRewrite{source: fromRev, rev: "HEAD"}
	if withPaths && !o.has("--pathspec-from-file") {
		rw.paths = append(slices.Clone(o.operands), o.paths...)
	}
	rw.alsoIndex = o.has("-k", "--keep-index", "-S", "--staged", "-p", "--patch") && !o.has("--no-keep-index")
	return rw, true
}

func firstOr(s []string, def string) string {
	if len(s) > 0 {
		return s[0]
	}
	return def
}

// gitReadOnlySubcommands only read the repository, so one may precede a
// rewrite on its line without changing what the rewrite copies.
var gitReadOnlySubcommands = map[string]bool{
	"status": true, "log": true, "show": true, "diff": true, "rev-parse": true, "ls-files": true,
	"ls-tree": true, "cat-file": true, "describe": true, "blame": true, "grep": true, "shortlog": true,
}

// isLocationChange reports whether a scanned command only changes the
// working directory (PowerShell's location cmdlets are renamed to these).
func isLocationChange(name string) bool {
	return name == "cd" || name == "pushd" || name == "popd"
}

// isRepoPinVar reports whether the variable chooses the repository, index or
// work tree git uses, so a query run from the command's directory would ask
// another repository.
func isRepoPinVar(name string) bool {
	switch name {
	case "GIT_DIR", "GIT_WORK_TREE", "GIT_INDEX_FILE", "GIT_COMMON_DIR", "GIT_OBJECT_DIRECTORY":
		return true
	}
	return false
}

// worktreeRewriteReason returns why a shell command's git would overwrite a
// protected file in the working tree, or cannot be checked, or "".
func worktreeRewriteReason(ctx *EvalContext) string {
	if !cmdscan.IsShellTool(ctx.ToolName) || ctx.Command == "" {
		return ""
	}
	if isPowerShell(ctx) {
		scs := ctx.psScanned()
		return ctx.rewriteWalk(scs, rewriteLine{psLocation: slices.ContainsFunc(scs, func(sc scannedCommand) bool {
			return isLocationChange(sc.Name)
		})})
	}
	scs, err := ctx.scannedCommands()
	if err != nil {
		return "" // GIT-001 denies a line that runs git but cannot be parsed
	}
	return ctx.rewriteWalk(scs, rewriteLine{})
}

// rewriteLine is what rewriteWalk knows of the line around the commands it
// walks.
type rewriteLine struct {
	// depth counts the shell scripts the commands are nested in.
	depth int
	// preceded reports that a command other than a directory change or a
	// read-only git runs before them.
	preceded bool
	// psLocation reports a PowerShell line that changes location.
	psLocation bool
}

// rewriteWalk checks the rewriting git commands of scs in order. Any command
// but cd, pushd, popd and read-only git that precedes a rewrite on its line
// may change the source the rewrite copies (the index, a fetched ref) before
// it runs, so such a rewrite is allowed only in a repository that tracks no
// protected file (askGit). A shell script (`sh -c`, eval) is walked in the
// directory its shell runs in.
func (ctx *EvalContext) rewriteWalk(scs []scannedCommand, line rewriteLine) string {
	for _, sc := range scs {
		words := append([]string{sc.Name}, sc.Args...)
		if run, ok := cmdscan.Script(words); ok && !isPowerShell(ctx) {
			if reason := ctx.scriptRewriteReason(sc, run, line); reason != "" {
				return reason
			}
			line.preceded = true
			continue
		}
		args, ok := gitInvocation(words)
		if !ok {
			line.preceded = line.preceded || !isLocationChange(sc.Name)
			continue
		}
		target := splitGitArgs(args)
		rw, ok := parseWorktreeRewrite(target.sub, target.rest)
		if !ok {
			line.preceded = line.preceded || !gitReadOnlySubcommands[target.sub]
			continue
		}
		if reason := ctx.checkRewrite(sc, target, rw, line); reason != "" {
			return reason
		}
		line.preceded = true
	}
	return ""
}

// scriptRewriteReason walks the script a shell command runs, from the
// directory that shell starts in, as part of line. A shell that reads startup
// files first may start anywhere.
func (ctx *EvalContext) scriptRewriteReason(sc scannedCommand, run cmdscan.ScriptRun, line rewriteLine) string {
	if line.depth >= maxGitScriptDepth {
		return "" // GIT-001 denies git nested this deep
	}
	sub := &EvalContext{ToolName: "Bash", Command: run.Script, Ctx: ctx.Ctx, gitRunner: ctx.gitRunner}
	if !sc.cwdUnknown && !run.ReadsStartup {
		sub.CWD = sc.cwd
	}
	scs, err := sub.scannedCommands()
	if err != nil {
		return ""
	}
	line.depth++
	return sub.rewriteWalk(scs, line)
}

// checkRewrite judges one rewriting git command.
func (ctx *EvalContext) checkRewrite(sc scannedCommand, target gitArgs, rw worktreeRewrite, line rewriteLine) string {
	what := "git " + target.sub
	if target.sub == "stash" && len(target.rest) > 0 && !isFlag(target.rest[0]) {
		what += " " + target.rest[0] // push, save, pop, apply or branch
	}
	switch {
	case line.psLocation:
		return what + " rewrites working-tree files on a PowerShell line that changes location; run it in its own call, from the session directory or with git -C, so the files it writes can be checked"
	case rw.unchecked != "":
		return what + " " + rw.unchecked
	case target.pinned || envAssignment(sc.Command, isRepoPinVar) != "":
		return what + " rewrites working-tree files of a repository chosen by option or variable, which cannot be checked"
	case sc.HasExpansion:
		return what + " rewrites working-tree files named through a shell expansion, which cannot be checked"
	}
	dir, ok := rewriteDir(sc, target.chdirs)
	if !ok {
		return what + " rewrites working-tree files in a directory that is not known, so they cannot be checked"
	}
	if ctx.Ctx == nil {
		return what + " rewrites working-tree files, and this check has no deadline to ask git which"
	}
	verdict, err := ctx.askGit(dir, rw, line.preceded)
	switch {
	case errors.Is(err, gitrepo.ErrTimeout):
		return what + " rewrites working-tree files, and git did not answer in time to tell which"
	case err != nil:
		return "" // not a repository or an unknown revision: git itself fails
	case verdict.tracked != "":
		return what + " rewrites working-tree files after another command on the same line, which may change what it copies, in a repository that tracks protected " + verdict.tracked + "; run it in its own call so the files it writes can be checked"
	case verdict.overwritten != "":
		return what + " would overwrite protected " + verdict.overwritten + " in the working tree"
	}
	return ""
}

// gitVerdict is what git reported about a rewrite: the protected path it
// would overwrite, or, for a rewrite preceded on its line, a protected path
// the repository tracks.
type gitVerdict struct {
	overwritten, tracked string
}

// askGit asks git, read-only and in dir, about rewrite rw. A rewrite preceded
// on its line by another command may copy something that command changes
// (the index, a fetched ref), so what it writes is known only in a
// repository that tracks no protected file; any other rewrite is judged by
// the paths it would write. When the first operand of an ambiguous checkout
// names no revision, git reads its operands as paths, and so does askGit.
func (ctx *EvalContext) askGit(dir string, rw worktreeRewrite, preceded bool) (gitVerdict, error) {
	client := gitrepo.Client{Runner: ctx.gitRunner}
	top, err := client.Toplevel(ctx.Ctx, dir)
	if err != nil {
		return gitVerdict{}, err
	}
	if preceded {
		tracked, err := client.TrackedPaths(ctx.Ctx, top)
		return gitVerdict{tracked: firstProtected(top, tracked)}, err
	}
	paths, err := rewrittenPaths(ctx.Ctx, client, dir, rw)
	if errors.Is(err, gitrepo.ErrGit) && rw.orPaths != nil {
		paths, err = rewrittenPaths(ctx.Ctx, client, dir, worktreeRewrite{source: fromIndex, paths: rw.orPaths})
	}
	return gitVerdict{overwritten: firstProtected(top, paths)}, err
}

// firstProtected returns the first of paths, relative to top, that canon
// protects, or "".
func firstProtected(top string, paths []string) string {
	for _, p := range paths {
		if protected, _ := canon.IsProtected(filepath.Join(top, filepath.FromSlash(p))); protected {
			return p
		}
	}
	return ""
}

// rewriteDir returns the directory a git command runs in: the command's
// working directory, moved by each -C. It is false when that directory is
// not known.
func rewriteDir(sc scannedCommand, chdirs []string) (string, bool) {
	if sc.cwdUnknown || !isRooted(sc.cwd) {
		return "", false
	}
	dir := sc.cwd
	for _, d := range chdirs {
		if d = expandTilde(d); isRooted(d) {
			dir = filepath.Clean(d)
		} else {
			dir = filepath.Join(dir, d)
		}
	}
	return dir, true
}

// rewrittenPaths asks git which paths, relative to the toplevel, the rewrite
// would write.
func rewrittenPaths(ctx context.Context, client gitrepo.Client, dir string, rw worktreeRewrite) ([]string, error) {
	if rw.alsoIndex {
		fromHead, err := rewrittenPaths(ctx, client, dir, worktreeRewrite{source: rw.source, rev: rw.rev, paths: rw.paths})
		if err != nil {
			return nil, err
		}
		fromIdx, err := client.ChangedPaths(ctx, dir, "", "", rw.paths)
		return append(fromHead, fromIdx...), err
	}
	switch rw.source {
	case fromIndex:
		return client.ChangedPaths(ctx, dir, "", "", rw.paths)
	case fromRev:
		return client.ChangedPaths(ctx, dir, rw.rev, "", rw.paths)
	case betweenRevs:
		return client.ChangedPaths(ctx, dir, "HEAD", rw.rev, rw.paths)
	case fromStash:
		return client.StashPaths(ctx, dir, rw.rev)
	}
	return nil, nil
}
