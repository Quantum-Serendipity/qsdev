package rules

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/gitrepo"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// This file holds the worktree-rewrite check of SP-007 (U18-02c). A git
// command such as `git checkout evil -- .`, `git reset --hard evil`, `git
// merge evil` or `git apply x.patch` writes files into the working tree
// without naming them, so a protected
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
	// betweenRevs moves the work tree from HEAD (or base) to a revision,
	// carrying local changes along: only the paths that differ between the
	// two trees are written.
	betweenRevs
	// fromStash applies a stash entry.
	fromStash
	// mergeBase merges revisions into HEAD (or base), or rebases onto them:
	// the paths each changed since its merge base with HEAD are written
	// (`git diff HEAD...rev`; the two-dot form would also name what only
	// HEAD's side changed).
	mergeBase
	// fromCommits replays commits (cherry-pick, revert): the paths they
	// change are written.
	fromCommits
	// fromPatches applies patch files (apply, am): the paths they name are
	// written, tracked or not.
	fromPatches
	// sparseRules changes the sparse-checkout patterns (sparse-checkout set,
	// add, init, reapply): any tracked path may leave the working tree, and
	// which ones cannot be told in advance.
	sparseRules
	// removesTracked deletes the tracked paths a pathspec matches (git rm):
	// the files go from the working tree as well as from the index.
	removesTracked
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
	// base is where betweenRevs and mergeBase start from; "" is HEAD.
	base string
	// revs are the revisions a mergeBase merges or the commits fromCommits
	// replays.
	revs []string
	// patches are the files fromPatches applies, with the command's -p count
	// (strip) and --directory prefix.
	patches          []string
	strip, directory string
	// paths limits the rewrite (a pathspec relative to the command's
	// directory); none means the whole tree.
	paths []string
	// orPaths, when set, is the other reading of an ambiguous `git checkout
	// <operand>...` without "--": when the first operand does not name a
	// revision, git restores all the operands as paths from the index.
	orPaths []string
	// also are further comparisons the command writes: the index after a
	// stash that keeps it (`git stash -k`), the branch a rebase switches to
	// before it replays.
	also []worktreeRewrite
	// unchecked, when set, is why the command writes files this check
	// cannot predict (an output prefix), so it is denied outright.
	unchecked string
	// remote is a pull's repository given as a path (isPathRemote): the
	// pull is the merge of revs when it names the repository itself, and a
	// fetch from elsewhere otherwise.
	remote string
	// clean, when set, is the git clean the command also runs over paths
	// (`git clean`, `git stash -u`): it removes the untracked files git
	// clean --dry-run with these options lists. An earlier index-only
	// command (`git rm -r --cached .`, `git read-tree --empty`) can make a
	// protected file one of them.
	clean *gitrepo.CleanOptions
}

// gitOptSpec describes a git subcommand's options for parseGitOpts.
type gitOptSpec struct {
	long       []string        // known long options, without --
	longValue  map[string]bool // long options taking the next word as value
	shortValue string          // short options taking a value
	// shortOptional are the short options taking an optional value, which
	// can only be attached: the rest of the cluster (`-Skey`, `-uno`), never
	// the next word (`-S -m x`).
	shortOptional string
}

