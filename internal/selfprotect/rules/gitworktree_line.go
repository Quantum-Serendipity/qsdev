package rules

import (
	"errors"
	"path"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/gitrepo"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// This file holds what SP-007's worktree-rewrite check (gitworktree.go)
// knows of the commands that run before a rewrite on its line. A rewrite is
// judged by asking git, before the line runs, which paths it would write;
// an earlier command that changes the index, a ref, the stash or a patch
// file can make that answer stale. Each earlier command is recorded by what
// it may change, so a rewrite is judged on its own when nothing before it
// changes what it reads (`echo start; git checkout -- main.go`), with the
// paths a modelled git command adds when one does (`git commit -am x && git
// switch main`), and by whether the repository tracks a protected path at
// all otherwise (`go test ./... && git checkout main`).

// latestStash is the stash entry a stash pop, apply or branch takes by
// default, and the one a stash push saves.
const latestStash = "stash@{0}"

// rewriteLine is what rewriteWalk knows of the line around the commands it
// walks.
type rewriteLine struct {
	// depth counts the shell scripts the commands are nested in.
	depth int
	// changed reports that a command before them may change the index, HEAD,
	// a ref or the working tree in a way the model does not follow: an
	// unknown program or script, git behind a wrapper, a rewrite.
	changed bool
	// stashChanged reports that a command before them may change the stash
	// list (refs/stash and its reflog), which most git commands never touch.
	stashChanged bool
	// committed reports that a git commit runs before them: HEAD and the
	// current branch move to a commit whose tree differs from the old HEAD's
	// only in the local changes it records.
	committed bool
	// added reports that a git add runs before them: it copies working-tree
	// content into the index, and may start tracking an untracked file, or
	// with --force (addedIgnored) an ignored one too.
	added, addedIgnored bool
	// chmods are the git add --chmod commands before them: each changes the
	// mode of the tracked paths it names in the index alone.
	chmods []chmodAdd
	// branched are the branches a git switch -c or checkout -b before them
	// created at HEAD and switched to (branchedHere): HEAD names the same
	// commit, but the new branch and the branch-relative revisions
	// (`@{-1}`, `@{u}`) name others.
	branched []string
	// stashPushed reports that a git stash push runs before them: it saves
	// the local changes as stash@{0} and resets them, which its own check
	// judged as a rewrite.
	stashPushed bool
	// psLocation reports a PowerShell line that changes location.
	psLocation bool
}

// changeAll records a command that may change anything a later rewrite
// reads.
func (l *rewriteLine) changeAll() {
	l.changed, l.stashChanged = true, true
}

// inertCommand reports whether sc, the i-th command of ctx's line and a
// command other than git, cannot change what a later rewrite reads: a
// directory change, or a read-only command (cmdscan.IsSafeReadCommand: echo,
// printf, true, test, cat, ...) that sets no variable and writes no file
// through a redirect. On a PowerShell line it is a command that only reads
// (cmdscan.PSCommand.IsRead: Write-Output, Get-Content, Test-Path, ...).
func (ctx *EvalContext) inertCommand(i int, sc scannedCommand) bool {
	if isLocationChange(sc.Name) {
		return true
	}
	if isPowerShell(ctx) {
		return ctx.powerShell().cmds[i].IsRead()
	}
	return len(sc.WriteRedirects) == 0 && cmdscan.IsSafeReadCommand(sc.Command)
}

// ranGit records what the git command sc runs (args, split as target) may
// change for a later rewrite on its line; rewrites reports that it is itself
// the rewrite rw, already judged. A read-only git changes nothing. A git run
// directly (directGit) is modelled: a stash push saves stash@{0}, a commit
// and an add are recorded for the rewrites after them (ranCommit, ranAdd),
// a branch created at HEAD is recorded (branchedHere), and any other git may
// change the index, HEAD, refs or the working tree, but the stash only when
// it is not one of stashKeepingGit (keepsStash). A stash pop, apply or branch
// also drops or shifts stash entries. A git behind a wrapper, a variable, a
// redirect, an expansion (but in a commit's option value) or a global option
// may do anything.
func (l *rewriteLine) ranGit(sc scannedCommand, args []string, target gitArgs, rw worktreeRewrite, rewrites bool) {
	commit := !rewrites && target.sub == "commit"
	var opts gitOpts
	marked := false
	if commit {
		var rest []string
		rest, marked = markedArgs(sc, target.rest)
		opts = parseGitOpts(rest, commitSpec)
	}
	switch {
	case !rewrites && gitReadOnly(target):
	case !directGit(sc, args, target) || sc.HasExpansion && (!commit || !marked || !expansionsAreValues(sc, opts)):
		l.changeAll()
	case rewrites && rw.source == fromStash:
		l.changeAll()
	case rewrites && target.sub == "stash":
		l.stashPushed = true
	case commit:
		l.ranCommit(opts)
	case !rewrites && (target.sub == "add" || target.sub == "stage"):
		l.ranAdd(sc, parseGitOpts(target.rest, addSpec))
	default:
		if branch, ok := branchedHere(target); ok && !rewrites {
			l.branched = append(l.branched, branch)
			return
		}
		l.changed = true
		l.stashChanged = l.stashChanged || !keepsStash(target)
	}
}

// directGit reports whether sc runs git itself on the repository of its
// directory, with nothing that changes what git does: no wrapper, variable,
// write redirect or global option before the subcommand, and no expansion
// in the command word. An expansion in an argument is judged by the caller
// (expansionsAreValues).
func directGit(sc scannedCommand, args []string, target gitArgs) bool {
	return path.Base(sc.Name) == "git" && !sc.NameHasExpansion && len(args) > 0 && len(args) == len(sc.Args) &&
		args[0] == target.sub && len(sc.Assigns) == 0 && !sc.AssignsDynamic && len(sc.WriteRedirects) == 0
}

// expansionsAreValues reports whether every argument of the direct git sc
// built from an expansion is the value of an option of its subcommand (o,
// parsed from the arguments markedArgs marked) and stays one word (not
// cmdscan.Command.SplitArgs): the separate value word (`git commit -am
// "$MSG"`), or a value attached to its option in a word whose expansions all
// follow the option (`--message="$MSG"`, `-m"$MSG"`; markedArgs cut each
// word at its first expansion). Such a value cannot become an option or an
// operand, so the command is what its literal words say; `git commit -am
// $MSG` or `-m$MSG` may split into `--amend`. Options the shell may expand
// into others (gitOpts.unmodelled) are never trusted.
func expansionsAreValues(sc scannedCommand, o gitOpts) bool {
	if len(sc.SplitArgs) > 0 || len(sc.ExpandedArgIndexes) == 0 || o.unmodelled {
		return false
	}
	return !slices.ContainsFunc(sc.ExpandedArgIndexes, func(i int) bool {
		j := i - 1 // Args[0] is the subcommand
		return !slices.Contains(o.valueArgs, j) && !slices.Contains(o.attachedArgs, j)
	})
}

// expansionMark stands in markedArgs for the value of an expansion: a NUL,
// which no shell word holds.
const expansionMark = "\x00"

// markedArgs returns the arguments rest after the subcommand of the direct
// git sc with each word built from an expansion cut to its literal text
// before the expansion (cmdscan.Command.ExpandedArgPrefixes) and
// expansionMark, so an expansion that may render as nothing still reads as a
// value attached to its option (`-m"$MSG" -a` is `-m<mark> -a`, not -m with
// the value -a), and reports whether it marked them. It returns rest and
// false when the scan does not know the literal text of each expanded word,
// or when one already holds expansionMark, which the marking could not tell
// from its own.
func markedArgs(sc scannedCommand, rest []string) ([]string, bool) {
	prefixes := sc.ExpandedArgPrefixes
	if len(prefixes) != len(sc.ExpandedArgIndexes) || len(rest) != len(sc.Args)-1 ||
		slices.ContainsFunc(prefixes, func(p string) bool { return strings.Contains(p, expansionMark) }) {
		return rest, false
	}
	marked := slices.Clone(rest)
	for k, i := range sc.ExpandedArgIndexes {
		if i >= 1 {
			marked[i-1] = prefixes[k] + expansionMark
		}
	}
	return marked, true
}

// branchedHere returns the branch the git command target creates at HEAD
// and switches to, leaving the index and the working tree alone: `git
// switch -c <name>` or `git checkout -b <name>` (or the -C / -B reset form)
// with no start point, no path and no other option than quiet, progress
// and --no-track. HEAD keeps its commit, so only revisions naming the new
// branch or relative to the current one (namesBranched) change.
func branchedHere(target gitArgs) (string, bool) {
	var spec gitOptSpec
	var create []string
	switch target.sub {
	case "switch":
		spec, create = switchSpec, []string{"-c", "--create", "-C", "--force-create"}
	case "checkout":
		spec, create = checkoutSpec, []string{"-b", "-B"}
	default:
		return "", false
	}
	o := parseGitOpts(target.rest, spec)
	branch := o.value(create...)
	if branch == "" || len(o.operands) > 0 || o.dashdash {
		return "", false
	}
	allowed := append(create, "-q", "--quiet", "--progress", "--no-progress", "--no-track")
	for opt := range o.set {
		if !slices.Contains(allowed, opt) {
			return "", false
		}
	}
	return branch, true
}

// namesBranched reports whether rev may name another commit after the line
// created and switched to the branches l.branched: it names one of them
// (`topic`, `heads/topic~1`, `main..topic`, case folded as on a
// case-insensitive file system), or a branch-relative or reflog revision
// (`@{-1}`, `@{u}`, `HEAD@{1}`).
func (l rewriteLine) namesBranched(rev string) bool {
	if strings.Contains(rev, "@{") {
		return true
	}
	for _, part := range strings.Split(strings.ReplaceAll(rev, "...", ".."), "..") {
		part = strings.TrimLeft(part, "^")
		if i := strings.IndexAny(part, "~^:@"); i >= 0 {
			part = part[:i]
		}
		part = strings.TrimPrefix(strings.TrimPrefix(part, "refs/"), "heads/")
		if slices.ContainsFunc(l.branched, func(b string) bool { return strings.EqualFold(b, part) }) {
			return true
		}
	}
	return false
}

var (
	commitSpec = gitOptSpec{
		long: []string{
			"all", "patch", "reuse-message", "reedit-message", "fixup", "squash", "reset-author", "short", "branch",
			"porcelain", "long", "null", "file", "author", "date", "message", "template", "signoff", "no-signoff",
			"trailer", "no-verify", "verify", "allow-empty", "allow-empty-message", "cleanup", "edit", "no-edit",
			"amend", "no-post-rewrite", "include", "only", "pathspec-from-file", "pathspec-file-nul",
			"untracked-files", "verbose", "quiet", "dry-run", "status", "no-status", "gpg-sign", "no-gpg-sign",
			"interactive",
		},
		longValue: map[string]bool{
			"reuse-message": true, "reedit-message": true, "fixup": true, "squash": true, "file": true, "author": true,
			"date": true, "message": true, "template": true, "trailer": true, "cleanup": true, "pathspec-from-file": true,
		},
		shortValue:    "cCFmt",
		shortOptional: "Su",
	}
	addSpec = gitOptSpec{
		long: []string{
			"dry-run", "verbose", "force", "interactive", "patch", "edit", "update", "all", "no-all",
			"ignore-removal", "no-ignore-removal", "intent-to-add", "refresh", "ignore-errors", "no-ignore-errors",
			"ignore-missing", "sparse", "chmod", "renormalize", "pathspec-from-file", "pathspec-file-nul",
			"no-warn-embedded-repo",
		},
		longValue: map[string]bool{"pathspec-from-file": true},
	}
)

// ranCommit records a git commit with options o. A dry run commits nothing.
// --amend replaces HEAD with a commit on HEAD's parent, so a revision then
// names a commit the model does not know, and --interactive edits the index
// as a patch; both count as an unmodelled change, as do options the shell
// may expand into others (gitOpts.unmodelled: `-{a,-amend}`). Any other
// commit is recorded (committed).
func (l *rewriteLine) ranCommit(o gitOpts) {
	switch {
	case o.unmodelled:
		l.changed = true
	case o.has("--dry-run"):
	case o.has("--amend", "--interactive"):
		l.changed = true
	default:
		l.committed = true
	}
}

// chmodAdd is a git add --chmod: it stages the mode it names for the tracked
// paths that pathspec, relative to dir, matches.
type chmodAdd struct {
	dir      string
	pathspec []string
}

// ranAdd records a git add sc with options o. A dry run stages nothing.
// --edit and --interactive stage an edited patch and --renormalize a
// converted content the working tree does not hold, so they count as an
// unmodelled change, as do options the shell may expand into others
// (gitOpts.unmodelled). Any other add copies working-tree content into the
// index (added), of ignored files too with --force (addedIgnored), and
// --chmod also stages a mode the working tree may not have for the paths its
// pathspec names (chmods), which a later rewrite may then write; it counts
// as an unmodelled change when those paths are not known
// (--pathspec-from-file, an unknown directory). Git applies --chmod to an
// explicit pathspec only: `git add -A --chmod=+x` and `git add -u
// --chmod=+x` stage the working tree's modes, as without --chmod.
func (l *rewriteLine) ranAdd(sc scannedCommand, o gitOpts) {
	switch {
	case o.unmodelled:
		l.changed = true
		return
	case o.has("-n", "--dry-run"):
		return
	case o.has("-e", "--edit", "-i", "--interactive", "--renormalize"):
		l.changed = true
		return
	}
	l.added = true
	l.addedIgnored = l.addedIgnored || o.has("-f", "--force")
	if !o.has("--chmod") {
		return
	}
	pathspec := append(slices.Clone(o.operands), o.paths...)
	if len(pathspec) == 0 && !o.has("--pathspec-from-file") {
		return
	}
	dir, ok := rewriteDir(sc, nil)
	if !ok || o.has("--pathspec-from-file") {
		l.changed = true
		return
	}
	l.chmods = append(l.chmods, chmodAdd{dir: dir, pathspec: pathspec})
}

// stashKeepingGit are the git subcommands that never change the stash list:
// they move HEAD or a branch, or change the index or the working tree.
var stashKeepingGit = map[string]bool{
	"add": true, "stage": true, "commit": true, "checkout": true, "switch": true, "branch": true, "restore": true,
	"reset": true, "rm": true, "mv": true, "merge": true, "rebase": true, "cherry-pick": true, "revert": true,
	"apply": true, "am": true, "clean": true, "read-tree": true, "checkout-index": true, "sparse-checkout": true,
	"fetch": true, "pull": true,
}

// keepsStash reports whether the git command target leaves the stash list
// alone (stashKeepingGit). A fetch or pull does, unless a refspec after the
// repository names where to store a ref (`git fetch . x:refs/stash`) or a
// --refmap maps one.
func keepsStash(target gitArgs) bool {
	if !stashKeepingGit[target.sub] {
		return false
	}
	if target.sub != "fetch" && target.sub != "pull" {
		return true
	}
	o := parseGitOpts(target.rest, pullSpec)
	refspecs := o.revisions()
	return !o.has("--refmap") && (len(refspecs) < 2 || !slices.ContainsFunc(refspecs[1:], func(s string) bool {
		return strings.Contains(s, ":")
	}))
}

// readsChanged reports whether the commands before rw on its line may have
// changed what it reads in a way the model cannot follow, so what it writes
// is known only in a repository that tracks no protected file. A stash entry
// changes only with the stash list; after a stash push, stash@{0} is either
// the entry the push saved, whose paths the push was judged to write, or,
// when it saved nothing, the entry it is now. A patch file changes with the
// working tree, which a stash push resets. After a commit, a revision that
// walks HEAD's history or reads a reflog names another commit
// (movesWithCommit), and after a branch is created at HEAD, a revision naming
// it or relative to the current branch (namesBranched).
func (l rewriteLine) readsChanged(rw worktreeRewrite) bool {
	switch rw.source {
	case fromStash:
		return l.stashChanged || l.stashPushed && rw.rev != latestStash
	case fromPatches:
		return l.changed || l.stashPushed
	}
	revs := rw.revisions()
	return l.changed || l.committed && slices.ContainsFunc(revs, movesWithCommit) ||
		len(l.branched) > 0 && slices.ContainsFunc(revs, l.namesBranched)
}

// localChanges is the comparison of what a commit records: the paths where
// the working tree differs from HEAD, and where the index differs from the
// working tree (a staged change the working tree has since undone).
var localChanges = worktreeRewrite{source: fromRev, rev: "HEAD", also: []worktreeRewrite{{source: fromIndex}}}

// precededBy returns how the commands before rw on its line change what it
// writes. preceded reports that they may change what it reads in a way the
// model cannot follow (readsChanged), or that a git add before it may start
// tracking an untracked protected path (an ignored one only after add
// --force), which a rewrite of tracked files then writes or removes (`git
// add -A && git reset --hard`). Otherwise also lists the paths whose index
// entry the commands before it change in a way the rewrite may write: the
// tracked paths a git add --chmod names (chmodded), and, for a rewrite from
// a revision, the paths a commit before it records (localChanges), where
// HEAD then differs from the HEAD git is asked about. A query that fails
// other than by a timeout (an unborn HEAD) counts as preceded.
func (ctx *EvalContext) precededBy(client gitrepo.Client, top, dir string, rw worktreeRewrite, line rewriteLine) (preceded bool, also []string, err error) {
	if line.readsChanged(rw) {
		return true, nil, nil
	}
	if line.added && rw.readsTracking() {
		opts := gitrepo.CleanOptions{Force: 1, Dirs: true, Ignored: line.addedIgnored}
		untracked, err := client.CleanPaths(ctx.Ctx, top, opts, nil)
		if errors.Is(err, gitrepo.ErrTimeout) {
			return false, nil, err
		}
		if err != nil || ctx.firstRemovedProtected(top, untracked) != "" {
			return true, nil, nil
		}
		also, err = ctx.chmodded(client, line.chmods)
		if errors.Is(err, gitrepo.ErrTimeout) {
			return false, nil, err
		}
		if err != nil {
			return true, nil, nil
		}
	}
	if !line.committed || !rw.readsRevisions() {
		return false, also, nil
	}
	committed, err := rewrittenPaths(ctx.Ctx, client, dir, localChanges)
	if errors.Is(err, gitrepo.ErrTimeout) {
		return false, nil, err
	}
	return err != nil, append(also, committed...), nil
}

// chmodded returns the tracked paths, relative to the toplevel, whose mode
// the git add --chmod commands chmods stage.
func (ctx *EvalContext) chmodded(client gitrepo.Client, chmods []chmodAdd) ([]string, error) {
	var paths []string
	for _, c := range chmods {
		tracked, err := client.TrackedPaths(ctx.Ctx, c.dir, c.pathspec)
		if err != nil {
			return nil, err
		}
		paths = append(paths, tracked...)
	}
	return paths, nil
}

// readsTracking reports whether what rw writes depends on which paths the
// index tracks: any rewrite but a stash pop, a patch, a clean alone or a
// sparse-checkout pattern change.
func (rw worktreeRewrite) readsTracking() bool {
	switch rw.source {
	case fromStash, fromPatches, sparseRules, 0:
		return false
	}
	return true
}

// readsRevisions reports whether rw copies from a revision, which a commit
// before it may move.
func (rw worktreeRewrite) readsRevisions() bool {
	switch rw.source {
	case fromRev, betweenRevs, mergeBase, fromCommits:
		return true
	}
	return false
}

// revisions returns the revisions rw and the comparisons it also writes
// name.
func (rw worktreeRewrite) revisions() []string {
	revs := slices.Concat([]string{rw.rev, rw.base}, rw.revs)
	for _, more := range rw.also {
		revs = append(revs, more.revisions()...)
	}
	return slices.DeleteFunc(revs, func(r string) bool { return r == "" })
}

// movesWithCommit reports whether rev may name another commit once a commit
// has been made: it walks history from a ref (`HEAD~1`, `main^2`), searches
// commit messages or names a path in a tree (`:/fix`, `HEAD:x`), or reads a
// reflog (`@{1}`, `HEAD@{yesterday}`). HEAD and the current branch name the
// new commit, whose tree differs from the old one only in what the commit
// recorded, and `@{-1}`, `@{u}`, `@{upstream}` and `@{push}` name other
// branches.
func movesWithCommit(rev string) bool {
	if strings.ContainsAny(rev, "~^:") {
		return true
	}
	for rest := rev; ; {
		_, after, ok := strings.Cut(rest, "@{")
		if !ok {
			return false
		}
		selector, _, _ := strings.Cut(after, "}")
		switch strings.ToLower(selector) {
		case "u", "upstream", "push":
		default:
			if !strings.HasPrefix(selector, "-") {
				return true
			}
		}
		rest = after
	}
}
