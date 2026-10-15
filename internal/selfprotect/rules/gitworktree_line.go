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

// inertCommand reports whether sc, a command other than git, cannot change
// what a later rewrite reads: a directory change, or a read-only command
// (cmdscan.IsSafeReadCommand: echo, printf, true, test, cat, ...) that sets
// no variable and writes no file through a redirect.
func inertCommand(sc scannedCommand) bool {
	return isLocationChange(sc.Name) || len(sc.WriteRedirects) == 0 && cmdscan.IsSafeReadCommand(sc.Command)
}

// ranGit records what the git command sc runs (args, split as target) may
// change for a later rewrite on its line; rewrites reports that it is itself
// the rewrite rw, already judged. A read-only git changes nothing. A git run
// directly (directGit) is modelled: a stash push saves stash@{0}, a commit
// and an add are recorded for the rewrites after them (ranCommit, ranAdd),
// and any other git may change the index, HEAD, refs or the working tree,
// but the stash only when it is not one of stashKeepingGit (keepsStash). A
// stash pop, apply or branch also drops or shifts stash entries. A git behind
// a wrapper, a variable, a redirect, an expansion or a global option may do
// anything.
func (l *rewriteLine) ranGit(sc scannedCommand, args []string, target gitArgs, rw worktreeRewrite, rewrites bool) {
	switch {
	case !rewrites && gitReadOnly(target):
	case !directGit(sc, args, target):
		l.changeAll()
	case rewrites && rw.source == fromStash:
		l.changeAll()
	case rewrites && target.sub == "stash":
		l.stashPushed = true
	case !rewrites && target.sub == "commit":
		l.ranCommit(parseGitOpts(target.rest, commitSpec))
	case !rewrites && (target.sub == "add" || target.sub == "stage"):
		l.ranAdd(parseGitOpts(target.rest, addSpec))
	default:
		l.changed = true
		l.stashChanged = l.stashChanged || !keepsStash(target)
	}
}

// directGit reports whether sc runs git itself on the repository of its
// directory, with nothing that changes what git does: no wrapper, variable,
// write redirect, expansion or global option before the subcommand.
func directGit(sc scannedCommand, args []string, target gitArgs) bool {
	return path.Base(sc.Name) == "git" && len(args) > 0 && args[0] == target.sub && len(sc.Assigns) == 0 &&
		!sc.AssignsDynamic && len(sc.WriteRedirects) == 0 && !sc.HasExpansion
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
		shortValue: "cCFmt",
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
// as a patch; both count as an unmodelled change. Any other commit is
// recorded (committed).
func (l *rewriteLine) ranCommit(o gitOpts) {
	switch {
	case o.has("--dry-run"):
	case o.has("--amend", "--interactive"):
		l.changed = true
	default:
		l.committed = true
	}
}

// ranAdd records a git add with options o. A dry run stages nothing. --edit
// and --interactive stage an edited patch, --chmod a mode and --renormalize
// a converted content the working tree does not hold, so they count as an
// unmodelled change. Any other add only copies working-tree content into the
// index (added), of ignored files too with --force (addedIgnored).
func (l *rewriteLine) ranAdd(o gitOpts) {
	switch {
	case o.has("-n", "--dry-run"):
	case o.has("-e", "--edit", "-i", "--interactive", "--chmod", "--renormalize"):
		l.changed = true
	default:
		l.added = true
		l.addedIgnored = l.addedIgnored || o.has("-f", "--force")
	}
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
// (movesWithCommit).
func (l rewriteLine) readsChanged(rw worktreeRewrite) bool {
	switch rw.source {
	case fromStash:
		return l.stashChanged || l.stashPushed && rw.rev != latestStash
	case fromPatches:
		return l.changed || l.stashPushed
	}
	return l.changed || l.committed && slices.ContainsFunc(rw.revisions(), movesWithCommit)
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
// add -A && git reset --hard`). Otherwise committed lists the paths a commit before it
// records (localChanges): HEAD then differs from the HEAD git is asked about
// in those paths, so a rewrite from a revision may write them too. A query
// that fails other than by a timeout (an unborn HEAD) counts as preceded.
func (ctx *EvalContext) precededBy(client gitrepo.Client, top, dir string, rw worktreeRewrite, line rewriteLine) (preceded bool, committed []string, err error) {
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
	}
	if !line.committed || !rw.readsRevisions() {
		return false, nil, nil
	}
	committed, err = rewrittenPaths(ctx.Ctx, client, dir, localChanges)
	if errors.Is(err, gitrepo.ErrTimeout) {
		return false, nil, err
	}
	return err != nil, committed, nil
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