// gitOpts is a git subcommand's arguments split into options and operands.
type gitOpts struct {
	set      map[string]string // canonical option ("-f", "--force") -> value
	operands []string          // before "--"
	paths    []string          // after "--"
	dashdash bool
	// valueArgs are the indexes in the parsed args of the words read as an
	// option's value (`-m <msg>`), not as an option or an operand.
	valueArgs []int
	// attachedArgs are the indexes in the parsed args of the words holding a
	// value option and its value (`-m<msg>`, `--message=<msg>`).
	attachedArgs []int
	// markOnlyArgs are the attachedArgs of a short value option whose value
	// is expansionMark alone (`-m<mark>`, from `-m"$MSG"`): the expansion may
	// be empty, and the option then takes the next word as its value.
	markOnlyArgs []int
	// unmodelled reports an option word whose option names hold a character
	// git's options never do (`-{a,-amend}`, `--{amend,x}`): the shell may
	// expand it, as a brace or glob pattern, into other options.
	unmodelled bool
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

// revisions returns the operands of a subcommand that takes no pathspec
// (merge, pull, rebase, cherry-pick, revert, apply, am): its "--" only ends
// the options, so the words after it are operands too (`git merge -- evil`).
func (o gitOpts) revisions() []string {
	return append(slices.Clone(o.operands), o.paths...)
}

// parseGitOpts splits args per spec. A long option is canonicalised to the
// known option it unambiguously abbreviates, as git's option parser does
// (`--har` is --hard); a short cluster (`-qf`, `-bname`) is split into its
// options, the rest of the cluster after a value option (or an option with an
// optional value, spec.shortOptional) being its value. An
// option name with a character outside [A-Za-z0-9] (and -, in a long name)
// sets unmodelled.
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
			raw, val, hasVal := strings.Cut(a[2:], "=")
			o.unmodelled = o.unmodelled || strings.ContainsFunc(raw, func(r rune) bool { return r != '-' && !optionChar(r) })
			name := abbreviatedOption(raw, spec.long)
			switch {
			case hasVal && spec.longValue[name]:
				o.attachedArgs = append(o.attachedArgs, i)
			case !hasVal && spec.longValue[name] && i+1 < len(args):
				i++
				val = args[i]
				o.valueArgs = append(o.valueArgs, i)
			}
			o.set["--"+name] = val
		case len(a) > 1 && a[0] == '-':
			for k := 1; k < len(a); k++ {
				opt := "-" + a[k:k+1]
				o.unmodelled = o.unmodelled || !optionChar(rune(a[k]))
				if strings.IndexByte(spec.shortOptional, a[k]) >= 0 {
					o.set[opt] = a[k+1:]
					if k+1 < len(a) {
						o.attachedArgs = append(o.attachedArgs, i)
					}
					break
				}
				if strings.IndexByte(spec.shortValue, a[k]) < 0 {
					o.set[opt] = ""
					continue
				}
				val := a[k+1:]
				switch {
				case val != "":
					o.attachedArgs = append(o.attachedArgs, i)
					if val == expansionMark {
						o.markOnlyArgs = append(o.markOnlyArgs, i)
					}
				case i+1 < len(args):
					i++
					val = args[i]
					o.valueArgs = append(o.valueArgs, i)
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

// optionChar reports whether r may appear in a git option name: an ASCII
// letter or digit.
func optionChar(r rune) bool {
	return r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9'
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
	rmSpec = gitOptSpec{
		long: []string{
			"force", "dry-run", "cached", "ignore-unmatch", "sparse", "quiet", "pathspec-from-file",
			"pathspec-file-nul",
		},
		longValue: map[string]bool{"pathspec-from-file": true},
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
	mergeSpec = gitOptSpec{
		long: slices.Concat(mergeLong, sequencerControls),
		longValue: map[string]bool{
			"cleanup": true, "strategy": true, "strategy-option": true, "message": true, "file": true, "into-name": true,
		},
		shortValue: "sXmF",
	}
	// pullSpec adds fetch's options to merge's.
	pullSpec = gitOptSpec{
		long: slices.Concat(mergeLong, []string{
			"rebase", "no-rebase", "all", "append", "atomic", "depth", "deepen", "shallow-since", "shallow-exclude",
			"unshallow", "update-shallow", "force", "keep", "jobs", "prune", "no-prune", "prune-tags",
			"recurse-submodules", "no-recurse-submodules", "tags", "no-tags", "upload-pack", "refmap",
			"server-option", "set-upstream", "ipv4", "ipv6", "negotiation-tip", "show-forced-updates",
			"no-show-forced-updates", "dry-run",
		}),
		longValue: map[string]bool{
			"cleanup": true, "strategy": true, "strategy-option": true, "depth": true, "deepen": true,
			"shallow-since": true, "shallow-exclude": true, "jobs": true, "upload-pack": true, "refmap": true,
			"server-option": true, "negotiation-tip": true,
		},
		shortValue: "sXoj",
	}
	rebaseSpec = gitOptSpec{
		long: slices.Concat([]string{
			"onto", "keep-base", "no-keep-base", "apply", "merge", "empty", "keep-empty", "no-keep-empty",
			"reapply-cherry-picks", "no-reapply-cherry-picks", "allow-empty-message", "strategy", "strategy-option",
			"rerere-autoupdate", "no-rerere-autoupdate", "gpg-sign", "no-gpg-sign", "quiet", "verbose", "stat",
			"no-stat", "verify", "no-verify", "force-rebase", "no-ff", "fork-point", "no-fork-point",
			"ignore-whitespace", "whitespace", "committer-date-is-author-date", "ignore-date", "reset-author-date",
			"signoff", "interactive", "rebase-merges", "no-rebase-merges", "update-refs", "no-update-refs", "root",
			"autosquash", "no-autosquash", "autostash", "no-autostash", "exec", "reschedule-failed-exec",
			"no-reschedule-failed-exec", "edit-todo", "show-current-patch",
		}, sequencerControls),
		longValue:  map[string]bool{"onto": true, "strategy": true, "strategy-option": true, "exec": true, "whitespace": true, "empty": true},
		shortValue: "sXxC",
	}
	// pickSpec is cherry-pick's and revert's.
	pickSpec = gitOptSpec{
		long: slices.Concat([]string{
			"edit", "no-edit", "mainline", "no-commit", "signoff", "no-signoff", "ff", "allow-empty",
			"allow-empty-message", "keep-redundant-commits", "empty", "strategy", "strategy-option", "gpg-sign",
			"no-gpg-sign", "rerere-autoupdate", "no-rerere-autoupdate", "cleanup", "reference", "no-reference",
		}, sequencerControls),
		longValue:  map[string]bool{"mainline": true, "strategy": true, "strategy-option": true, "cleanup": true},
		shortValue: "mX",
	}
	applySpec = gitOptSpec{
		long: []string{
			"stat", "numstat", "summary", "check", "index", "intent-to-add", "cached", "3way", "ours", "theirs",
			"union", "build-fake-ancestor", "apply", "no-add", "exclude", "include", "reject", "reverse",
			"unidiff-zero", "recount", "allow-empty", "inaccurate-eof", "directory", "unsafe-paths", "verbose",
			"quiet", "whitespace", "allow-overlap", "ignore-space-change", "ignore-whitespace",
		},
		longValue: map[string]bool{
			"build-fake-ancestor": true, "exclude": true, "include": true, "directory": true, "whitespace": true,
		},
		shortValue: "pC",
	}
	amSpec = gitOptSpec{
		long: slices.Concat([]string{
			"signoff", "keep", "keep-non-patch", "keep-cr", "no-keep-cr", "scissors", "no-scissors", "quoted-cr",
			"message-id", "no-message-id", "quiet", "utf8", "no-utf8", "3way", "no-3way", "rerere-autoupdate",
			"no-rerere-autoupdate", "ignore-space-change", "ignore-whitespace", "whitespace", "directory", "exclude",
			"include", "reject", "patch-format", "interactive", "commit-date-is-author-date", "ignore-date",
			"gpg-sign", "no-gpg-sign", "empty", "resolved", "retry", "show-current-patch", "allow-empty", "no-verify",
		}, sequencerControls),
		longValue: map[string]bool{
			"whitespace": true, "directory": true, "exclude": true, "include": true, "patch-format": true, "quoted-cr": true,
		},
		shortValue: "pC",
	}
)

// mergeLong are the long options of git merge, which git pull shares.
var mergeLong = []string{
	"commit", "no-commit", "edit", "no-edit", "cleanup", "ff", "no-ff", "ff-only", "gpg-sign", "no-gpg-sign",
	"log", "no-log", "signoff", "no-signoff", "stat", "no-stat", "summary", "no-summary", "squash", "no-squash",
	"verify", "no-verify", "strategy", "strategy-option", "verify-signatures", "no-verify-signatures", "quiet",
	"verbose", "progress", "no-progress", "autostash", "no-autostash", "allow-unrelated-histories",
	"rerere-autoupdate", "no-rerere-autoupdate", "overwrite-ignore", "no-overwrite-ignore", "message", "file",
	"into-name",
}

// sequencerControls are the long options that continue, skip or end a merge,
// rebase, cherry-pick, revert or am in progress instead of starting one. The
// operation was judged when it started; --abort restores what it replaced.
var sequencerControls = []string{"continue", "skip", "abort", "quit"}

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
	case "merge":
		return parseMerge(parseGitOpts(rest, mergeSpec))
	case "pull":
		return parsePull(parseGitOpts(rest, pullSpec))
	case "rebase":
		return parseRebase(parseGitOpts(rest, rebaseSpec))
	case "cherry-pick", "revert":
		return parsePick(parseGitOpts(rest, pickSpec))
	case "apply":
		o := parseGitOpts(rest, applySpec)
		if o.has("--cached") || patchListing(o) {
			return worktreeRewrite{}, false // --cached patches only the index
		}
		return parsePatches(o)
	case "am":
		o := parseGitOpts(rest, amSpec)
		if o.controls("-r", "--resolved", "--retry", "--show-current-patch") {
			return worktreeRewrite{}, false
		}
		return parsePatches(o)
	case "clean":
		return parseClean(rest)
	case "sparse-checkout":
		return parseSparseCheckout(rest)
	case "rm":
		return parseRm(parseGitOpts(rest, rmSpec))
	}
	return worktreeRewrite{}, false
}

// parseRm returns what `git rm` deletes from the working tree: every tracked
// path its pathspec matches (`git rm -r .` takes .claude/ with it). --cached
// only untracks the paths and a dry run (-n) removes nothing; a pathspec read
// from a file may match any tracked path, so it is judged as the whole
// index (":/"). Without -r git refuses a directory it would have to recurse
// into, so judging the pathspec as if -r were given only adds paths.
func parseRm(o gitOpts) (worktreeRewrite, bool) {
	if o.has("--cached", "-n", "--dry-run") {
		return worktreeRewrite{}, false
	}
	if o.has("--pathspec-from-file") {
		return worktreeRewrite{source: removesTracked, paths: []string{":/"}}, true
	}
	paths := append(slices.Clone(o.operands), o.paths...)
	if len(paths) == 0 {
		return worktreeRewrite{}, false // git rm requires a pathspec
	}
	return worktreeRewrite{source: removesTracked, paths: paths}, true
}

// cleanLong are git clean's long options.
var cleanLong = []string{"dry-run", "no-dry-run", "force", "no-force", "interactive", "quiet", "no-quiet", "exclude"}

// parseClean returns what `git clean` removes: the untracked files its
// options select (-d, -x, -X, -f, -e) under its pathspec. A dry run (-n,
// unless a later --no-dry-run) removes nothing; an interactive one removes
// what the listing names, at most.
func parseClean(rest []string) (worktreeRewrite, bool) {
	s := scanClean(rest)
	if s.dryRun {
		return worktreeRewrite{}, false
	}
	// Without -f git refuses unless clean.requireForce is off; judge it as
	// the forced clean it may be.
	s.opts.Force = max(s.opts.Force, 1)
	return worktreeRewrite{clean: &s.opts, paths: s.paths}, true
}

// cleanScan is git clean's arguments read: the options, the pathspec, whether
// it is a dry run, and the positions in its arguments of the words that only
// carry -e/--exclude patterns (the option word and a separate value).
type cleanScan struct {
	opts     gitrepo.CleanOptions
	paths    []string
	dryRun   bool
	patterns []int
}

// scanClean reads git clean's arguments rest. -f counts (twice also removes
// nested repositories) and -e repeats, so the options are read here rather
// than by parseGitOpts.
func scanClean(rest []string) cleanScan {
	var s cleanScan
	opts := &s.opts
	for i := 0; i < len(rest); i++ {
		a := rest[i]
		switch {
		case a == "--":
			s.paths = append(s.paths, rest[i+1:]...)
			i = len(rest)
		case strings.HasPrefix(a, "--"):
			name, val, hasVal := strings.Cut(a[2:], "=")
			switch abbreviatedOption(name, cleanLong) {
			case "dry-run":
				s.dryRun = true
			case "no-dry-run":
				s.dryRun = false
			case "force":
				opts.Force++
			case "no-force":
				opts.Force = 0
			case "exclude":
				s.patterns = append(s.patterns, i)
				if !hasVal && i+1 < len(rest) {
					i++
					val = rest[i]
					s.patterns = append(s.patterns, i)
				}
				opts.Excludes = append(opts.Excludes, val)
			}
		case len(a) > 1 && a[0] == '-':
			for k := 1; k < len(a); k++ {
				switch a[k] {
				case 'n':
					s.dryRun = true
				case 'f':
					opts.Force++
				case 'd':
					opts.Dirs = true
				case 'x':
					opts.Ignored = true
				case 'X':
					opts.OnlyIgnored = true
				case 'e':
					s.patterns = append(s.patterns, i)
					val := a[k+1:]
					if val == "" && i+1 < len(rest) {
						i++
						val = rest[i]
						s.patterns = append(s.patterns, i)
					}
					opts.Excludes = append(opts.Excludes, val)
					k = len(a)
				}
			}
		default:
			s.paths = append(s.paths, a)
		}
	}
	return s
}

// cleanPatternArgs returns the positions in sc.Args of the words that only
// carry the -e/--exclude patterns of a git clean (scanClean), or nil when sc
// runs no git clean. An exclude selects what the clean keeps; SP-007 judges
// it by git's own dry run with the same excludes, so it names no target.
// That dry run sees only the words on the line, so under a wrapper that
// appends arguments from its input (cmdscan.ArgsFromInput: `xargs git clean
// -fdx -e .claude -e < list`, where list may hold `!.claude`) the excludes
// stay operands, as any other word naming a protected path does.
func cleanPatternArgs(sc scannedCommand) []int {
	words := append([]string{sc.Name}, sc.Args...)
	args, ok := gitInvocation(words)
	if !ok || cmdscan.ArgsFromInput(words) {
		return nil
	}
	g := splitGitArgs(args)
	if g.sub != "clean" {
		return nil
	}
	patterns := scanClean(g.rest).patterns
	offset := len(sc.Args) - len(g.rest) // g.rest is a suffix of sc.Args
	for i := range patterns {
		patterns[i] += offset
	}
	return patterns
}

// parseSparseCheckout returns what `git sparse-checkout <sub>` writes. set,
// init and reapply apply a new or re-read pattern set, which may drop any
// tracked file from the working tree; add does too unless it only adds
// patterns that include more (none starting with '!', none read from
// --stdin). list and check-rules only read, and disable only restores the
// full tree. Any other subcommand is judged as a pattern change.
func parseSparseCheckout(rest []string) (worktreeRewrite, bool) {
	if len(rest) == 0 {
		return worktreeRewrite{}, false
	}
	switch rest[0] {
	case "list", "check-rules", "disable":
		return worktreeRewrite{}, false
	case "add":
		if !slices.ContainsFunc(rest[1:], func(a string) bool {
			return a == "--stdin" || strings.HasPrefix(a, "!") || strings.HasPrefix(a, `\!`)
		}) {
			return worktreeRewrite{}, false
		}
	}
	return worktreeRewrite{source: sparseRules}, true
}

// controls reports whether the options continue, skip or end an operation in
// progress (sequencerControls, or one of the subcommand's own extra).
func (o gitOpts) controls(extra ...string) bool {
	for _, c := range sequencerControls {
		if o.has("--" + c) {
			return true
		}
	}
	return o.has(extra...)
}

// Why merge-family rewrites are denied outright.
const (
	interactiveUnchecked = "replays commits from a todo list that can be edited (--interactive), so the files it writes cannot be checked"
	rootUnchecked        = "replays every commit from the root (--root), so the files it writes cannot be checked"
	execUnchecked        = "runs commands between the commits it replays (--exec), which cannot be checked"
	stdinCommitUnchecked = "reads the commits to replay from standard input (--stdin), so the files it writes cannot be checked"
	revOptionUnchecked   = "selects the commits to replay by revision option, so the files it writes cannot be checked"
	stdinPatchUnchecked  = "reads the patch from standard input, so the files it writes cannot be checked; save the patch to a file and apply that file"
)

// parseMerge returns what `git merge [<rev>...]` writes: what each revision
// changed since its merge base with HEAD (an octopus merges several). With
// no revision it merges the upstream.
func parseMerge(o gitOpts) (worktreeRewrite, bool) {
	if o.controls() {
		return worktreeRewrite{}, false
	}
	revs := o.revisions()
	previousBranch(revs)
	return worktreeRewrite{source: mergeBase, revs: orDefault(revs, "@{upstream}")}, true
}

// parsePull returns what `git pull <repo> <ref>...` writes when it pulls from
// the repository itself: from remote "." it merges or rebases onto the refs,
// which writes the same paths. A remote given as a filesystem path (`./`,
// `/abs`, a file:// URL, or a word the scan cannot render) is judged the same
// way, in askGit, when it names this repository (remote). A pull from
// another remote fetches first, so what it writes is not known before it
// runs; it is left to the documented residual (`git fetch` and then a
// judged `git merge FETCH_HEAD`).
func parsePull(o gitOpts) (worktreeRewrite, bool) {
	ops := o.revisions()
	if len(ops) == 0 || ops[0] != "." && !isPathRemote(ops[0]) {
		return worktreeRewrite{}, false
	}
	if v := o.value("--rebase"); v == "interactive" || v == "i" {
		return worktreeRewrite{unchecked: interactiveUnchecked}, true
	}
	var refs []string
	for _, spec := range ops[1:] {
		// A refspec "+src:dst" merges src.
		if src, _, _ := strings.Cut(strings.TrimPrefix(spec, "+"), ":"); src != "" {
			refs = append(refs, src)
		}
	}
	rw := worktreeRewrite{source: mergeBase, revs: orDefault(refs, "@{upstream}")}
	if ops[0] != "." {
		rw.remote = ops[0]
	}
	return rw, true
}

// isPathRemote reports whether a pull's repository operand may be a
// filesystem path, which can name the repository itself: an absolute,
// dot-relative or home path, a file:// URL, or a word the scan rendered empty
// (an expansion, which checkRewrite denies). A remote's name or a URL of
// another scheme is not one.
func isPathRemote(remote string) bool {
	switch {
	case remote == "", remote == "..", strings.HasPrefix(remote, "file://"):
		return true
	}
	return slices.ContainsFunc([]string{"/", "./", "../", "~"}, func(p string) bool {
		return strings.HasPrefix(filepath.ToSlash(remote), p)
	}) || isRooted(remote)
}

// remoteIsRepo reports whether a pull's path remote names the repository
// whose top level is top, run from dir: the top level itself or its .git
// directory.
func remoteIsRepo(remote, dir, top string) bool {
	p := expandTilde(strings.TrimPrefix(remote, "file://"))
	if !isRooted(p) {
		p = filepath.Join(dir, p)
	}
	p = filepath.Clean(p)
	return p == top || p == filepath.Join(top, ".git")
}

// parseRebase returns what `git rebase [--onto <newbase>] [<upstream>
// [<branch>]]` writes. Replaying HEAD's commits onto upstream writes what
// upstream changed since their merge base; onto a newbase, what differs
// between HEAD and the newbase. A branch operand is switched to first, so
// the paths between HEAD and it are written too, and the replay starts from
// it.
func parseRebase(o gitOpts) (worktreeRewrite, bool) {
	switch {
	case o.controls("--edit-todo", "--show-current-patch"):
		return worktreeRewrite{}, false
	case o.has("-i", "--interactive"):
		return worktreeRewrite{unchecked: interactiveUnchecked}, true
	case o.has("--root"):
		return worktreeRewrite{unchecked: rootUnchecked}, true
	case o.has("-x", "--exec"):
		return worktreeRewrite{unchecked: execUnchecked}, true
	}
	revs := o.revisions()
	previousBranch(revs)
	rw := worktreeRewrite{source: mergeBase, revs: []string{firstOr(revs, "@{upstream}")}}
	if o.has("--onto") {
		rw = worktreeRewrite{source: betweenRevs, rev: o.value("--onto")}
	}
	if len(revs) > 1 {
		rw.base = revs[1]
		rw.also = []worktreeRewrite{{source: betweenRevs, rev: rw.base}}
	}
	return rw, true
}

// parsePick returns what `git cherry-pick|revert <commit>...` writes: the
// paths the commits change.
func parsePick(o gitOpts) (worktreeRewrite, bool) {
	switch {
	case o.controls():
		return worktreeRewrite{}, false
	case o.has("--stdin"):
		return worktreeRewrite{unchecked: stdinCommitUnchecked}, true
	case unknownLongOption(o, pickSpec):
		// A rev-list option (--all, --branches=) selects the commits.
		return worktreeRewrite{unchecked: revOptionUnchecked}, true
	}
	revs := o.revisions()
	if len(revs) == 0 {
		return worktreeRewrite{}, false // git refuses to pick nothing
	}
	previousBranch(revs)
	return worktreeRewrite{source: fromCommits, revs: revs}, true
}

// parsePatches returns what `git apply|am <patch>...` writes: the paths the
// patch files name. With no file, or "-", the patch comes from standard
// input and cannot be read in advance.
func parsePatches(o gitOpts) (worktreeRewrite, bool) {
	files := o.revisions()
	if len(files) == 0 || slices.Contains(files, "-") {
		return worktreeRewrite{source: fromPatches, unchecked: stdinPatchUnchecked}, true
	}
	return worktreeRewrite{source: fromPatches, patches: files, strip: o.value("-p"), directory: o.value("--directory")}, true
}

// patchListing reports whether git apply's options only list or check the
// patch: those modes apply nothing unless --apply asks for both.
func patchListing(o gitOpts) bool {
	return o.has("--check", "--stat", "--numstat", "--summary") && !o.has("--apply")
}

// unknownLongOption reports whether o holds a long option spec does not
// know.
func unknownLongOption(o gitOpts, spec gitOptSpec) bool {
	for opt := range o.set {
		if name, ok := strings.CutPrefix(opt, "--"); ok && !slices.Contains(spec.long, name) {
			return true
		}
	}
	return false
}

// orDefault returns s, or def alone when s is empty.
func orDefault(s []string, def string) []string {
	if len(s) == 0 {
		return []string{def}
	}
	return s
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
		// to it), or, with --force, is reset to HEAD (`git checkout -f -b
		// x` discards local changes as `git checkout -f` does).
		if len(o.operands) == 0 {
			if force {
				return worktreeRewrite{source: fromRev, rev: "HEAD"}, true
			}
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
		return worktreeRewrite{source: fromStash, rev: firstOr(o.operands, latestStash)}, true
	case "branch":
		// stash branch <name> [<stash>]
		ref := latestStash
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
// written too (also). withPaths reports whether the operands are a
// pathspec (push) rather than a message (save).
func parseStashPush(o gitOpts, withPaths bool) (worktreeRewrite, bool) {
	rw := worktreeRewrite{source: fromRev, rev: "HEAD"}
	if withPaths && !o.has("--pathspec-from-file") {
		rw.paths = append(slices.Clone(o.operands), o.paths...)
	}
	if o.has("-k", "--keep-index", "-S", "--staged", "-p", "--patch") && !o.has("--no-keep-index") {
		rw.also = []worktreeRewrite{{source: fromIndex, paths: rw.paths}}
	}
	// --include-untracked and --all then remove the untracked (and, with
	// --all, ignored) files they saved, as `git clean -f -d [-x]` does.
	if all := o.has("-a", "--all"); all || o.has("-u", "--include-untracked") {
		rw.clean = &gitrepo.CleanOptions{Force: 1, Dirs: true, Ignored: all}
	}
	return rw, true
}

func firstOr(s []string, def string) string {
	if len(s) > 0 {
		return s[0]
	}
	return def
}

// gitReadOnly reports whether the git command only reads the repository, so
// it may precede a rewrite on its line without changing what the rewrite
// copies: a read-only subcommand form (cmdscan.GitReadOnly, so `git branch`
// but not `git branch newb`), or git apply checking or listing a patch (`git
// apply --check fix.patch && git apply fix.patch`).
func gitReadOnly(target gitArgs) bool {
	return cmdscan.GitReadOnly(target.sub, target.rest) || target.sub == "apply" && patchListing(parseGitOpts(target.rest, applySpec))
}

// isLocationChange reports whether a scanned command only changes the
// working directory (PowerShell's location cmdlets are renamed to these).
func isLocationChange(name string) bool {
	return name == "cd" || name == "pushd" || name == "popd"
}

// isRepoPinVar reports whether the variable chooses the repository, index or
// work tree git uses, so a query run from the command's directory would ask
// another repository: a location pin or a write-target pin.
func isRepoPinVar(name string) bool {
	return isLocationPinVar(name) || isWriteTargetPinVar(name)
}

// isLocationPinVar reports whether the variable only names where git looks
// (`GIT_DIR=<dir>` as `--git-dir=<dir>`), so a read git runs under it writes
// nothing that `git --git-dir=<dir> --work-tree=<dir>` would not also write.
func isLocationPinVar(name string) bool {
	switch name {
	case "GIT_DIR", "GIT_WORK_TREE", "GIT_COMMON_DIR":
		return true
	}
	return false
}

// isWriteTargetPinVar reports whether the variable names a file or directory
// git writes even for a read: status, diff and describe --dirty refresh and
// rewrite the index GIT_INDEX_FILE names, and git stores objects (cached
// textconv output) in GIT_OBJECT_DIRECTORY.
func isWriteTargetPinVar(name string) bool {
	return name == "GIT_INDEX_FILE" || name == "GIT_OBJECT_DIRECTORY"
}

// repoPinnedRead reports whether sc runs a read-only git command
// (cmdscan.IsSafeReadCommand: a read-only subcommand form, cmdscan.GitReadOnly,
// with no option that writes a file or runs a program) in a repository chosen
// only by repo-pin variables (repoPinnedGit): `GIT_DIR=<dir> git branch` and
// `env GIT_DIR=<dir> git status` read as `git --git-dir=<dir> branch` does.
// A write-target pin's value then names no protected path, and a prefix
// assignment changes nothing a later command runs.
func repoPinnedRead(sc scannedCommand) bool {
	args, pins := repoPinnedGit(sc)
	return len(pins) > 0 && cmdscan.IsSafeReadCommand(cmdscan.Command{Name: "git", Args: args})
}

// repoPinnedGit returns the arguments of the git command sc runs and the
// repo-pin variables (isRepoPinVar) it sets, when those variables, set as
// prefix assignments or through env (`GIT_DIR=<dir> git ...`, `env
// GIT_DIR=<dir> git ...`), are the only ones it sets. It returns no pins for
// any other command, or when sc sets another variable, sets one dynamically,
// uses an expansion, or sets a pin to a protected path
// (assignValueProtected), which the pinned-write workaround would not clear.
func repoPinnedGit(sc scannedCommand) (args, pins []string) {
	if program := path.Base(sc.Name); program != "git" && program != "env" || sc.AssignsDynamic || sc.HasExpansion {
		return nil, nil
	}
	if slices.ContainsFunc(sc.Assigns, func(name string) bool { return !isRepoPinVar(name) }) {
		return nil, nil
	}
	words := append([]string{sc.Name}, sc.Args...)
	args, ok := gitInvocation(words)
	if !ok {
		return nil, nil
	}
	pins = slices.Clone(sc.Assigns)
	// Only NAME=value words may stand between env and git: an option (-i,
	// -u, -C) or another wrapper changes more than the repository.
	var between []string
	if gitAt := len(words) - len(args) - 1; gitAt > 0 {
		between = words[1:gitAt]
	}
	for _, w := range between {
		name, _, isAssign := strings.Cut(w, "=")
		if !isAssign || !isRepoPinVar(name) {
			return nil, nil
		}
		pins = append(pins, name)
	}
	if slices.ContainsFunc(sc.AssignTexts, func(a string) bool { return assignValueProtected(sc, a) }) ||
		slices.ContainsFunc(between, func(a string) bool { return assignValueProtected(sc, a) }) {
		return nil, nil
	}
	return args, pins
}

// assignValueProtected reports whether the rendered assignment (`NAME=value`)
// of a git command sets a variable to a protected path. A location pin
// (isLocationPinVar) is judged as the option naming the same location is
// (`--git-dir=<value>`, `--work-tree=<value>`: refersProtected), so
// `GIT_DIR=.claude git status` counts as `git --git-dir=.claude status` does
// and `GIT_DIR=../other/.git` does not. Any other value is judged as an
// operand is, a git-control holder included (gitControlHolder):
// `GIT_INDEX_FILE=.claude/hooks/x git status` rewrites that file.
func assignValueProtected(sc scannedCommand, assign string) bool {
	name, value, _ := strings.Cut(assign, "=")
	return refersProtected(sc, value) || !isLocationPinVar(name) && gitControlHolder(sc, value)
}

// prefixedGitJudged reports whether sc runs git directly with prefix
// assignments only (cmdscan.Command.PrefixAssignsOnly: `LC_ALL=C git log`)
// that SP-007 judges through the command they prefix, as it judges `env
// NAME=value git ...`: git is then not proven read-only, so each of its
// operands counts as a mutation target (isMutating, mutationTargets). It
// holds when no value names a protected path (assignValueProtected), none of
// the variables makes git run another program (isGitEnvVar, which GIT-001
// also judges: `GIT_EXTERNAL_DIFF=x git diff` stays a mutation here) and,
// when one of them pins the repository (isRepoPinVar), the git is a read
// (`TZ=UTC GIT_DIR=../other/.git git log`), as a variable-pinned git is judged
// only for reads (see pinnedGitWriteReason).
func prefixedGitJudged(sc scannedCommand) bool {
	if path.Base(sc.Name) != "git" || !sc.PrefixAssignsOnly() || slices.ContainsFunc(sc.Assigns, isGitEnvVar) ||
		slices.ContainsFunc(sc.AssignTexts, func(a string) bool { return assignValueProtected(sc, a) }) {
		return false
	}
	return !slices.ContainsFunc(sc.Assigns, isRepoPinVar) ||
		cmdscan.IsSafeReadCommand(cmdscan.Command{Name: "git", Args: sc.Args})
}

// pinnedGitWriteReason returns why a line that runs git in a repository
// pinned only by variables (repoPinnedGit) is denied when that git is not a
// read (`GIT_DIR=../o/.git git branch newb`): such a git is judged only for
// reads, and the --git-dir and --work-tree options name the repository in a
// form the guard judges. It returns "" when no command on the line is such a
// git, or when that git names a protected path or git-control file itself
// (`GIT_DIR=.git git log --output=.git/hooks/pre-commit`), which the
// workaround would not clear.
func pinnedGitWriteReason(scs []scannedCommand) string {
	for _, sc := range scs {
		args, pins := repoPinnedGit(sc)
		if len(pins) == 0 || repoPinnedRead(sc) || anyRefersProtected(sc, sc.WriteRedirects) ||
			slices.ContainsFunc(args, func(w string) bool { return namesGitControl(sc, w) }) {
			continue
		}
		slices.Sort(pins)
		return "git runs in a repository named through " + strings.Join(slices.Compact(pins), " and ") +
			", and a variable-pinned git is judged only for reads; name the repository with" +
			" git --git-dir=<dir> [--work-tree=<dir>] instead"
	}
	return ""
}

// namesGitControl reports whether a git argument names a protected path
// (argRefersProtected) or a git-control holder, as an operand or as the
// value of a long option (`--output=.git/hooks/pre-commit`).
func namesGitControl(sc scannedCommand, arg string) bool {
	if argRefersProtected(sc, arg) {
		return true
	}
	if name, value, ok := strings.Cut(arg, "="); ok && strings.HasPrefix(name, "--") {
		arg = value
	}
	return !isFlag(arg) && gitControlHolder(sc, arg)
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

// rewriteWalk checks the rewriting git commands of scs in order. A command
// that precedes a rewrite on its line may change the source the rewrite
// copies (the index, a fetched ref, the stash) before it runs; what each
// command may change is recorded in line (see rewriteLine), and a rewrite
// whose source a command may have changed in a way the model cannot follow
// is allowed only in a repository that tracks no protected file (askGit). A
// shell script (`sh -c`, eval) is walked in the directory its shell runs in.
func (ctx *EvalContext) rewriteWalk(scs []scannedCommand, line rewriteLine) string {
	for i, sc := range scs {
		words := append([]string{sc.Name}, sc.Args...)
		if run, ok := cmdscan.Script(words); ok && !isPowerShell(ctx) {
			if reason := ctx.scriptRewriteReason(sc, run, line); reason != "" {
				return reason
			}
			line.changeAll()
			continue
		}
		args, ok := gitInvocation(words)
		if !ok {
			if reason := unknownProgramRewrite(sc); reason != "" {
				return reason
			}
			if !ctx.inertCommand(i, sc) {
				line.changeAll()
			}
			continue
		}
		target := splitGitArgs(args)
		rw, rewrites := parseWorktreeRewrite(target.sub, target.rest)
		if rewrites {
			if reason := ctx.checkRewrite(sc, target, rw, line); reason != "" {
				return reason
			}
		}
		line.ranGit(sc, args, target, rw, rewrites)
	}
	return ""
}

// unknownProgramRewrite returns why a command whose program word is an
// expansion the scan cannot render (`g=$(command -v git); $g reset --hard
// evil`) is denied when its arguments, read as git's, rewrite working-tree
// files: the program may be git, and what it would write cannot be checked.
// A program word the scan renders is judged as that program instead.
func unknownProgramRewrite(sc scannedCommand) string {
	if !sc.NameHasExpansion || sc.Name != "" {
		return ""
	}
	target := splitGitArgs(sc.Args)
	if _, ok := parseWorktreeRewrite(target.sub, target.rest); !ok {
		return ""
	}
	return "a program named through a shell expansion, run as git " + target.sub + ", rewrites working-tree files, which cannot be checked"
}

// scriptRewriteReason walks the script a shell command runs, from the
// directory that shell starts in, as part of line. A shell that reads startup
// files first may start anywhere.
func (ctx *EvalContext) scriptRewriteReason(sc scannedCommand, run cmdscan.ScriptRun, line rewriteLine) string {
	if line.depth >= maxScriptDepth {
		return "" // GIT-001 denies git nested this deep
	}
	sub := ctx.scriptContext(sc, run)
	scs, err := sub.scannedCommands()
	if err != nil {
		return ""
	}
	line.depth++
	return sub.rewriteWalk(scs, line)
}

// scriptContext returns the context of the script run the shell command sc
// runs: a Bash line in the directory that shell starts in, or in an unknown
// one when the shell reads startup files first (which may change directory).
func (ctx *EvalContext) scriptContext(sc scannedCommand, run cmdscan.ScriptRun) *EvalContext {
	sub := &EvalContext{ToolName: "Bash", Command: run.Script, Ctx: ctx.Ctx, gitRunner: ctx.gitRunner}
	if !sc.cwdUnknown && !run.ReadsStartup {
		sub.CWD = sc.cwd
	}
	return sub
}

// checkRewrite judges one rewriting git command.
func (ctx *EvalContext) checkRewrite(sc scannedCommand, target gitArgs, rw worktreeRewrite, line rewriteLine) string {
	what := "git " + target.sub
	if (target.sub == "stash" || target.sub == "sparse-checkout") && len(target.rest) > 0 && !isFlag(target.rest[0]) {
		what += " " + target.rest[0] // stash push, save, pop, apply or branch; sparse-checkout set, ...
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
	if rw.source == fromPatches && line.readsChanged(rw) {
		// The patch may name any path, tracked or not, so tracking tells
		// nothing about a patch an earlier command may have written.
		return what + " applies a patch after another command on the same line, which may change the patch; run it in its own call so the files it writes can be checked"
	}
	if ctx.Ctx == nil {
		return what + " rewrites working-tree files, and this check has no deadline to ask git which"
	}
	verdict, err := ctx.askGit(dir, rw, line)
	switch {
	case errors.Is(err, gitrepo.ErrTimeout):
		return what + " rewrites working-tree files, and git did not answer in time to tell which"
	case errors.Is(err, errPatchUnread):
		return what + " applies a patch that git apply cannot read in advance (am also takes a Maildir), so the files it writes cannot be checked; save the patch as a file and apply that file"
	case err != nil:
		return "" // not a repository or an unknown revision: git itself fails
	case verdict.tracked != "" && rw.clean != nil:
		return what + " removes untracked files after another command on the same line, which may change what it removes, in a repository that tracks protected " + verdict.tracked + "; run it in its own call so the files it removes can be checked"
	case verdict.tracked != "":
		return what + " rewrites working-tree files after another command on the same line, which may change what it copies, in a repository that tracks protected " + verdict.tracked + "; run it in its own call so the files it writes can be checked"
	case verdict.sparse != "":
		return what + " changes which tracked files stay in the working tree, and may remove protected " + verdict.sparse + " that the repository tracks; a person must change the sparse-checkout patterns"
	case verdict.overwritten != "" && rw.source == removesTracked:
		return what + " would delete protected " + verdict.overwritten + " from the working tree; name the paths to remove, or untrack them only with git rm --cached"
	case verdict.overwritten != "":
		return what + " would overwrite protected " + verdict.overwritten + " in the working tree"
	case verdict.removed != "" && target.sub == "clean":
		return what + " would remove protected " + verdict.removed + " from the working tree, which the index does not track; limit it with a pathspec (git clean -fdx -- build dist) or exclude the protected path (for example -e .claude)"
	case verdict.removed != "":
		return what + " would remove protected " + verdict.removed + " from the working tree, which the index does not track"
	}
	return ""
}

// gitVerdict is what git reported about a rewrite: the protected path it
// would overwrite or, as an untracked file, remove; for a rewrite preceded on
// its line, a protected path the repository tracks; for a sparse-checkout
// pattern change, a tracked protected path it may drop.
type gitVerdict struct {
	overwritten, removed, tracked, sparse string
}

// askGit asks git, read-only and in dir, about rewrite rw. A rewrite whose
// source an earlier command of its line may have changed in a way the line's
// model cannot follow (precededBy) is known only in a repository that tracks
// no protected file; any other rewrite is judged by the paths it would
// write, with those an earlier commit recorded or an earlier add --chmod
// staged a mode for. When the first operand of an
// ambiguous checkout names no revision, git reads its operands as paths, and
// so does askGit. A sparse-checkout pattern change is judged by whether the
// repository tracks a protected path at all, and a clean (rw.clean) by the
// untracked paths it would remove, also after another command, as tracking
// may not cover them.
func (ctx *EvalContext) askGit(dir string, rw worktreeRewrite, line rewriteLine) (gitVerdict, error) {
	client := gitrepo.Client{Runner: ctx.gitRunner}
	top, err := client.Toplevel(ctx.Ctx, dir)
	if errors.Is(err, gitrepo.ErrNotRepo) && rw.source == fromPatches {
		top, err = dir, nil // git apply patches files outside a repository too
	}
	if err != nil {
		return gitVerdict{}, err
	}
	if rw.remote != "" && !remoteIsRepo(rw.remote, dir, top) {
		return gitVerdict{}, nil // a pull from another repository: the residual
	}
	preceded, also, err := ctx.precededBy(client, top, dir, rw, line)
	if err != nil {
		return gitVerdict{}, err
	}
	if preceded || rw.source == sparseRules {
		tracked, err := client.TrackedPaths(ctx.Ctx, top, nil)
		first := firstProtected(top, tracked)
		if rw.source == sparseRules {
			return gitVerdict{sparse: first}, err
		}
		if err != nil || first != "" || rw.clean == nil {
			return gitVerdict{tracked: first}, err
		}
	} else {
		paths, err := rewrittenPaths(ctx.Ctx, client, dir, rw)
		if errors.Is(err, gitrepo.ErrGit) && rw.orPaths != nil {
			paths, err = rewrittenPaths(ctx.Ctx, client, dir, worktreeRewrite{source: fromIndex, paths: rw.orPaths})
		}
		if first := firstProtected(top, append(paths, also...)); err != nil || first != "" || rw.clean == nil {
			return gitVerdict{overwritten: first}, err
		}
	}
	removed, err := client.CleanPaths(ctx.Ctx, dir, *rw.clean, rw.paths)
	return gitVerdict{removed: ctx.firstRemovedProtected(dir, removed)}, err
}

// firstRemovedProtected returns the first of paths, relative to dir, whose
// removal takes a protected path with it: a protected file or directory, an
// entry in a protected directory, or a directory removed as a whole (a
// trailing "/") that holds a git directory or a .claude-like location
// (holdsProtectedEntry). It returns "" when there is none.
func (ctx *EvalContext) firstRemovedProtected(dir string, paths []string) string {
	for _, p := range paths {
		abs := filepath.Join(dir, filepath.FromSlash(p))
		if pathProtected(abs) || strings.HasSuffix(p, "/") && holdsProtectedEntry(&ctx.fs, abs) {
			return p
		}
	}
	return ""
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
	paths, err := sourcePaths(ctx, client, dir, rw)
	if err != nil {
		return nil, err
	}
	for _, more := range rw.also {
		p, err := rewrittenPaths(ctx, client, dir, more)
		if err != nil {
			return nil, err
		}
		paths = append(paths, p...)
	}
	return paths, nil
}

// sourcePaths asks git which paths rw's own comparison writes.
func sourcePaths(ctx context.Context, client gitrepo.Client, dir string, rw worktreeRewrite) ([]string, error) {
	base := cmp.Or(rw.base, "HEAD")
	switch rw.source {
	case fromIndex:
		return client.ChangedPaths(ctx, dir, "", "", rw.paths)
	case fromRev:
		return client.ChangedPaths(ctx, dir, rw.rev, "", rw.paths)
	case betweenRevs:
		return client.ChangedPaths(ctx, dir, base, rw.rev, rw.paths)
	case fromStash:
		return client.StashPaths(ctx, dir, rw.rev)
	case mergeBase:
		revs, err := mergeHeads(ctx, client, dir, rw.revs)
		if err != nil {
			return nil, err
		}
		var paths []string
		for _, rev := range revs {
			p, err := mergedPaths(ctx, client, dir, base, rev)
			if err != nil {
				return nil, err
			}
			paths = append(paths, p...)
		}
		return paths, nil
	case fromCommits:
		return client.CommitPaths(ctx, dir, rw.revs)
	case removesTracked:
		return client.TrackedPaths(ctx, dir, rw.paths)
	case fromPatches:
		paths, err := client.PatchPaths(ctx, dir, rw.patches, rw.strip, rw.directory)
		if err != nil && !errors.Is(err, gitrepo.ErrTimeout) {
			// git am also takes a Maildir, which git apply cannot read.
			return nil, fmt.Errorf("%w: %w", errPatchUnread, err)
		}
		return paths, err
	}
	return nil, nil
}

// mergeHeads returns the revisions a merge-family command takes in: revs,
// with FETCH_HEAD standing for every head the last fetch recorded for
// merging (`git fetch . a evil && git merge FETCH_HEAD` merges both, while a
// revision lookup of FETCH_HEAD resolves only the first).
func mergeHeads(ctx context.Context, client gitrepo.Client, dir string, revs []string) ([]string, error) {
	if !slices.Contains(revs, "FETCH_HEAD") {
		return revs, nil
	}
	heads, err := client.FetchHeadMerges(ctx, dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, rev := range revs {
		if rev == "FETCH_HEAD" {
			out = append(out, heads...)
		} else {
			out = append(out, rev)
		}
	}
	return out, nil
}

// mergedPaths asks git which paths merging rev into base, or rebasing base
// onto rev, writes: what rev changed since their merge base (`git diff
// base...rev`). A rev that shares no history with base has no merge base,
// and git still merges it (--allow-unrelated-histories) or rebases onto it,
// writing any path where the two trees differ; the two-dot diff names those.
// An error of that diff too means git cannot resolve rev and itself fails.
func mergedPaths(ctx context.Context, client gitrepo.Client, dir, base, rev string) ([]string, error) {
	paths, err := client.ChangedPaths(ctx, dir, base+"..."+rev, "", nil)
	if errors.Is(err, gitrepo.ErrGit) {
		return client.ChangedPaths(ctx, dir, base, rev, nil)
	}
	return paths, err
}

// errPatchUnread is a patch git apply --numstat could not read: what applying
// it writes cannot be checked.
var errPatchUnread = errors.New("patch cannot be read")
