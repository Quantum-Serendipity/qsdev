package rules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/gitrepo"
	"github.com/Quantum-Serendipity/qsdev/internal/procexec"
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// fakeGit is a gitrepo.Runner that answers from canned output keyed by the
// query (fakeKey) and records every call, so the worktree-rewrite rule can be
// judged without a repository and its git spawns counted. An ls-files with a
// pathspec is answered from answers.
type fakeGit struct {
	top     string            // the toplevel rev-parse prints; "" means not a repository
	answers map[string]string // fakeKey -> NUL-separated paths
	tracked string            // the NUL-separated paths ls-files prints without a pathspec
	err     error             // returned for every query when set
	// fetchHead is the FETCH_HEAD file `rev-parse --git-path FETCH_HEAD`
	// prints; "" makes that query fail.
	fetchHead string

	mu    sync.Mutex
	calls []gitrepo.Call
}

// fakeKey names a query by the arguments that matter to it: "toplevel",
// "fetch-head", "tracked [<pathspec>...]", "clean <options> -- [<pathspec>...]" (its answer
// is git clean --dry-run's text), "diff <from> [<to>] -- [<pathspec>...]", "stash
// <ref>", "log <rev>... --" or "apply [-p<n>] [--directory=<d>] -- <file>..."
// (the forward and the reverse (-R) read of a patch share a key and an
// answer, in --numstat form).
func fakeKey(args []string) string {
	switch {
	case slices.Contains(args, "rev-parse") && slices.Contains(args, "FETCH_HEAD"):
		return "fetch-head"
	case slices.Contains(args, "rev-parse"):
		return "toplevel"
	case slices.Contains(args, "ls-files"):
		return keyAfter("tracked", args, "--")
	case slices.Contains(args, "clean"):
		return keyAfter("clean", args, "--dry-run")
	case slices.Contains(args, "stash"):
		return "stash " + args[len(args)-1]
	case slices.Contains(args, "diff"):
		return keyAfter("diff", args, "--end-of-options")
	case slices.Contains(args, "log"):
		return keyAfter("log", args, "--end-of-options")
	case slices.Contains(args, "apply"):
		return keyAfter("apply", slices.DeleteFunc(slices.Clone(args), func(a string) bool { return a == "-R" }), "-z")
	}
	return strings.Join(args, " ")
}

// keyAfter joins name and the arguments after the marker argument.
func keyAfter(name string, args []string, marker string) string {
	return strings.Join(append([]string{name}, args[slices.Index(args, marker)+1:]...), " ")
}

// numstat renders NUL-separated paths as git apply -z --numstat prints them.
func numstat(paths string) string {
	var b strings.Builder
	for p := range strings.SplitSeq(paths, "\x00") {
		if p != "" {
			b.WriteString("1\t1\t" + p + "\x00")
		}
	}
	return b.String()
}

func (f *fakeGit) run(_ context.Context, c gitrepo.Call) ([]byte, error) {
	f.mu.Lock()
	f.calls = append(f.calls, c)
	f.mu.Unlock()
	if f.err != nil {
		return nil, f.err
	}
	key := fakeKey(c.Args)
	if key == "toplevel" {
		if f.top == "" {
			return nil, fmt.Errorf("%w: fatal: not a git repository", gitrepo.ErrNotRepo)
		}
		return []byte(f.top + "\n"), nil
	}
	if key == "tracked" {
		return []byte(f.tracked), nil
	}
	if key == "fetch-head" && f.fetchHead != "" {
		return []byte(filepath.ToSlash(f.fetchHead) + "\n"), nil
	}
	out, ok := f.answers[key]
	if !ok {
		return nil, fmt.Errorf("%w (exit 128): fatal: bad revision", gitrepo.ErrGit)
	}
	return []byte(out), nil
}

func (f *fakeGit) callCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.calls)
}

// rewriteRepo returns a fake repository at top. The evil one tracks
// .claude/settings.json and pkg/.claude/settings.json, and branch evil, the
// previous branch, FETCH_HEAD, the index and the stash all change them; the
// clean one tracks and changes only src/x.go. In both, `:/`, `:(top)`,
// `:!x`, `src/..` and `pkg` name no revision (git fails to diff them as
// one) and select the changed paths as a pathspec.
func rewriteRepo(top string, evil bool) *fakeGit {
	changed := "src/x.go\x00"
	if evil {
		changed = ".claude/settings.json\x00pkg/.claude/settings.json\x00src/x.go\x00"
	}
	nested := ""
	if evil {
		nested = "pkg/.claude/settings.json\x00"
	}
	return &fakeGit{top: top, tracked: changed, answers: map[string]string{
		"diff -- :/":              changed,
		"diff -- :(top)":          changed,
		"diff -- :!x":             changed,
		"diff -- src/..":          changed,
		"diff -- pkg":             nested,
		"diff HEAD @{-1} --":      changed,
		"diff --":                 changed,
		"diff -- .":               changed,
		"diff evil --":            changed,
		"diff evil -- .":          changed,
		"diff HEAD evil --":       changed,
		"diff FETCH_HEAD --":      changed,
		"diff -- src/x.go":        "src/x.go\x00",
		"diff -- x.go":            "src/x.go\x00",
		"diff HEAD main --":       "src/x.go\x00",
		"stash stash@{0}":         changed,
		"diff HEAD -- src/x.go":   "src/x.go\x00",
		"diff HEAD~1 --":          changed,
		"diff evil -- src":        "src/x.go\x00",
		"diff evil -- .claude":    changed,
		"diff evil -- sub":        "",
		"diff HEAD evil -- other": "",
		"diff HEAD --":            "", // the work tree matches HEAD
		// Merge family: what evil changed since its merge base, the commits
		// on evil, and patch files x.patch and x.mbox carry the change;
		// branches a, feature and main and fix.patch change only src/x.go.
		"diff HEAD...evil --":               changed,
		"diff HEAD..." + evilOID + " --":    changed,
		"diff HEAD..." + featureOID + " --": "src/x.go\x00",
		// orphan shares no history with HEAD: the merge-base diff fails, and
		// the two trees differ wherever the branch's tree does.
		"diff HEAD orphan --":                  changed,
		"diff HEAD...a --":                     "src/x.go\x00",
		"diff HEAD...feature --":               "src/x.go\x00",
		"diff HEAD...main --":                  "src/x.go\x00",
		"diff HEAD...@{upstream} --":           "src/x.go\x00",
		"diff HEAD feature --":                 "src/x.go\x00",
		"diff feature...main --":               "src/x.go\x00",
		"diff evil...main --":                  "src/x.go\x00",
		"log evil --":                          changed,
		"log main..evil --":                    changed,
		"log feature --":                       "src/x.go\x00",
		"apply -- x.patch":                     numstat(changed),
		"apply -- x.mbox":                      numstat(changed),
		"apply -p0 --directory=sub -- x.patch": numstat(changed),
		"apply -- fix.patch":                   numstat("src/x.go\x00"),
		// git rm: the tracked paths each pathspec matches.
		"tracked .":          changed,
		"tracked :/":         changed,
		"tracked .claude/..": changed,
		"tracked src":        "src/x.go\x00",
	}}
}

// The objects the fake FETCH_HEAD files name: evilOID is branch evil's tip,
// featureOID branch feature's.
var (
	evilOID    = strings.Repeat("e", 40)
	featureOID = strings.Repeat("f", 40)
)

// withFetchHead gives git a FETCH_HEAD file holding content.
func withFetchHead(t *testing.T, git *fakeGit, content string) *fakeGit {
	t.Helper()
	git.fetchHead = filepath.Join(t.TempDir(), "FETCH_HEAD")
	if err := os.WriteFile(git.fetchHead, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return git
}

// headMovedRepo is the evil rewriteRepo after `git reset --soft evil` (or
// update-ref, or symbolic-ref) moved HEAD to evil: the work tree's protected
// files now differ from HEAD.
func headMovedRepo(top string) *fakeGit {
	git := rewriteRepo(top, true)
	git.answers["diff HEAD --"] = git.answers["diff --"]
	return git
}

// rewriteVerdict evaluates command with the Tier-1 rules against git, and
// returns the verdict and the SP-007 reason ("" when SP-007 did not match).
func rewriteVerdict(t *testing.T, tool, command, cwd string, git *fakeGit) (Verdict, string, []RuleMatch) {
	t.Helper()
	ctx := EvalContext{ToolName: tool, Command: command, CWD: cwd, Ctx: t.Context(), gitRunner: git.run}
	got, matches := Tier1Rules.EvaluateAll(&ctx)
	for _, m := range matches {
		if m.Rule.ID == "SP-007" {
			return got, m.Reason, matches
		}
	}
	return got, "", matches
}

// TestWorktreeRewriteWithTrackedProtected pins U18-02c: a git command that
// rewrites working-tree files is denied when it would overwrite a tracked
// protected file, judged by what git reports would change, and allowed when
// the protected files would not change, outside a repository, or for an
// unknown revision (git itself fails). A rewrite after another command of the
// line is denied in a repository that tracks a protected file and allowed in
// one that tracks none. A rewrite in an unknown directory, or when git does
// not answer in time, is denied because what it writes cannot be checked.
func TestWorktreeRewriteWithTrackedProtected(t *testing.T) {
	t.Parallel()
	top := filepath.Join(homeDir(t), "project")
	const overwrite = "would overwrite protected .claude/settings.json in the working tree"
	const tracksOwnCall = "in a repository that tracks protected .claude/settings.json; run it in its own call"

	// Rewrites whose verdict follows the repository: denied in the evil one,
	// allowed in the clean one. wantDir is the directory git must run in.
	repoRows := []struct {
		command string
		wantDir string
	}{
		{"git checkout-index -a -f", top},
		{"git checkout evil -- .", top},
		{"git checkout -f evil", top},
		{"git reset --hard evil", top},
		{"git stash pop", top},
		{"git stash apply", top},
		{"git restore --source evil .", top},
		{"git restore --source=evil -W .", top},
		{"git read-tree -u -m evil", top},
		{"git switch evil", top},
		{"git checkout evil", top},
		{"cd sub && git reset --hard evil", filepath.Join(top, "sub")},
		{"git -C . reset --hard evil", top},
		{"git -C sub reset --hard evil", filepath.Join(top, "sub")},
		{"sh -c 'git reset --hard evil'", top},
		{"cd sub && sh -c 'cd .. && git reset --hard evil'", top},
		{"git status && git reset --hard evil", top},
		{"sudo git reset --hard evil", top},
		{"git reset --har evil", top},
		// An operand that names no revision is a pathspec restored from
		// the index (after `git reset evil` staged the protected file).
		{"git checkout :/", top},
		{"git checkout ':(top)'", top},
		{"git checkout ':!x'", top},
		{"git checkout src/..", top},
		// "-" is the previous branch.
		{"git checkout -", top},
		{"git switch -", top},
		// An unborn branch with a start point, and a patch restore of the
		// whole tree from a revision.
		{"git checkout --orphan x evil", top},
		{"git checkout -q --orphan=x evil", top},
		{"git restore -p -s evil", top},
		{"git restore --patch --source=evil", top},
		// After an index-only change (`git reset evil`, `git read-tree
		// evil`), a patch checkout or restore copies the index, and a stash
		// that keeps the index sets the work tree to it.
		{"git checkout -p", top},
		{"git restore -p", top},
		{"git stash -k", top},
		{"git stash push --keep-index", top},
	}
	for _, row := range repoRows {
		for _, evil := range []bool{true, false} {
			name := fmt.Sprintf("%s/evil=%v", row.command, evil)
			t.Run(name, func(t *testing.T) {
				t.Parallel()
				git := rewriteRepo(top, evil)
				got, reason, matches := rewriteVerdict(t, "Bash", row.command, top, git)
				if evil {
					if got != Deny || !strings.Contains(reason, overwrite) {
						t.Errorf("verdict = %v, SP-007 reason %q (matches %v), want Deny with %q", got, reason, matchIDs(matches), overwrite)
					}
				} else if got != Allow {
					t.Errorf("verdict = %v (matches %v, SP-007 %q), want Allow", got, matchIDs(matches), reason)
				}
				for _, c := range git.calls {
					if c.Dir != row.wantDir {
						t.Errorf("git ran in %q, want %q", c.Dir, row.wantDir)
					}
				}
			})
		}
	}

	evilRepo := func() *fakeGit { return rewriteRepo(top, true) }
	rows := []struct {
		name    string
		tool    string
		command string
		cwd     string
		git     *fakeGit
		want    Verdict
		reason  string // a substring of SP-007's reason when want is Deny
	}{
		// A command before the rewrite may change what it reads: denied in a
		// repository that tracks a protected file, whatever the rewrite's
		// own target, and allowed in one that tracks none.
		{"index replaced first, evil", "Bash", "cp /tmp/i .git/index && git checkout-index -a -f", top, evilRepo(), Deny, tracksOwnCall},
		{"fetch first, evil", "Bash", "git fetch o && git reset --hard FETCH_HEAD", top, evilRepo(), Deny, tracksOwnCall},
		{"echo writing a file first, evil", "Bash", "echo hi > x; git reset --hard HEAD", top, evilRepo(), Deny, tracksOwnCall},
		{"second rewrite on the line, evil", "Bash", "git checkout -- src/x.go && git reset --hard evil", top, evilRepo(), Deny, tracksOwnCall},
		// (The fake answers no untracked-file query, so the add counts as
		// an unmodelled change; TestWorktreeRewrite_PrecedingCommands
		// judges an add and a commit against a repository that answers.)
		{"commit then benign switch, evil", "Bash", "git add -A && git commit -m x && git switch main", top, evilRepo(), Deny, tracksOwnCall},
		// The stash push may save nothing, and the pop then applies the
		// stash entry that changes the protected file.
		{"stash around a benign switch, evil", "Bash", "git stash && git switch main && git stash pop", top, evilRepo(), Deny, "git stash pop " + overwrite},
		// echo cannot change what the reset copies: judged on its own.
		{"echo first, evil", "Bash", "echo hi; git reset --hard HEAD", top, evilRepo(), Allow, ""},
		// A branch or tag creation changes what a later checkout copies; a
		// read-only form (cmdscan.GitReadOnly) does not.
		{"branch creation first, evil", "Bash", "git branch newb && git checkout newb", top, evilRepo(), Deny, tracksOwnCall},
		{"tag creation first, evil", "Bash", "git tag v1 && git checkout -- src/x.go", top, evilRepo(), Deny, tracksOwnCall},
		{"ls-tree first, evil", "Bash", "git ls-tree HEAD; git checkout -- src/x.go", top, evilRepo(), Allow, ""},
		{"branch listing first, evil", "Bash", "git branch -a; git stash list && git checkout -- src/x.go", top, evilRepo(), Allow, ""},
		{"index replaced first, clean", "Bash", "cp /tmp/i .git/index && git checkout-index -a -f", top, rewriteRepo(top, false), Allow, ""},
		{"fetch first, clean", "Bash", "git fetch o && git reset --hard FETCH_HEAD", top, rewriteRepo(top, false), Allow, ""},
		{"echo first, clean", "Bash", "echo hi; git reset --hard HEAD", top, rewriteRepo(top, false), Allow, ""},
		{"second rewrite on the line, clean", "Bash", "git checkout -- src/x.go && git reset --hard evil", top, rewriteRepo(top, false), Allow, ""},
		{"commit then switch, clean", "Bash", "git add -A && git commit -m x && git checkout main", top, rewriteRepo(top, false), Allow, ""},
		{"commit -am then switch, clean", "Bash", "git commit -am wip && git switch evil", top, rewriteRepo(top, false), Allow, ""},
		{"stash around a switch, clean", "Bash", "git stash && git checkout evil && git stash pop", top, rewriteRepo(top, false), Allow, ""},
		{"tests then checkout, clean", "Bash", "go test ./... && git checkout -- src/x.go", top, rewriteRepo(top, false), Allow, ""},
		{"fetch then checkout, clean", "Bash", "git fetch origin && git checkout evil", top, rewriteRepo(top, false), Allow, ""},
		{"new repository", "Bash", "mkdir r && cd r && git init && git checkout -b x evil", top, &fakeGit{}, Allow, ""},
		{"fetch then reset in its own call", "Bash", "git reset --hard FETCH_HEAD", top, evilRepo(), Deny, overwrite},
		{"stash pop names its action", "Bash", "git stash pop", top, evilRepo(), Deny, "git stash pop " + overwrite},
		// A directory operand that names no revision restores the nested
		// protected file from the index.
		{"checkout of a directory", "Bash", "git checkout pkg", top, evilRepo(), Deny, "would overwrite protected pkg/.claude/settings.json"},
		{"checkout of a directory, clean", "Bash", "git checkout pkg", top, rewriteRepo(top, false), Allow, ""},
		// What git would write cannot be known.
		{"unknown directory", "Bash", `cd "$D" && git reset --hard evil`, top, rewriteRepo(top, false), Deny, "not known"},
		{"unknown session directory", "Bash", "git reset --hard evil", "", rewriteRepo(top, false), Deny, "not known"},
		{"expansion in the rev", "Bash", "git reset --hard $REV", top, rewriteRepo(top, false), Deny, "expansion"},
		{"repository chosen by option", "Bash", "git --git-dir=/tmp/x/.git --work-tree=. reset --hard evil", top, rewriteRepo(top, false), Deny, "repository"},
		// (A .git component on the line would let the generic mutation rule
		// deny the assignment first.)
		{"repository chosen by variable", "Bash", "GIT_DIR=/tmp/x.git git reset --hard evil", top, rewriteRepo(top, false), Deny, "repository"},
		// A write-target pin still chooses what a rewrite reads (as on base).
		{"index chosen by variable", "Bash", "GIT_INDEX_FILE=x git checkout evil -- .", top, rewriteRepo(top, false), Deny, "repository chosen by option or variable"},
		{"checkout-index with a prefix", "Bash", "git checkout-index -a --prefix=out/", top, rewriteRepo(top, false), Deny, "prefix"},
		{"timeout", "Bash", "git reset --hard evil", top, &fakeGit{err: fmt.Errorf("%w: slow", gitrepo.ErrTimeout)}, Deny, "in time"},
		// PowerShell: the same verdicts in its dialect, and a location
		// change on the line denies.
		{"powershell evil", "PowerShell", "git reset --hard evil", top, evilRepo(), Deny, overwrite},
		{"powershell clean", "PowerShell", "git reset --hard evil", top, rewriteRepo(top, false), Allow, ""},
		{"powershell location change", "PowerShell", `Set-Location sub; git reset --hard evil`, top, rewriteRepo(top, false), Deny, "location"},
		// Allowed: the protected files would not change, the command does
		// not rewrite the working tree, or git itself would fail.
		{"checkout of a source file", "Bash", "git checkout -- src/x.go", top, evilRepo(), Allow, ""},
		{"checkout of a source file from a subdirectory", "Bash", "cd src && git checkout -- x.go", top, evilRepo(), Allow, ""},
		{"restore of a source file from HEAD", "Bash", "git restore --source HEAD src/x.go", top, evilRepo(), Allow, ""},
		{"restore of the source dir from evil", "Bash", "git restore -s evil src", top, evilRepo(), Allow, ""},
		{"restore --staged", "Bash", "git restore --staged x", top, evilRepo(), Allow, ""},
		{"reset --soft", "Bash", "git reset --soft HEAD~1", top, evilRepo(), Allow, ""},
		{"reset --mixed", "Bash", "git reset HEAD~1", top, evilRepo(), Allow, ""},
		{"stash push", "Bash", "git stash", top, evilRepo(), Allow, ""},
		{"stash push of untracked files", "Bash", "git stash -u", top, evilRepo(), Allow, ""},
		// After HEAD moved to evil, a stash resets the work tree to it.
		// A forced new branch discards the local changes to HEAD's version.
		{"forced new branch after HEAD moved", "Bash", "git checkout -f -b feature", top, headMovedRepo(top), Deny, "git checkout " + overwrite},
		{"new branch after HEAD moved", "Bash", "git checkout -b feature", top, headMovedRepo(top), Allow, ""},
		{"stash push after HEAD moved", "Bash", "git stash", top, headMovedRepo(top), Deny, "git stash " + overwrite},
		{"stash -u after HEAD moved", "Bash", "git stash -u", top, headMovedRepo(top), Deny, overwrite},
		{"stash save after HEAD moved", "Bash", "git stash save wip", top, headMovedRepo(top), Deny, "git stash save " + overwrite},
		{"stash of another path after HEAD moved", "Bash", "git stash push -- src/x.go", top, headMovedRepo(top), Allow, ""},
		{"new branch from HEAD", "Bash", "git checkout -b feature", top, evilRepo(), Allow, ""},
		{"switch to an unchanged branch", "Bash", "git switch main", top, evilRepo(), Allow, ""},
		{"outside a repository", "Bash", "git reset --hard evil", top, &fakeGit{}, Allow, ""},
		{"unknown revision", "Bash", "git reset --hard nosuch", top, evilRepo(), Allow, ""},
		{"read-only git after a rewrite", "Bash", "git switch main && git log", top, evilRepo(), Allow, ""},
		// git rm deletes every tracked path its pathspec matches from the
		// working tree (W1 wave fix); --cached only untracks them, and a dry
		// run removes nothing.
		{"rm of the whole tree", "Bash", "git rm -r .", top, evilRepo(), Deny, "git rm would delete protected .claude/settings.json from the working tree"},
		{"rm of the whole tree after --", "Bash", "git rm -r -q -- .", top, evilRepo(), Deny, "would delete protected .claude/settings.json"},
		{"rm through a parent of .claude", "Bash", "git rm -r -- .claude/..", top, evilRepo(), Deny, ""},
		{"rm of a pathspec read from a file", "Bash", "git rm --pathspec-from-file=list", top, evilRepo(), Deny, "would delete protected .claude/settings.json"},
		{"powershell rm of the whole tree", "PowerShell", "git rm -r .", top, evilRepo(), Deny, "would delete protected .claude/settings.json"},
		{"rm of the whole tree, clean", "Bash", "git rm -r .", top, rewriteRepo(top, false), Allow, ""},
		{"rm of a source directory", "Bash", "git rm -r src", top, evilRepo(), Allow, ""},
		{"rm --cached of the whole tree", "Bash", "git rm --cached -r .", top, evilRepo(), Allow, ""},
		{"rm dry run of the whole tree", "Bash", "git rm -n -r .", top, evilRepo(), Allow, ""},
		{"rm dry run, long", "Bash", "git rm -r --dry-run .", top, evilRepo(), Allow, ""},
	}
	for _, tt := range rows {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, reason, matches := rewriteVerdict(t, tt.tool, tt.command, tt.cwd, tt.git)
			if got != tt.want {
				t.Fatalf("%q: verdict = %v (matches %v, SP-007 %q), want %v", tt.command, got, matchIDs(matches), reason, tt.want)
			}
			if tt.want == Deny && !strings.Contains(reason, tt.reason) {
				t.Errorf("%q: SP-007 reason = %q, want it to contain %q", tt.command, reason, tt.reason)
			}
		})
	}

	// Without the hook's deadline no git query runs, and the rewrite cannot
	// be checked.
	t.Run("no deadline", func(t *testing.T) {
		t.Parallel()
		git := rewriteRepo(top, false)
		ctx := EvalContext{ToolName: "Bash", Command: "git reset --hard evil", CWD: top, gitRunner: git.run}
		got, matches := Tier1Rules.EvaluateAll(&ctx)
		if got != Deny || firstRuleID(matches) != "SP-007" || git.callCount() != 0 {
			t.Errorf("verdict = %v (matches %v), %d git calls; want an SP-007 Deny with none", got, matchIDs(matches), git.callCount())
		}
	})

	// Every shell tool judges a rewrite (Monitor reads POSIX like Bash).
	for _, tool := range cmdscan.ShellTools {
		t.Run("shell tool "+tool, func(t *testing.T) {
			t.Parallel()
			if got, reason, _ := rewriteVerdict(t, tool, "git reset --hard evil", top, rewriteRepo(top, true)); got != Deny || !strings.Contains(reason, overwrite) {
				t.Errorf("%s: verdict = %v, SP-007 reason %q, want Deny with %q", tool, got, reason, overwrite)
			}
		})
	}
}

// TestWorktreeRewrite_PrecedingCommands pins what an earlier command of the
// line changes for a rewrite (W1 wave fix, rewriteLine): a command that
// cannot change the index, a ref, the stash or a patch (echo, true, cat, a
// read-only git) leaves the rewrite judged on its own; a commit adds the
// paths it records, an add counts when an untracked protected path exists, a
// stash push leaves stash@{0} to its own check, and a fetch or pull keeps the
// stash. Anything else is judged by whether the repository tracks a
// protected path, as before.
func TestWorktreeRewrite_PrecedingCommands(t *testing.T) {
	t.Parallel()
	top := filepath.Join(homeDir(t), "project")
	const tracksOwnCall = "in a repository that tracks protected .claude/settings.json; run it in its own call"
	// tracked tracks .claude/settings.json, but neither its branches, its
	// stash, its index nor its working tree change it, and it has no
	// untracked file.
	tracked := func() *fakeGit {
		git := rewriteRepo(top, false)
		git.tracked = ".claude/settings.json\x00src/x.go\x00"
		git.answers["clean -d -f --"] = ""
		git.answers["clean -d -x -f --"] = ""
		git.answers["tracked src/x.go"] = "src/x.go\x00"
		return git
	}
	// wholeTree is tracked where `.` matches every tracked path.
	wholeTree := func() *fakeGit {
		git := tracked()
		git.answers["tracked ."] = git.tracked
		return git
	}
	// protectedChmod is tracked where a pathspec names the protected file.
	protectedChmod := func() *fakeGit {
		git := tracked()
		git.answers["tracked .claude/settings.json"] = ".claude/settings.json\x00"
		return git
	}
	// untracked also holds an untracked protected file a git add may track.
	untracked := func() *fakeGit {
		git := tracked()
		git.answers["clean -d -f --"] = "Would remove .claude/settings.local.json\n"
		git.answers["clean -d -x -f --"] = "Would remove .claude/settings.local.json\n"
		return git
	}
	// ignored holds an ignored protected directory, which only add --force
	// tracks.
	ignored := func() *fakeGit {
		git := tracked()
		git.answers["clean -d -x -f --"] = "Would remove .qsdev/\n"
		return git
	}
	// unborn has no HEAD: what a commit records cannot be listed.
	unborn := func() *fakeGit {
		git := tracked()
		delete(git.answers, "diff HEAD --")
		return git
	}
	// evil's index holds a protected change that a commit records.
	evil := func() *fakeGit {
		git := rewriteRepo(top, true)
		git.answers["clean -d -x -f --"] = ""
		return git
	}
	tests := []struct {
		name, tool, command string
		git                 *fakeGit
		want                Verdict
		reason              string // a substring of SP-007's reason when want is Deny
	}{
		{"stash, pull, pop", "Bash", "git stash && git pull && git stash pop", tracked(), Allow, ""},
		{"stash, pull, pop in PowerShell", "PowerShell", "git stash; git pull; git stash pop", tracked(), Allow, ""},
		{"stash and pop", "Bash", "git stash && git stash pop", tracked(), Allow, ""},
		{"commit then switch", "Bash", "git commit -am x && git switch main", tracked(), Allow, ""},
		{"add, commit, checkout", "Bash", "git add -A && git commit -m x && git checkout main", tracked(), Allow, ""},
		{"new branch then pop", "Bash", "git checkout -b topic && git stash pop", tracked(), Allow, ""},
		{"fetch then pop", "Bash", "git fetch origin && git stash pop", tracked(), Allow, ""},
		{"echo then checkout", "Bash", "echo start; git checkout -- src/x.go", tracked(), Allow, ""},
		{"true then restore", "Bash", "true && git restore src/x.go", tracked(), Allow, ""},
		{"no-op then restore", "Bash", ": && test -f go.mod && git restore src/x.go", tracked(), Allow, ""},
		{"cat then switch", "Bash", "cat go.mod && git switch main", tracked(), Allow, ""},
		{"commit then patch", "Bash", "git commit -m x && git apply fix.patch", tracked(), Allow, ""},
		{"dry-run add then checkout", "Bash", "git add -n . && git checkout -- src/x.go", untracked(), Allow, ""},
		{"add with an ignored protected directory", "Bash", "git add -A && git commit -m x && git checkout main", ignored(), Allow, ""},
		{"forced add with an ignored protected directory", "Bash", "git add -f -A && git reset --hard", ignored(), Deny, tracksOwnCall},
		// The commit records the protected change staged in the index, and
		// the switch then writes main's version over it.
		{"commit of a protected change then switch", "Bash", "git commit -am x && git switch main", evil(), Deny,
			"git switch would overwrite protected .claude/settings.json"},
		// An unmodelled change before the rewrite.
		{"tests then checkout", "Bash", "go test ./... && git checkout main", tracked(), Deny, tracksOwnCall},
		{"fetch then rebase", "Bash", "git fetch && git rebase main", tracked(), Deny, tracksOwnCall},
		{"checkout then merge", "Bash", "git checkout main && git merge feature", tracked(), Deny, tracksOwnCall},
		{"redirect then checkout", "Bash", "printf x > idx && git checkout -- src/x.go", tracked(), Deny, tracksOwnCall},
		{"wrapped commit then switch", "Bash", "env git commit -m x && git switch main", tracked(), Deny, tracksOwnCall},
		{"commit with -C then switch", "Bash", "git -C . commit -m x && git switch main", tracked(), Deny, tracksOwnCall},
		{"amend then switch", "Bash", "git commit --amend -m x && git switch main", tracked(), Deny, tracksOwnCall},
		{"commit then reset to the old HEAD", "Bash", "git commit -m x && git reset --hard HEAD~1", tracked(), Deny, tracksOwnCall},
		{"commit then reflog revision", "Bash", "git commit -m x && git checkout HEAD@{1} -- .", tracked(), Deny, tracksOwnCall},
		{"commit in an unborn repository", "Bash", "git commit -m x && git switch main", unborn(), Deny, tracksOwnCall},
		{"edited add then switch", "Bash", "git add -e && git switch main", tracked(), Deny, tracksOwnCall},
		{"add of an untracked protected file", "Bash", "git add -A && git checkout -- src/x.go", untracked(), Deny, tracksOwnCall},
		{"stash then patch", "Bash", "git stash && git apply fix.patch", tracked(), Deny, "applies a patch after another command"},
		// The stash list changed: stash@{0} is not the entry judged now.
		{"drop then pop", "Bash", "git stash drop && git stash pop", tracked(), Deny, tracksOwnCall},
		{"fetch into the stash then pop", "Bash", "git fetch . x:refs/stash && git stash pop", tracked(), Deny, tracksOwnCall},
		{"fetch with a refmap then pop", "Bash", "git fetch --refmap=x origin main && git stash pop", tracked(), Deny, tracksOwnCall},
		{"pop then pop", "Bash", "git stash pop && git stash pop", tracked(), Deny, tracksOwnCall},
		{"stash then an older entry", "Bash", "git stash && git stash pop stash@{1}", tracked(), Deny, tracksOwnCall},

		// A redirect that writes no file (W1 wave fix round 2).
		{"commit with stderr discarded then switch", "Bash", "git commit -am x 2>/dev/null && git switch main", tracked(), Allow, ""},
		{"read to /dev/null then switch", "Bash", "cat src/x.go > /dev/null && git switch main", tracked(), Allow, ""},
		{"commit with output merged then switch", "Bash", "git commit -am x >/dev/null 2>&1 && git switch main", tracked(), Allow, ""},
		{"commit logged to a file then switch", "Bash", "git commit -am x > log.txt && git switch main", tracked(), Deny, tracksOwnCall},
		// A commit message from a variable: an option's value, one word.
		{"commit with a quoted message variable then switch", "Bash", `git commit -am "$MSG" && git switch main`, tracked(), Allow, ""},
		{"commit with a substituted message then switch", "Bash", `git commit -m "wip $(date)" && git switch main`, tracked(), Deny, tracksOwnCall},
		{"commit with an unquoted message variable then switch", "Bash", "git commit -am $MSG && git switch main", tracked(), Deny, tracksOwnCall},
		{"commit with a variable option then switch", "Bash", `git commit "$OPT" -m x && git switch main`, tracked(), Deny, tracksOwnCall},
		// A double-quoted value attached to its option stays one word and
		// cannot become an option or an operand either.
		{"commit with an attached quoted message then switch", "Bash", `git commit -m"$MSG" && git switch main`, tracked(), Allow, ""},
		{"commit with an attached quoted message and -a then switch", "Bash", `git commit -m"$MSG" -a && git switch main`, tracked(), Allow, ""},
		{"commit with a quoted long message then switch", "Bash", `git commit --message="$MSG" -a && git switch main`, tracked(), Allow, ""},
		{"commit with an abbreviated quoted long message then switch", "Bash", `git commit --mess="$MSG" -a && git switch main`, tracked(), Allow, ""},
		{"commit with a quoted message cluster then switch", "Bash", `git commit -am"$MSG" && git switch main`, tracked(), Allow, ""},
		{"commit with a wholly quoted message option then switch", "Bash", `git commit "-m$MSG" -a && git switch main`, tracked(), Allow, ""},
		{"commit with a quoted author then switch", "Bash", `git commit --author="$WHO" -am x && git switch main`, tracked(), Allow, ""},
		// An unquoted attached value may split into an option or an operand.
		{"commit with an unquoted attached message then switch", "Bash", "git commit -m$MSG -a && git switch main", tracked(), Deny, tracksOwnCall},
		{"commit with an unquoted long message then switch", "Bash", "git commit --message=$MSG -a && git switch main", tracked(), Deny, tracksOwnCall},
		// The expansion is not in an option's value.
		{"commit with a quoted expansion after a flag then switch", "Bash", `git commit -a"$X" -m x && git switch main`, tracked(), Deny, tracksOwnCall},
		{"commit with an expanded long option name then switch", "Bash", `git commit --mess"$X"=y -a && git switch main`, tracked(), Deny, tracksOwnCall},
		{"commit with an expanded value-less long option then switch", "Bash", `git commit --amend="$X" -m x && git switch main`, tracked(), Deny, tracksOwnCall},
		{"commit with an expanded short option then switch", "Bash", `git commit -"$O" -m x && git switch main`, tracked(), Deny, tracksOwnCall},
		// The attached value does not swallow the next word: --amend is seen.
		{"amend after an attached quoted message then switch", "Bash", `git commit -m"$MSG" --amend && git switch main`, tracked(), Deny, tracksOwnCall},
		{"amend after a quoted long message then switch", "Bash", `git commit --message="$MSG" --amend && git switch main`, tracked(), Deny, tracksOwnCall},
		// Bash ends an ANSI-C string at a NUL: `-$'\x00m'"$X"` is `-` and
		// the value of X, which may be `-amend`.
		{"commit with a hex NUL before an expanded option then switch", "Bash", `git commit -C HEAD -$'\x00m'"$X" && git switch main`, tracked(), Deny, tracksOwnCall},
		{"commit with an octal NUL before an expanded option then switch", "Bash", `git commit -C HEAD -$'\0m'"$X" && git switch main`, tracked(), Deny, tracksOwnCall},
		{"commit with a message and a NUL before an expanded option then switch", "Bash", `git commit -m x -$'\x00m'"$X" && git switch main`, tracked(), Deny, tracksOwnCall},
		// Brace and glob patterns in an option word may expand into other
		// options (`-{a,-amend}` is `-a --amend`).
		{"commit with a braced expanded option then switch", "Bash", `git commit -{m"$X",-amend} && git switch main`, tracked(), Deny, tracksOwnCall},
		{"commit with a braced option cluster then switch", "Bash", "git commit -m x -{a,-amend} && git switch main", tracked(), Deny, tracksOwnCall},
		{"commit with a braced long option then switch", "Bash", "git commit -m x --{amend,all} && git switch main", tracked(), Deny, tracksOwnCall},
		{"commit with a globbed option then switch", "Bash", "git commit -m x -[a]* && git switch main", tracked(), Deny, tracksOwnCall},
		// -S and -u take an optional value, attached only.
		{"commit with an attached signing key then switch", "Bash", "git commit -Suser@example.com -m x && git switch main", tracked(), Allow, ""},
		{"commit with a bare -S then switch", "Bash", "git commit -S -m x && git switch main", tracked(), Allow, ""},
		{"commit with an untracked-files mode then switch", "Bash", "git commit -uno -am x && git switch main", tracked(), Allow, ""},
		{"commit with a quoted signing key then switch", "Bash", `git commit -S"$KEY" -am x && git switch main`, tracked(), Allow, ""},
		{"amend after a bare -S then switch", "Bash", "git commit -S --amend -m x && git switch main", tracked(), Deny, tracksOwnCall},
		{"add with a braced option then switch", "Bash", "git add -{A,f} && git switch main", tracked(), Deny, tracksOwnCall},
		{"add with a variable then switch", "Bash", `git add "$F" && git switch main`, tracked(), Deny, tracksOwnCall},
		// A branch created at HEAD keeps the commit a later rewrite reads,
		// unless the rewrite names the branch or the previous one.
		{"new branch, commit, switch back", "Bash", "git switch -c feat && git commit -am wip && git switch main", tracked(), Allow, ""},
		{"checkout -b, commit, checkout back", "Bash", "git checkout -q -b feat && git commit -am wip && git checkout main", tracked(), Allow, ""},
		{"new branch then switch to the previous", "Bash", "git switch -c feat && git switch -", tracked(), Deny, tracksOwnCall},
		{"new branch then a checkout of it", "Bash", "git switch -c feat && git checkout feat -- .", tracked(), Deny, tracksOwnCall},
		{"new branch then a reset to it", "Bash", "git checkout -b feat && git reset --hard heads/Feat~0", tracked(), Deny, tracksOwnCall},
		{"tracking new branch then switch", "Bash", "git switch -c feat --track && git switch main", tracked(), Deny, tracksOwnCall},
		{"branch reset then switch", "Bash", "git branch -f feat && git switch main", tracked(), Deny, tracksOwnCall},
		// add --chmod stages a mode for the tracked paths it names.
		{"chmod of a source file then switch", "Bash", "git add --chmod=+x src/x.go && git switch main", tracked(), Allow, ""},
		{"chmod of a directory holding a protected file then checkout", "Bash", "git add --chmod=-x . && git checkout -- .", wholeTree(), Deny,
			"git checkout would overwrite protected .claude/settings.json"},
		// Git applies --chmod to an explicit pathspec only: without one, -A
		// and -u stage the working tree's modes.
		{"chmod with -A and no pathspec then switch", "Bash", "git add -A --chmod=+x && git switch main", tracked(), Allow, ""},
		{"chmod with -u and no pathspec then switch", "Bash", "git add --chmod=+x -u && git switch main", tracked(), Allow, ""},
		{"chmod with --all and no pathspec then switch", "Bash", "git add --all --chmod=-x && git switch main", tracked(), Allow, ""},
		{"chmod with no pathspec then switch", "Bash", "git add --chmod=+x && git switch main", tracked(), Allow, ""},
		{"chmod of everything with -A then switch", "Bash", "git add -A --chmod=+x . && git switch main", wholeTree(), Deny,
			"git switch would overwrite protected .claude/settings.json"},
		// Naming a protected path to git add is itself a protected change.
		{"chmod of a protected file then switch", "Bash", "git add --chmod=-x .claude/settings.json && git switch main", protectedChmod(), Deny,
			"modifies or exfiltrates a protected config path"},
		{"chmod of a protected file after -u then switch", "Bash", "git add -u --chmod=+x -- .claude/settings.json && git switch main", protectedChmod(), Deny,
			"modifies or exfiltrates a protected config path"},
		{"chmod of the tree then switch", "Bash", "git add --chmod=+x . && git switch main", wholeTree(), Deny,
			"git switch would overwrite protected .claude/settings.json"},
		{"chmod from a pathspec file then switch", "Bash", "git add --chmod=+x --pathspec-from-file=list && git switch main", tracked(), Deny, tracksOwnCall},
		// PowerShell commands that only read or print.
		{"Write-Output then checkout in PowerShell", "PowerShell", "Write-Output start; git checkout -- src/x.go", tracked(), Allow, ""},
		{"Get-Content then restore in PowerShell", "PowerShell", "Get-Content go.mod; git restore src/x.go", tracked(), Allow, ""},
		{"Write-Output to a file then checkout in PowerShell", "PowerShell", "Write-Output x > idx; git checkout -- src/x.go", tracked(), Deny, tracksOwnCall},
		{"Set-Content then checkout in PowerShell", "PowerShell", "Set-Content idx x; git checkout -- src/x.go", tracked(), Deny, tracksOwnCall},
		// `cd -` returns to the directory the line's last cd left when that
		// cd surely ran.
		{"cd and back then switch", "Bash", "cd .. && cd - && git switch main", tracked(), Allow, ""},
		{"cd back after a cd that may fail", "Bash", "cd ..; cd - && git switch main", tracked(), Deny, "directory that is not known"},
		{"cd back after a subshell cd", "Bash", "(cd ..) && cd - && git switch main", tracked(), Deny, "directory that is not known"},
		{"cd back with no cd before", "Bash", "cd - && git switch main", tracked(), Deny, "directory that is not known"},
		{"cd back after OLDPWD is set", "Bash", "cd .. && OLDPWD=/tmp && cd - && git switch main", tracked(), Deny, "directory that is not known"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, reason, matches := rewriteVerdict(t, tt.tool, tt.command, top, tt.git)
			if got != tt.want {
				t.Fatalf("%q: verdict = %v (matches %v, SP-007 %q), want %v", tt.command, got, matchIDs(matches), reason, tt.want)
			}
			if tt.want == Deny && !strings.Contains(reason, tt.reason) {
				t.Errorf("%q: SP-007 reason = %q, want it to contain %q", tt.command, reason, tt.reason)
			}
		})
	}
}

// TestMarkedArgs pins that markedArgs marks each expanded word at its first
// expansion, and fails closed when the scan has no literal text for one, or
// when that text already holds expansionMark.
func TestMarkedArgs(t *testing.T) {
	t.Parallel()
	cmd := func(prefixes ...string) scannedCommand {
		var sc scannedCommand
		sc.Args = []string{"commit", "-m", "-a"}
		sc.ExpandedArgIndexes = []int{1}
		sc.ExpandedArgPrefixes = prefixes
		return sc
	}
	tests := []struct {
		name   string
		sc     scannedCommand
		want   []string
		marked bool
	}{
		{"attached value", cmd("-m"), []string{"-m" + expansionMark, "-a"}, true},
		{"no literal text", cmd(), []string{"-m", "-a"}, false},
		{"literal text holding the mark", cmd("-" + expansionMark + "m"), []string{"-m", "-a"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, marked := markedArgs(tt.sc, tt.sc.Args[1:])
			if !slices.Equal(got, tt.want) || marked != tt.marked {
				t.Errorf("markedArgs = %q, %v; want %q, %v", got, marked, tt.want, tt.marked)
			}
		})
	}
}

// TestParseGitOpts_Unmodelled pins the option words whose names hold a
// character git's options never do, which the shell may expand into other
// options.
func TestParseGitOpts_Unmodelled(t *testing.T) {
	t.Parallel()
	for args, want := range map[string]bool{
		"-am x": false, "-m{x,y}": false, "--message={a,b}": false, "-C HEAD -1": false, "--no-verify": false,
		"-{a,-amend}": true, "-a{m,-amend}": true, "--{amend,all}": true, "-[a]*": true, "--am*": true,
		"-m x -" + expansionMark:  true,
		"-Suser@example.com -m x": false, "-S -m x": false, "-uno": false, "-qS{a,b}": false,
	} {
		if got := parseGitOpts(strings.Fields(args), commitSpec).unmodelled; got != want {
			t.Errorf("parseGitOpts(%q).unmodelled = %v, want %v", args, got, want)
		}
	}
}

// TestMovesWithCommit pins the revisions a commit before a rewrite may
// redirect to another commit.
func TestMovesWithCommit(t *testing.T) {
	t.Parallel()
	for rev, want := range map[string]bool{
		"HEAD": false, "main": false, "evil": false, "@{-1}": false, "@{u}": false, "main@{upstream}": false,
		"@{push}": false, "origin/main": false, "a..b": false,
		"HEAD~1": true, "main^": true, "HEAD^{tree}": true, ":/fix": true, "HEAD:x": true, "@{1}": true,
		"HEAD@{yesterday}": true, "main@{u}@{2}": true,
	} {
		if got := movesWithCommit(rev); got != want {
			t.Errorf("movesWithCommit(%q) = %v, want %v", rev, got, want)
		}
	}
}

// TestWorktreeRewrite_CallCount pins the cost of the rule on every OS: the
// file tools and shell commands that do not rewrite the working tree spawn no
// git, and a rewrite spawns at most two (the toplevel and one diff, stash
// show or, after another command, ls-files), or three for a checkout operand
// that names no revision (the diff as a revision, then as a pathspec), for
// a stash that keeps the index (the diffs against HEAD and the index), for a
// rebase of a branch operand (the switch, then the replay) and for a patch
// (read forward and in reverse). A rewrite denied outright asks nothing.
func TestWorktreeRewrite_CallCount(t *testing.T) {
	t.Parallel()
	top := filepath.Join(homeDir(t), "project")
	file := filepath.Join(top, "src", "x.go")
	tests := []struct {
		name string
		ctx  EvalContext
		max  int
	}{
		{"Read", EvalContext{ToolName: "Read", FilePath: file, CanonicalPath: file, CWD: top}, 0},
		{"Grep", EvalContext{ToolName: "Grep", FilePath: file, CanonicalPath: file, CWD: top}, 0},
		{"Glob", EvalContext{ToolName: "Glob", FilePath: top, CanonicalPath: top, CWD: top}, 0},
		{"Write", EvalContext{ToolName: "Write", FilePath: file, CanonicalPath: file, Content: "package x\n", CWD: top}, 0},
		{"git status", EvalContext{ToolName: "Bash", Command: "git status", CWD: top}, 0},
		{"git commit", EvalContext{ToolName: "Bash", Command: "git commit -m x", CWD: top}, 0},
		{"rm", EvalContext{ToolName: "Bash", Command: "rm x", CWD: top}, 0},
		{"new branch", EvalContext{ToolName: "Bash", Command: "git checkout -b feature", CWD: top}, 0},
		{"restore --staged", EvalContext{ToolName: "Bash", Command: "git restore --staged x", CWD: top}, 0},
		{"powershell git status", EvalContext{ToolName: "PowerShell", Command: "git status", CWD: top}, 0},
		{"reset --hard", EvalContext{ToolName: "Bash", Command: "git reset --hard evil", CWD: top}, 2},
		{"stash pop", EvalContext{ToolName: "Bash", Command: "git stash pop", CWD: top}, 2},
		{"stash push", EvalContext{ToolName: "Bash", Command: "git stash", CWD: top}, 2},
		{"stash keeping the index", EvalContext{ToolName: "Bash", Command: "git stash -k", CWD: top}, 3},
		{"stash list", EvalContext{ToolName: "Bash", Command: "git stash list", CWD: top}, 0},
		{"switch", EvalContext{ToolName: "Bash", Command: "git switch evil", CWD: top}, 2},
		{"checkout-index", EvalContext{ToolName: "Monitor", Command: "git checkout-index -a -f", CWD: top}, 2},
		{"powershell reset --hard", EvalContext{ToolName: "PowerShell", Command: "git reset --hard evil", CWD: top}, 2},
		{"preceded rewrite", EvalContext{ToolName: "Bash", Command: "git fetch o && git reset --hard FETCH_HEAD", CWD: top}, 2},
		{"ambiguous operand", EvalContext{ToolName: "Bash", Command: "git checkout pkg", CWD: top}, 3},
		{"two rewrites", EvalContext{ToolName: "Bash", Command: "git switch main && git reset --hard evil", CWD: top}, 4},
		{"merge", EvalContext{ToolName: "Bash", Command: "git merge evil", CWD: top}, 2},
		{"cherry-pick", EvalContext{ToolName: "Bash", Command: "git cherry-pick evil", CWD: top}, 2},
		{"rebase of a branch operand", EvalContext{ToolName: "Bash", Command: "git rebase main feature", CWD: top}, 3},
		{"apply", EvalContext{ToolName: "Bash", Command: "git apply x.patch", CWD: top}, 3},
		{"merge --continue", EvalContext{ToolName: "Bash", Command: "git merge --continue", CWD: top}, 0},
		{"apply from standard input", EvalContext{ToolName: "Bash", Command: "git apply < x.patch", CWD: top}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			git := rewriteRepo(top, false)
			ctx := tt.ctx
			ctx.Ctx, ctx.gitRunner = t.Context(), git.run
			Tier1Rules.EvaluateAll(&ctx)
			n := git.callCount()
			if n > tt.max || (tt.max == 0 && n != 0) {
				t.Errorf("git spawned %d times, want at most %d: %v", n, tt.max, git.calls)
			}
			if tt.max > 0 && n == 0 {
				t.Errorf("a rewrite spawned no git; it was not checked")
			}
		})
	}
}

// sameRewrite compares two rewrites; for an unchecked one only that it is
// unchecked matters, not the reason's wording.
func sameRewrite(a, b worktreeRewrite) bool {
	if a.unchecked != "" || b.unchecked != "" {
		return a.unchecked != "" && b.unchecked != ""
	}
	return a.source == b.source && a.rev == b.rev && a.base == b.base && a.strip == b.strip && a.remote == b.remote &&
		a.directory == b.directory && slices.Equal(a.revs, b.revs) && slices.Equal(a.patches, b.patches) &&
		slices.Equal(a.paths, b.paths) && slices.Equal(a.orPaths, b.orPaths) && slices.EqualFunc(a.also, b.also, sameRewrite)
}

// TestParseWorktreeRewrite pins the comparison each recognised git form
// yields, and that the forms which leave the working tree alone yield none.
func TestParseWorktreeRewrite(t *testing.T) {
	t.Parallel()
	idx := func(paths ...string) worktreeRewrite { return worktreeRewrite{source: fromIndex, paths: paths} }
	rev := func(r string, paths ...string) worktreeRewrite {
		return worktreeRewrite{source: fromRev, rev: r, paths: paths}
	}
	between := func(r string) worktreeRewrite { return worktreeRewrite{source: betweenRevs, rev: r} }
	stash := func(r string) worktreeRewrite { return worktreeRewrite{source: fromStash, rev: r} }
	// or adds the pathspec reading of an ambiguous checkout operand list.
	or := func(rw worktreeRewrite, paths ...string) worktreeRewrite { rw.orPaths = paths; return rw }
	// keep adds the index comparison of a stash that keeps the index.
	keep := func(rw worktreeRewrite) worktreeRewrite { rw.also = []worktreeRewrite{idx(rw.paths...)}; return rw }
	merge := func(revs ...string) worktreeRewrite { return worktreeRewrite{source: mergeBase, revs: revs} }
	commits := func(revs ...string) worktreeRewrite { return worktreeRewrite{source: fromCommits, revs: revs} }
	patches := func(files ...string) worktreeRewrite { return worktreeRewrite{source: fromPatches, patches: files} }
	// from switches to branch first, then compares from it.
	from := func(branch string, rw worktreeRewrite) worktreeRewrite {
		rw.base, rw.also = branch, []worktreeRewrite{between(branch)}
		return rw
	}
	// unchecked is a rewrite denied outright.
	unchecked := worktreeRewrite{unchecked: "any"}

	rm := func(paths ...string) worktreeRewrite { return worktreeRewrite{source: removesTracked, paths: paths} }

	tests := []struct {
		command string
		want    worktreeRewrite
		ok      bool
	}{
		// rm deletes the tracked paths its pathspec matches, unless it only
		// untracks them or is a dry run.
		{"rm -r .", rm("."), true},
		{"rm -rf -- . src", rm(".", "src"), true},
		{"rm -q --ignore-unmatch x", rm("x"), true},
		{"rm --pathspec-from-file=list", rm(":/"), true},
		{"rm --pathspec-from-file list -r", rm(":/"), true},
		{"rm --cached -r .", worktreeRewrite{}, false},
		{"rm --cac -r .", worktreeRewrite{}, false},
		{"rm -rn .", worktreeRewrite{}, false},
		{"rm --dry-run -r .", worktreeRewrite{}, false},
		{"rm -r", worktreeRewrite{}, false},
		// checkout-index
		{"checkout-index -a -f", idx(), true},
		{"checkout-index -af", idx(), true},
		{"checkout-index --all", idx(), true},
		{"checkout-index -f -- a b", idx("a", "b"), true},
		{"checkout-index --stdin", idx(), true},
		{"checkout-index", worktreeRewrite{}, false},
		// checkout
		{"checkout -- src/x.go", idx("src/x.go"), true},
		{"checkout evil -- .", rev("evil", "."), true},
		{"checkout evil .claude", or(rev("evil", ".claude"), "evil", ".claude"), true},
		{"checkout .", idx("."), true},
		{"checkout -f", rev("HEAD"), true},
		{"checkout -f evil", or(rev("evil"), "evil"), true},
		{"checkout --force evil", or(rev("evil"), "evil"), true},
		{"checkout evil", or(between("evil"), "evil"), true},
		{"checkout --detach evil", or(between("evil"), "evil"), true},
		// An operand that names no revision is read as a pathspec.
		{"checkout :/", or(between(":/"), ":/"), true},
		{"checkout :(top)", or(between(":(top)"), ":(top)"), true},
		{"checkout :!x", or(between(":!x"), ":!x"), true},
		{"checkout src/..", or(between("src/.."), "src/.."), true},
		{"checkout pkg", or(between("pkg"), "pkg"), true},
		// "-" is the previous branch.
		{"checkout -", or(between("@{-1}"), "@{-1}"), true},
		{"checkout -f -", or(rev("@{-1}"), "@{-1}"), true},
		{"checkout -b feature -", between("@{-1}"), true},
		{"switch -", between("@{-1}"), true},
		{"switch -f -", rev("@{-1}"), true},
		{"checkout -b feature evil", between("evil"), true},
		{"checkout -bfeature evil", between("evil"), true},
		{"checkout -B feature -f evil", rev("evil"), true},
		{"checkout --pathspec-from-file=list evil", rev("evil"), true},
		{"checkout -b feature", worktreeRewrite{}, false},
		{"checkout --orphan feature", worktreeRewrite{}, false},
		// A forced new branch with no start point resets the work tree to
		// HEAD, as `checkout -f` does.
		{"checkout -f -b feature", rev("HEAD"), true},
		{"checkout -fB feature", rev("HEAD"), true},
		{"checkout --force --orphan feature", rev("HEAD"), true},
		// An unborn branch with a start point moves the work tree to it.
		{"checkout --orphan x evil", between("evil"), true},
		{"checkout -q --orphan=x evil", between("evil"), true},
		{"checkout --orphan x -f evil", rev("evil"), true},
		// --patch copies hunks of the index, or of a tree-ish.
		{"checkout -p", idx(), true},
		{"checkout --patch", idx(), true},
		{"checkout --patc", idx(), true},
		{"checkout -p evil", or(rev("evil"), "evil"), true},
		{"checkout -p evil .claude", or(rev("evil", ".claude"), "evil", ".claude"), true},
		{"checkout -p evil -- .", rev("evil", "."), true},
		{"checkout", worktreeRewrite{}, false},
		// switch
		{"switch evil", between("evil"), true},
		{"switch -f evil", rev("evil"), true},
		{"switch --discard-changes evil", rev("evil"), true},
		{"switch -c feature evil", between("evil"), true},
		{"switch --create=feature evil", between("evil"), true},
		{"switch --detach evil", between("evil"), true},
		{"switch --orphan feature", between(emptyTree), true},
		{"switch -c feature", worktreeRewrite{}, false},
		// restore
		{"restore x", idx("x"), true},
		{"restore --source evil .", rev("evil", "."), true},
		{"restore --source=evil .", rev("evil", "."), true},
		{"restore -s evil .", rev("evil", "."), true},
		{"restore -sevil .", rev("evil", "."), true},
		{"restore -W -- x", idx("x"), true},
		{"restore --staged --worktree x", rev("HEAD", "x"), true},
		{"restore -SW x", rev("HEAD", "x"), true},
		{"restore --staged x", worktreeRewrite{}, false},
		{"restore -S x", worktreeRewrite{}, false},
		// --patch needs no pathspec: it offers hunks of the whole tree.
		{"restore -p", idx(), true},
		{"restore -p -s evil", rev("evil"), true},
		{"restore --patch --source=evil", rev("evil"), true},
		{"restore", worktreeRewrite{}, false},
		{"restore -s evil", worktreeRewrite{}, false},
		// reset
		{"reset --hard", rev("HEAD"), true},
		{"reset --hard evil", rev("evil"), true},
		{"reset --keep evil", rev("evil"), true},
		{"reset --merge evil", rev("evil"), true},
		{"reset -q --hard evil", rev("evil"), true},
		{"reset --soft HEAD~1", worktreeRewrite{}, false},
		{"reset --mixed HEAD~1", worktreeRewrite{}, false},
		{"reset HEAD~1", worktreeRewrite{}, false},
		{"reset -- x", worktreeRewrite{}, false},
		// read-tree
		{"read-tree -u -m evil", rev("evil"), true},
		{"read-tree -mu HEAD evil", rev("evil"), true},
		{"read-tree --reset -u evil", rev("evil"), true},
		{"read-tree -u --empty", rev(emptyTree), true},
		{"read-tree -m evil", worktreeRewrite{}, false},
		{"read-tree -u -n evil", worktreeRewrite{}, false},
		// stash
		{"stash pop", stash("stash@{0}"), true},
		{"stash apply", stash("stash@{0}"), true},
		{"stash pop --index stash@{2}", stash("stash@{2}"), true},
		{"stash apply -q 1", stash("1"), true},
		{"stash branch b", stash("stash@{0}"), true},
		{"stash branch b stash@{1}", stash("stash@{1}"), true},
		// push resets the work tree to HEAD, and with a kept index then to
		// the index.
		{"stash", rev("HEAD"), true},
		{"stash push -m x", rev("HEAD"), true},
		{"stash push -m x -- src", rev("HEAD", "src"), true},
		{"stash push src", rev("HEAD", "src"), true},
		{"stash save wip", rev("HEAD"), true},
		{"stash -u", rev("HEAD"), true},
		{"stash -k", keep(rev("HEAD")), true},
		{"stash --keep-index -u", keep(rev("HEAD")), true},
		{"stash push --staged", keep(rev("HEAD")), true},
		{"stash push -p", keep(rev("HEAD")), true},
		{"stash push -p --no-keep-index", rev("HEAD"), true},
		{"stash -- .claude", rev("HEAD", ".claude"), true},
		{"stash list", worktreeRewrite{}, false},
		{"stash clear", worktreeRewrite{}, false},
		{"stash show -p", worktreeRewrite{}, false},
		{"stash drop", worktreeRewrite{}, false},
		// merge writes what each revision changed since its merge base.
		{"merge evil", merge("evil"), true},
		{"merge --ff-only evil", merge("evil"), true},
		{"merge --no-commit --squash evil", merge("evil"), true},
		{"merge -m msg -s ort -X theirs evil", merge("evil"), true},
		{"merge --strategy-option=theirs a evil", merge("a", "evil"), true},
		{"merge -", merge("@{-1}"), true},
		{"merge", merge("@{upstream}"), true},
		// "--" ends the options; the revisions follow it.
		{"merge -- evil", merge("evil"), true},
		{"merge --no-ff a -- evil", merge("a", "evil"), true},
		{"merge -- -", merge("@{-1}"), true},
		{"cherry-pick -- evil", commits("evil"), true},
		{"revert --no-edit -- a evil", commits("a", "evil"), true},
		{"rebase -- evil", merge("evil"), true},
		{"rebase -- main feature", from("feature", merge("main")), true},
		{"pull . -- evil", merge("evil"), true},
		{"pull -- . evil", merge("evil"), true},
		// A path to the repository may be the repository itself.
		{"pull ./ evil", worktreeRewrite{source: mergeBase, revs: []string{"evil"}, remote: "./"}, true},
		{"pull /r evil", worktreeRewrite{source: mergeBase, revs: []string{"evil"}, remote: "/r"}, true},
		{"pull ../r", worktreeRewrite{source: mergeBase, revs: []string{"@{upstream}"}, remote: "../r"}, true},
		{"pull file:///r/.git evil", worktreeRewrite{source: mergeBase, revs: []string{"evil"}, remote: "file:///r/.git"}, true},
		{"pull https://example.com/r evil", worktreeRewrite{}, false},
		{"pull host:r evil", worktreeRewrite{}, false},
		{"merge --abort", worktreeRewrite{}, false},
		{"merge --continue", worktreeRewrite{}, false},
		{"merge --quit", worktreeRewrite{}, false},
		// pull from the repository itself merges (or rebases onto) its refs;
		// from a remote it is a residual.
		{"pull . evil", merge("evil"), true},
		{"pull --rebase . evil", merge("evil"), true},
		{"pull --depth 1 . +evil:tmp", merge("evil"), true},
		{"pull .", merge("@{upstream}"), true},
		{"pull --rebase=interactive . evil", unchecked, true},
		{"pull", worktreeRewrite{}, false},
		{"pull origin main", worktreeRewrite{}, false},
		{"pull --rebase origin evil", worktreeRewrite{}, false},
		// rebase
		{"rebase evil", merge("evil"), true},
		{"rebase", merge("@{upstream}"), true},
		{"rebase -s ort -X theirs evil", merge("evil"), true},
		{"rebase --onto evil main", between("evil"), true},
		{"rebase --onto=evil main feature", from("feature", between("evil")), true},
		{"rebase main feature", from("feature", merge("main")), true},
		{"rebase -i main", unchecked, true},
		{"rebase --interactive main", unchecked, true},
		{"rebase --root", unchecked, true},
		{"rebase -x true main", unchecked, true},
		{"rebase --exec=true main", unchecked, true},
		{"rebase --continue", worktreeRewrite{}, false},
		{"rebase --skip", worktreeRewrite{}, false},
		{"rebase --abort", worktreeRewrite{}, false},
		{"rebase --quit", worktreeRewrite{}, false},
		{"rebase --edit-todo", worktreeRewrite{}, false},
		// cherry-pick and revert replay the commits' changes.
		{"cherry-pick evil", commits("evil"), true},
		{"cherry-pick -x -m 1 evil", commits("evil"), true},
		{"cherry-pick main..evil", commits("main..evil"), true},
		{"cherry-pick a b", commits("a", "b"), true},
		{"cherry-pick -", commits("@{-1}"), true},
		{"revert --no-edit evil", commits("evil"), true},
		{"cherry-pick --stdin", unchecked, true},
		{"cherry-pick --all", unchecked, true},
		{"cherry-pick --branches=evil", unchecked, true},
		{"cherry-pick --continue", worktreeRewrite{}, false},
		{"cherry-pick --skip", worktreeRewrite{}, false},
		{"revert --abort", worktreeRewrite{}, false},
		{"cherry-pick", worktreeRewrite{}, false},
		// apply and am apply the patch files' paths; standard input cannot
		// be read in advance.
		{"apply x.patch", patches("x.patch"), true},
		{"apply --index -3 x.patch y.patch", patches("x.patch", "y.patch"), true},
		{"apply -p0 --directory=sub x.patch", worktreeRewrite{source: fromPatches, patches: []string{"x.patch"}, strip: "0", directory: "sub"}, true},
		{"apply -p 2 x.patch", worktreeRewrite{source: fromPatches, patches: []string{"x.patch"}, strip: "2"}, true},
		{"apply --stat --apply x.patch", patches("x.patch"), true},
		{"am x.mbox", patches("x.mbox"), true},
		{"am -3 --directory sub x.mbox", worktreeRewrite{source: fromPatches, patches: []string{"x.mbox"}, directory: "sub"}, true},
		{"apply", unchecked, true},
		{"apply -", unchecked, true},
		{"am", unchecked, true},
		{"am -3", unchecked, true},
		{"apply --check x.patch", worktreeRewrite{}, false},
		{"apply --stat x.patch", worktreeRewrite{}, false},
		{"apply --numstat x.patch", worktreeRewrite{}, false},
		{"apply --summary x.patch", worktreeRewrite{}, false},
		{"apply --cached x.patch", worktreeRewrite{}, false},
		{"am --continue", worktreeRewrite{}, false},
		{"am -r", worktreeRewrite{}, false},
		{"am --skip", worktreeRewrite{}, false},
		{"am --abort", worktreeRewrite{}, false},
		{"am --quit", worktreeRewrite{}, false},
		{"am --show-current-patch=diff", worktreeRewrite{}, false},
		// not rewrites
		{"status", worktreeRewrite{}, false},
		{"commit -m x", worktreeRewrite{}, false},
		{"diff evil", worktreeRewrite{}, false},
		{"fetch origin", worktreeRewrite{}, false},
		{"log evil", worktreeRewrite{}, false},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			words := strings.Fields(tt.command)
			got, ok := parseWorktreeRewrite(words[0], words[1:])
			if ok != tt.ok {
				t.Fatalf("parseWorktreeRewrite(%q) ok = %v, want %v (got %+v)", tt.command, ok, tt.ok, got)
			}
			if ok && !sameRewrite(got, tt.want) {
				t.Errorf("parseWorktreeRewrite(%q) = %+v, want %+v", tt.command, got, tt.want)
			}
		})
	}

	for _, command := range []string{"checkout-index -a --prefix=out/", "read-tree -u --prefix=sub/ evil"} {
		t.Run(command, func(t *testing.T) {
			t.Parallel()
			words := strings.Fields(command)
			if got, ok := parseWorktreeRewrite(words[0], words[1:]); !ok || got.unchecked == "" {
				t.Errorf("parseWorktreeRewrite(%q) = %+v, %v; want an unchecked rewrite", command, got, ok)
			}
		})
	}
}

// TestWorktreeRewrite_MergeFamily pins W1bN-07: merge, pull from ".", rebase,
// cherry-pick, revert, apply and am write the tracked .claude/settings.json
// without naming it. Each is denied in the evil repository (where what it
// merges, replays or patches changes the file) and allowed in the clean one;
// one whose input cannot be read in advance is denied in both.
func TestWorktreeRewrite_MergeFamily(t *testing.T) {
	t.Parallel()
	top := filepath.Join(homeDir(t), "project")
	const overwrite = "would overwrite protected .claude/settings.json in the working tree"

	repoRows := []string{
		"git merge evil",
		"git merge --ff-only evil",
		"git merge --no-commit evil",
		"git merge a evil",
		"git cherry-pick evil",
		"git cherry-pick main..evil",
		"git revert --no-edit evil",
		"git rebase evil",
		"git rebase --onto evil main",
		"git rebase main evil", // switches to evil first
		"git pull . evil",
		"git pull --rebase . evil",
		// After "--" the words are still revisions.
		"git merge -- evil",
		"git merge --no-ff -- evil",
		"git cherry-pick -- evil",
		"git revert -- evil",
		"git rebase -- evil",
		"git pull . -- evil",
		// The repository itself spelled as a path is remote ".".
		"git pull --no-rebase ./ evil",
		"git pull " + top + " evil",
		"git pull file://" + filepath.ToSlash(top) + " evil",
		// No merge base: git merges (or rebases onto) the unrelated tree.
		"git merge --allow-unrelated-histories orphan",
		"git merge --allow-unrelated-histories -X theirs orphan",
		"git rebase orphan",
		"git rebase -X ours orphan",
		"git apply x.patch",
		"git apply -p0 --directory=sub x.patch",
		"git am x.mbox",
		"git -C . merge evil",
		"sh -c 'git merge evil'",
		"git status && git cherry-pick evil",
	}
	for _, command := range repoRows {
		for _, evil := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/evil=%v", command, evil), func(t *testing.T) {
				t.Parallel()
				got, reason, matches := rewriteVerdict(t, "Bash", command, top, rewriteRepo(top, evil))
				if evil {
					if got != Deny || !strings.Contains(reason, overwrite) {
						t.Errorf("verdict = %v, SP-007 reason %q (matches %v), want Deny with %q", got, reason, matchIDs(matches), overwrite)
					}
				} else if got != Allow {
					t.Errorf("verdict = %v (matches %v, SP-007 %q), want Allow", got, matchIDs(matches), reason)
				}
			})
		}
	}

	// What these write cannot be known in advance: denied in both
	// repositories, without asking git.
	uncheckedRows := []struct{ command, reason string }{
		{"git diff main evil | git apply", "save the patch to a file"},
		{"git format-patch -1 evil --stdout | git am", "save the patch to a file"},
		{"git apply - < x.patch", "standard input"},
		{"git rebase -i main", "--interactive"},
		{"git rebase --root", "--root"},
		{"git cherry-pick --stdin < revs", "--stdin"},
		{"git pull --rebase=interactive . evil", "--interactive"},
		// git apply cannot read it (a Maildir, a missing file).
		{"git am mails/", "cannot read"},
		// An earlier command may write the patch.
		{"cp /tmp/p x.patch && git apply x.patch", "may change the patch"},
	}
	for _, row := range uncheckedRows {
		for _, evil := range []bool{true, false} {
			t.Run(fmt.Sprintf("%s/evil=%v", row.command, evil), func(t *testing.T) {
				t.Parallel()
				git := rewriteRepo(top, evil)
				got, reason, matches := rewriteVerdict(t, "Bash", row.command, top, git)
				if got != Deny || !strings.Contains(reason, row.reason) {
					t.Errorf("verdict = %v, SP-007 reason %q (matches %v), want Deny with %q", got, reason, matchIDs(matches), row.reason)
				}
			})
		}
	}

	rows := []struct {
		name    string
		tool    string
		command string
		git     *fakeGit
		want    Verdict
		reason  string
	}{
		// Only HEAD's side changed the settings file since feature forked:
		// the two-dot diff names it, but merging feature writes only what
		// feature changed.
		{"three-dot merge", "Bash", "git merge feature", threeDotRepo(top), Allow, ""},
		{"three-dot rebase", "Bash", "git rebase feature", threeDotRepo(top), Allow, ""},
		{"three-dot pull", "Bash", "git pull . feature", threeDotRepo(top), Allow, ""},
		{"clean merge of feature", "Bash", "git merge feature", rewriteRepo(top, false), Allow, ""},
		{"clean cherry-pick of feature", "Bash", "git cherry-pick feature", rewriteRepo(top, false), Allow, ""},
		{"clean rebase onto main", "Bash", "git rebase main", rewriteRepo(top, false), Allow, ""},
		{"clean patch", "Bash", "git apply fix.patch", rewriteRepo(top, true), Allow, ""},
		{"check then apply", "Bash", "git apply --check fix.patch && git apply fix.patch", rewriteRepo(top, true), Allow, ""},
		// git apply patches files outside a repository too.
		{"patch outside a repository", "Bash", "git apply x.patch", &fakeGit{answers: rewriteRepo(top, true).answers}, Deny, "would overwrite protected .claude/settings.json"},
		{"unknown merge revision", "Bash", "git merge nosuch", rewriteRepo(top, true), Allow, ""},
		{"unknown merge revision after --", "Bash", "git merge -- nosuch", rewriteRepo(top, true), Allow, ""},
		// git merge FETCH_HEAD merges every head recorded for merging, the
		// documented replacement for a pull from a remote.
		{"FETCH_HEAD of evil", "Bash", "git merge FETCH_HEAD",
			withFetchHead(t, rewriteRepo(top, true), evilOID+"\t\tbranch 'evil' of ../r\n"), Deny, overwrite},
		{"FETCH_HEAD with evil second", "Bash", "git merge FETCH_HEAD",
			withFetchHead(t, rewriteRepo(top, true), featureOID+"\t\tbranch 'feature' of .\n"+evilOID+"\t\tbranch 'evil' of .\n"), Deny, overwrite},
		{"FETCH_HEAD rebase", "Bash", "git rebase FETCH_HEAD",
			withFetchHead(t, rewriteRepo(top, true), featureOID+"\t\tbranch 'feature' of .\n"+evilOID+"\t\tbranch 'evil' of .\n"), Deny, overwrite},
		{"FETCH_HEAD with evil not for merge", "Bash", "git merge FETCH_HEAD",
			withFetchHead(t, rewriteRepo(top, true), featureOID+"\t\tbranch 'feature' of .\n"+evilOID+"\tnot-for-merge\tbranch 'evil' of .\n"), Allow, ""},
		{"no FETCH_HEAD", "Bash", "git merge FETCH_HEAD", rewriteRepo(top, true), Allow, ""},
		{"malformed FETCH_HEAD", "Bash", "git merge FETCH_HEAD", withFetchHead(t, rewriteRepo(top, true), "not a line\n"), Allow, ""},
		// A pull from another repository fetches first: the residual.
		{"pull from another path", "Bash", "git pull ../other evil", rewriteRepo(top, true), Allow, ""},
		{"pull from a computed path", "Bash", `git pull --no-rebase "$PWD" evil`, rewriteRepo(top, true), Deny, "shell expansion"},
		// A program the scan cannot name may be git.
		{"computed program", "Bash", "g=$(command -v git); $g reset --hard evil", rewriteRepo(top, false), Deny, "named through a shell expansion"},
		{"computed program merge", "Bash", `"$G" merge evil`, rewriteRepo(top, false), Deny, "named through a shell expansion"},
		{"computed program read-only", "Bash", "g=$(command -v git); $g log evil", rewriteRepo(top, true), Allow, ""},
		{"lookup with stderr discarded", "Bash", "$(which git 2>/dev/null) reset --hard evil", rewriteRepo(top, true), Deny, "named through a shell expansion"},
		{"lookup after --", "Bash", "$(command -v -- git) reset --hard evil", rewriteRepo(top, true), Deny, "named through a shell expansion"},
		{"powershell merge", "PowerShell", "git merge evil", rewriteRepo(top, true), Deny, overwrite},
		{"powershell apply", "PowerShell", "git apply x.patch", rewriteRepo(top, true), Deny, overwrite},
		{"monitor cherry-pick", "Monitor", "git cherry-pick evil", rewriteRepo(top, true), Deny, overwrite},
	}
	for _, tt := range rows {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, reason, matches := rewriteVerdict(t, tt.tool, tt.command, top, tt.git)
			if got != tt.want {
				t.Fatalf("%q: verdict = %v (matches %v, SP-007 %q), want %v", tt.command, got, matchIDs(matches), reason, tt.want)
			}
			if tt.want == Deny && !strings.Contains(reason, tt.reason) {
				t.Errorf("%q: SP-007 reason = %q, want it to contain %q", tt.command, reason, tt.reason)
			}
		})
	}
}

// threeDotRepo is the evil rewriteRepo where HEAD changed the settings file
// after branch feature forked: the two-dot diff to feature names the file,
// the merge-base diff does not.
func threeDotRepo(top string) *fakeGit {
	git := rewriteRepo(top, true)
	git.answers["diff HEAD feature --"] = git.answers["diff --"]
	git.answers["diff HEAD..feature --"] = git.answers["diff --"]
	return git
}

// TestWorktreeRewrite_MergeFamilyStillAllowed pins the merge-family forms
// that start nothing: continuing, skipping, quitting or aborting an operation
// in progress, apply's listing and index-only modes, a pull or fetch from a
// remote and a log. They are allowed in the evil repository and, as on base,
// ask git nothing.
func TestWorktreeRewrite_MergeFamilyStillAllowed(t *testing.T) {
	t.Parallel()
	top := filepath.Join(homeDir(t), "project")
	for _, command := range []string{
		"git merge --abort",
		"git merge --continue",
		"git merge --quit",
		"git rebase --continue",
		"git rebase --skip",
		"git rebase --abort",
		"git rebase --quit",
		"git cherry-pick --continue",
		"git cherry-pick --skip",
		"git cherry-pick --abort",
		"git revert --quit",
		"git am --continue",
		"git am --skip",
		"git am --abort",
		"git am --quit",
		"git apply --check fix.patch",
		"git apply --stat fix.patch",
		"git apply --numstat x.patch",
		"git apply --cached x.patch",
		"git pull",
		"git pull origin main",
		"git pull --rebase origin main",
		"git fetch origin",
		"git log evil",
	} {
		t.Run(command, func(t *testing.T) {
			t.Parallel()
			git := rewriteRepo(top, true)
			got, reason, matches := rewriteVerdict(t, "Bash", command, top, git)
			if got != Allow {
				t.Errorf("verdict = %v (matches %v, SP-007 %q), want Allow", got, matchIDs(matches), reason)
			}
			if n := git.callCount(); n != 0 {
				t.Errorf("git spawned %d times, want none: %v", n, git.calls)
			}
		})
	}
}

// cleanRepo is a rewriteRepo whose git clean --dry-run lists what each
// option set would remove. In the evil one the index no longer tracks
// .claude (after `git rm -r --cached .` or `git read-tree --empty`), so a
// clean removes it; in the other only build output is untracked. Both track
// what rewriteRepo does.
func cleanRepo(top string, evil bool) *fakeGit {
	git := rewriteRepo(top, evil)
	dirs, ignored := "Would remove build/\n", "Would remove build/\nWould remove x.log\n"
	if evil {
		dirs = "Would remove .claude/\nWould remove build/\n"
		ignored = "Would remove .claude/settings.local.json\n"
	}
	git.answers["clean -f --"] = "Would remove build.out\n"
	git.answers["clean -d -f --"] = dirs
	git.answers["clean -d -f -f --"] = dirs + "Would remove nested/\n"
	git.answers["clean -d -x -f --"] = dirs + ignored
	git.answers["clean -x -f --"] = ignored
	git.answers["clean -d -f -- src"] = ""
	git.answers["clean -d -f --exclude=keep --"] = "Would remove build/\n"
	// An exclude that matches .claude keeps it; one that does not (or
	// only negates) leaves the listing as it was.
	git.answers["clean -d -x -f --exclude=.claude --"] = "Would remove build/\nWould remove x.log\n"
	git.answers["clean -d -x -f --exclude=src --"] = dirs + ignored
	git.answers["clean -d -x -f --exclude=!.claude --"] = dirs + ignored
	git.answers["clean -d -x -f -- .claude"] = "Would remove .claude/\n"
	return git
}

// TestWorktreeRewrite_CleanAndSparse pins the two ways git drops a protected
// file from the working tree without rewriting it: git clean (and stash -u /
// --all, which clean what they save) removing a protected path the index
// does not track, judged by git clean --dry-run, also after another command
// on the line; and a sparse-checkout pattern change, denied in a repository
// that tracks a protected path. Listing, dry runs, disable and an add of
// patterns that only include more ask git nothing.
func TestWorktreeRewrite_CleanAndSparse(t *testing.T) {
	t.Parallel()
	top := filepath.Join(homeDir(t), "project")
	const removed = "would remove protected .claude/"
	const sparse = "a person must change the sparse-checkout patterns"
	const ownCall = "run it in its own call"
	untrackedOnly := func(top string) *fakeGit {
		git := cleanRepo(top, true)
		git.tracked = "src/x.go\x00" // .claude was never tracked
		return git
	}
	rows := []struct {
		name, command string
		git           *fakeGit
		want          Verdict
		reason        string
	}{
		{"clean of an untracked .claude", "git clean -fd", cleanRepo(top, true), Deny, removed},
		{"clean with ignored files", "git clean -fdx", cleanRepo(top, true), Deny, removed},
		{"clean of ignored settings", "git clean -f -x", cleanRepo(top, true), Deny, "would remove protected .claude/settings.local.json"},
		{"clean, long options", "git clean --force -d", cleanRepo(top, true), Deny, removed},
		{"clean of nested repositories", "git clean -ffd", cleanRepo(top, true), Deny, removed},
		{"clean without -f", "git clean -d", cleanRepo(top, true), Deny, removed},
		{"dry run turned off", "git clean -n --no-dry-run -fd", cleanRepo(top, true), Deny, removed},
		{"clean in a script", "sh -c 'git clean -fd'", cleanRepo(top, true), Deny, removed},
		{"after untracking on the line", "git rm -r -q --cached . && git clean -fd", cleanRepo(top, true), Deny, ownCall},
		{"after read-tree on the line", "git read-tree --empty; git clean -fdq", cleanRepo(top, true), Deny, ownCall},
		{"preceded, never tracked", "true && git clean -fd", untrackedOnly(top), Deny, removed},
		{"stash with untracked files", "git stash -u", cleanRepo(top, true), Deny, removed},
		{"stash with ignored files", "git stash push --all", cleanRepo(top, true), Deny, removed},
		{"powershell clean", "PowerShell", cleanRepo(top, true), Deny, removed},
		{"clean of build output", "git clean -fd", cleanRepo(top, false), Allow, ""},
		{"clean with ignored build output", "git clean -fdx", cleanRepo(top, false), Allow, ""},
		{"clean of a subdirectory", "git clean -fd -- src", cleanRepo(top, true), Allow, ""},
		{"clean with an exclude pattern", "git clean -fd -e keep", cleanRepo(top, true), Allow, ""},
		// W1D-RESIDUAL item 4: the reason names the ways to proceed, an
		// exclude is a pattern judged by the dry run with the same
		// excludes, and a preceded clean removes rather than copies.
		{"clean names a pathspec", "git clean -fdx", cleanRepo(top, true), Deny, "git clean -fdx -- build dist"},
		{"clean names an exclude", "git clean -fdx", cleanRepo(top, true), Deny, "-e .claude"},
		{"clean excluding .claude", "git clean -fdx -e .claude", cleanRepo(top, true), Allow, ""},
		{"clean excluding .claude, attached", "git clean -fdx -e.claude", cleanRepo(top, true), Allow, ""},
		{"clean excluding .claude, long", "git clean -fdx --exclude=.claude", cleanRepo(top, true), Allow, ""},
		{"clean excluding .claude, long separate", "git clean -fdx --exclude .claude", cleanRepo(top, true), Allow, ""},
		{"clean excluding .claude, in a cluster", "git clean -fdxe .claude", cleanRepo(top, true), Allow, ""},
		{"clean excluding .claude, -C", "git -C . clean -fdx -e .claude", cleanRepo(top, true), Allow, ""},
		{"clean excluding something else", "git clean -fdx -e src", cleanRepo(top, true), Deny, removed},
		{"clean with a negated exclude", "git clean -fdx -e '!.claude'", cleanRepo(top, true), Deny, removed},
		{"clean of .claude itself", "git clean -fdx .claude", cleanRepo(top, true), Deny, ""},
		{"preceded clean removes", "git read-tree --empty; git clean -fd", cleanRepo(top, true), Deny, "may change what it removes"},
		{"stash with untracked build output", "git stash -u", cleanRepo(top, false), Allow, ""},
		{"sparse set", "git sparse-checkout set x", rewriteRepo(top, true), Deny, sparse},
		{"sparse init", "git sparse-checkout init --no-cone", rewriteRepo(top, true), Deny, sparse},
		{"sparse reapply", "git sparse-checkout reapply", rewriteRepo(top, true), Deny, sparse},
		{"sparse add of a negation", "git sparse-checkout add '!/*/'", rewriteRepo(top, true), Deny, sparse},
		{"sparse add from stdin", "git sparse-checkout add --stdin", rewriteRepo(top, true), Deny, sparse},
		{"sparse set after a command", "git status && git sparse-checkout set /go.mod", rewriteRepo(top, true), Deny, sparse},
		{"sparse set, nothing protected tracked", "git sparse-checkout set x", rewriteRepo(top, false), Allow, ""},
	}
	for _, tt := range rows {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tool, command := "Bash", tt.command
			if command == "PowerShell" {
				tool, command = "PowerShell", "git clean -fd"
			}
			got, reason, matches := rewriteVerdict(t, tool, command, top, tt.git)
			if got != tt.want {
				t.Fatalf("%q: verdict = %v (matches %v, SP-007 %q), want %v", command, got, matchIDs(matches), reason, tt.want)
			}
			if tt.want == Deny && !strings.Contains(reason, tt.reason) {
				t.Errorf("%q: SP-007 reason = %q, want it to contain %q", command, reason, tt.reason)
			}
			if strings.Contains(command, "clean") && strings.Contains(reason, "copies") {
				t.Errorf("%q: SP-007 reason = %q, want a clean to remove, not copy", command, reason)
			}
		})
	}

	t.Run("real git", func(t *testing.T) {
		t.Parallel()
		cleanRealGit(t)
	})

	for _, command := range []string{
		"git clean -n -fd",
		"git clean --dry-run -fdx",
		"git sparse-checkout list",
		"git sparse-checkout check-rules",
		"git sparse-checkout disable",
		"git sparse-checkout add src docs",
	} {
		t.Run(command, func(t *testing.T) {
			t.Parallel()
			git := cleanRepo(top, true)
			if got, reason, matches := rewriteVerdict(t, "Bash", command, top, git); got != Allow {
				t.Errorf("verdict = %v (matches %v, SP-007 %q), want Allow", got, matchIDs(matches), reason)
			}
			if n := git.callCount(); n != 0 {
				t.Errorf("git spawned %d times, want none: %v", n, git.calls)
			}
		})
	}
}

// cleanRealGit judges git clean against a real repository (skipped without
// git) whose .claude/logs/ is ignored and untracked: the clean is denied with
// a reason naming a pathspec and an exclude, an exclude that keeps .claude is
// allowed because git's own dry run with that exclude removes nothing
// protected, and an exclude that does not keep it, or a pathspec naming it,
// stays denied.
func cleanRealGit(t *testing.T) {
	t.Helper()
	if _, err := procexec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	env := append(gitrepo.ScopedEnv(os.Environ()),
		"GIT_CONFIG_GLOBAL="+filepath.Join(t.TempDir(), "gitconfig"),
		"GIT_CONFIG_NOSYSTEM=1",
	)
	repo := t.TempDir()
	git := func(args ...string) {
		t.Helper()
		full := append([]string{"-c", "user.name=test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false"}, args...)
		cmd := procexec.Command("git", full...)
		cmd.Dir = repo
		cmd.Env = env
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	for rel, content := range map[string]string{
		".gitignore":            ".claude/logs/\nbuild/\n",
		".claude/settings.json": "{}\n",
		".claude/logs/a.log":    "log\n",
		"src/x.go":              "package x\n",
		"build/out":             "out\n",
	} {
		p := filepath.Join(repo, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	git("init", "-q", "-b", "main")
	git("add", ".gitignore", ".claude/settings.json", "src/x.go")
	git("commit", "-q", "-m", "init")
	runner := func(ctx context.Context, c gitrepo.Call) ([]byte, error) {
		c.Env = append(slices.Clone(env), c.Env...)
		return gitrepo.ExecRunner(ctx, c)
	}

	tests := []struct {
		command string
		want    Verdict
		reason  []string
	}{
		{"git clean -fdx", Deny, []string{"would remove protected .claude/logs/", "git clean -fdx -- build dist", "-e .claude"}},
		{"git clean -fdx -e .claude", Allow, nil},
		{"git clean -fdx -e.claude", Allow, nil},
		{"git clean -fdx --exclude=.claude", Allow, nil},
		{"git clean -fdx --exclude .claude", Allow, nil},
		{"git clean -fdx -- build", Allow, nil},
		{"git clean -fdx -e src", Deny, []string{"would remove protected .claude/logs/"}},
		{"git clean -fdx -e '!.claude'", Deny, []string{"would remove protected .claude/logs/"}},
		{"git clean -fdx .claude", Deny, nil},
		{"git read-tree --empty; git clean -fd", Deny, []string{"may change what it removes"}},
		// Denied as on w1-s5: a wrapper may append an exclude (`!.claude`)
		// the dry run never sees.
		{"xargs git clean -fdx -e .claude -e < list", Deny, nil},
		{"xargs -a list git clean -fdx -e .claude", Deny, nil},
		{"parallel git clean -fdx -e .claude ::: x", Deny, nil},
	}
	for _, tt := range tests {
		ctx := EvalContext{ToolName: "Bash", Command: tt.command, CWD: repo, Ctx: t.Context(), gitRunner: runner}
		got, matches := Tier1Rules.EvaluateAll(&ctx)
		reason := ""
		for _, m := range matches {
			if m.Rule.ID == "SP-007" {
				reason = m.Reason
			}
		}
		if got != tt.want {
			t.Errorf("%q: verdict = %v (matches %v, SP-007 %q), want %v", tt.command, got, matchIDs(matches), reason, tt.want)
			continue
		}
		for _, want := range tt.reason {
			if !strings.Contains(reason, want) {
				t.Errorf("%q: SP-007 reason = %q, want it to contain %q", tt.command, reason, want)
			}
		}
		if strings.Contains(reason, "copies") {
			t.Errorf("%q: SP-007 reason = %q, want a clean to remove, not copy", tt.command, reason)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, ".claude", "logs", "a.log")); err != nil {
		t.Errorf("judging removed .claude/logs/a.log: %v", err)
	}
}

// TestOperandArgs_GitClean pins that the -e/--exclude values of a git clean
// are patterns, not operands, in every spelling git accepts and after git's
// global options, while tar's and rsync's pattern options are unchanged and
// another subcommand's -e stays an operand.
func TestOperandArgs_GitClean(t *testing.T) {
	t.Parallel()
	tests := []struct {
		command string
		want    []string
	}{
		{"git clean -fdx -e .claude x", []string{"clean", "-fdx", "x"}},
		{"git -C d clean --exclude=.claude", []string{"-C", "d", "clean"}},
		{"git clean -fdx -e.claude --exclude .claude -- -e", []string{"clean", "-fdx", "--", "-e"}},
		{"git clean -fdxe .claude", []string{"clean"}},
		{"git clean --exc=.claude build", []string{"clean", "build"}},
		{"env GIT_TRACE=0 git clean -e .claude", []string{"GIT_TRACE=0", "git", "clean"}},
		{"git grep -e .claude", []string{"grep", "-e", ".claude"}},
		{"git clean-x -e .claude", []string{"clean-x", "-e", ".claude"}},
		// A wrapper that appends arguments from its input may add an exclude
		// the dry run never sees, so the excludes stay operands.
		{"xargs git clean -fdx -e .claude -e", []string{"git", "clean", "-fdx", "-e", ".claude", "-e"}},
		{"xargs -a list git clean -fdx -e .claude", []string{"-a", "list", "git", "clean", "-fdx", "-e", ".claude"}},
		{"parallel git clean -fdx -e .claude ::: x", []string{"git", "clean", "-fdx", "-e", ".claude", ":::", "x"}},
		{"tar --exclude=.claude -cf a.tar .", []string{"-cf", "a.tar", "."}},
		{"rsync -a --exclude .claude src/ dst/", []string{"-a", "src/", "dst/"}},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			ctx := EvalContext{ToolName: "Bash", Command: tt.command, CWD: t.TempDir()}
			scs, err := ctx.scannedCommands()
			if err != nil || len(scs) != 1 {
				t.Fatalf("scannedCommands(%q) = %d commands, %v; want one", tt.command, len(scs), err)
			}
			if got := operandArgs(scs[0]); !slices.Equal(got, tt.want) {
				t.Errorf("operandArgs(%q) = %q, want %q", tt.command, got, tt.want)
			}
		})
	}
}

// TestRepoPinnedRead pins which commands read a repository a variable pins
// the way `git --git-dir=<dir>` reads it: a read-only git whose assignments,
// prefix or through env, only choose the repository. Any other variable, an
// env option, an option that writes or runs a program, or a rewriting
// subcommand is not one.
func TestRepoPinnedRead(t *testing.T) {
	t.Parallel()
	tests := []struct {
		command string
		want    bool
	}{
		{"GIT_DIR=../other/.git git log", true},
		{"GIT_DIR=.git git status", true},
		{"GIT_DIR=../o/.git GIT_WORK_TREE=../o git status", true},
		{"env GIT_DIR=../other/.git git log --oneline -5", true},
		{"GIT_DIR=.git env GIT_WORK_TREE=. git diff", true},
		{"git log", false},
		{"env git log", false},
		{"GIT_DIR=.git GIT_EXTERNAL_DIFF=/tmp/x git diff", false},
		{"env GIT_DIR=.git GIT_EXTERNAL_DIFF=/tmp/x git diff", false},
		{"env -i GIT_DIR=.git git log", false},
		{"env GIT_DIR=.git nice git log", false},
		{"GIT_DIR=.git git log --output=x", false},
		{"GIT_DIR=.git git -c core.pager=x log", false},
		{"GIT_DIR=.git git checkout evil -- .", false},
		{"GIT_DIR=$d git log", false},
		{"GIT_DIR=.git cat x", false},
		// Every read-only subcommand form, as cmdscan.GitReadOnly judges it.
		{"GIT_DIR=../other/.git git branch", true},
		{"GIT_DIR=../other/.git git branch -a", true},
		{"GIT_DIR=../other/.git git tag", true},
		{"GIT_DIR=../other/.git git tag -l 'v*'", true},
		{"GIT_DIR=../other/.git git remote -v", true},
		{"GIT_DIR=../other/.git git stash list", true},
		{"GIT_DIR=../other/.git git rev-list HEAD", true},
		{"GIT_DIR=../other/.git git cat-file -p HEAD", true},
		{"GIT_DIR=../other/.git git describe --always", true},
		{"GIT_DIR=../other/.git git for-each-ref", true},
		{"GIT_DIR=../other/.git git shortlog -s HEAD", true},
		{"GIT_DIR=../other/.git git ls-tree HEAD", true},
		{"env GIT_DIR=../other/.git git branch", true},
		{"GIT_DIR=../other/.git GIT_WORK_TREE=../other git tag", true},
		{"GIT_DIR=../other/.git git branch newb", false},
		{"GIT_DIR=../other/.git git tag v1", false},
		{"GIT_DIR=../other/.git git stash drop", false},
		// Still read as on w1-s5: the wider read set and a leading --no-pager
		// or -P.
		{"GIT_DIR=../other/.git git --no-pager log", true},
		{"GIT_DIR=../other/.git git -P log", true},
		{"GIT_DIR=../other/.git git --no-pager branch", true},
		{"GIT_DIR=../other/.git git merge-base HEAD HEAD", true},
		{"GIT_DIR=../other/.git git name-rev HEAD", true},
		{"GIT_DIR=../other/.git git show-ref", true},
		{"GIT_DIR=../other/.git git count-objects -v", true},
		{"GIT_DIR=../other/.git git var -l", true},
		{"GIT_DIR=../other/.git git check-ignore x", true},
		{"GIT_DIR=../other/.git git whatchanged", true},
		{"GIT_DIR=../other/.git git range-diff a...b", true},
		{"GIT_DIR=../other/.git git reflog", true},
		{"GIT_DIR=../other/.git git worktree list", true},
		{"GIT_DIR=../other/.git git notes list", true},
		{"GIT_DIR=../other/.git git symbolic-ref HEAD", true},
		{"GIT_DIR=../other/.git git --no-pager branch newb", false},
		{"GIT_DIR=../other/.git git reflog expire --all", false},
		{"GIT_DIR=../other/.git git worktree add x", false},
		{"GIT_DIR=../other/.git git symbolic-ref HEAD refs/heads/x", false},
		// Location pins (GIT_DIR, GIT_WORK_TREE, GIT_COMMON_DIR) only name
		// where git looks; a value is judged as --git-dir=<value> and
		// --work-tree=<value> are, so a protected one does not count (as on
		// w1-s5).
		{"GIT_WORK_TREE=. git status", true},
		{"GIT_WORK_TREE=. git checkout evil -- .", false},
		{"GIT_COMMON_DIR=../o/.git git log", true},
		{"GIT_COMMON_DIR=../o/.git git stash drop", false},
		{"GIT_DIR=.claude git status", false},
		{"GIT_WORK_TREE=.claude git status", false},
		{"GIT_COMMON_DIR=.claude git log", false},
		{"env GIT_DIR=.claude git status", false},
		// Write-target pins: git status rewrites the index GIT_INDEX_FILE
		// names, and git writes objects into GIT_OBJECT_DIRECTORY, so their
		// value must name no protected path or git-control holder.
		{"GIT_INDEX_FILE=../o/.git/index git status", true},
		{"GIT_INDEX_FILE=.claude/hooks/x git status", false},
		{"GIT_INDEX_FILE=.claude/hooks/guard.py git status", false},
		{"GIT_INDEX_FILE=.git/hooks/pre-commit git status", false},
		{"GIT_INDEX_FILE=.git git status", false},
		{"GIT_INDEX_FILE=$f git status", false},
		{"env GIT_INDEX_FILE=.claude/x git status", false},
		{"GIT_DIR=.git GIT_INDEX_FILE=.claude/x git status", false},
		{"GIT_OBJECT_DIRECTORY=.git/objects git log", true},
		{"GIT_OBJECT_DIRECTORY=.claude/hooks git log", false},
		{"env GIT_OBJECT_DIRECTORY=.claude/hooks git log", false},
		// Every other variable changes more than the repository.
		{"GIT_TRACE=x git log", false},
		{"GIT_TRACE=.claude/hooks/x git log", false},
		{"GIT_EXTERNAL_DIFF=x git diff", false},
		{"GIT_CONFIG_COUNT=1 git log", false},
		{"PATH=x git log", false},
		{"GIT_NAMESPACE=x git log", false},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			ctx := EvalContext{ToolName: "Bash", Command: tt.command, CWD: t.TempDir()}
			scs, err := ctx.scannedCommands()
			if err != nil || len(scs) != 1 {
				t.Fatalf("scannedCommands(%q) = %d commands, %v; want one", tt.command, len(scs), err)
			}
			if got := repoPinnedRead(scs[0]); got != tt.want {
				t.Errorf("repoPinnedRead(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

// TestSP007_PrefixedGit pins that SP-007 judges a git with prefix
// assignments as it judges the same git behind env (`env LC_ALL=C git ...`):
// by its operands and the values assigned, not by the assignment alone. The
// --git-dir workaround the pinned-write reason names stays allowed with a
// harmless variable in front (as on w1-s5), while a value or operand that
// names a protected path, a pinned write, and a variable set for the rest of
// the line still deny. A deny whose pin names a protected path does not
// recommend the --git-dir workaround, which would be denied too.
func TestSP007_PrefixedGit(t *testing.T) {
	t.Parallel()
	cwd := filepath.Join(homeDir(t), "project")
	tests := []struct {
		command string
		want    Verdict
	}{
		// Allowed, as on w1-s5 and as the env form is.
		{"LC_ALL=C git --git-dir=../other/.git log", Allow},
		{"TZ=UTC git --git-dir=../other/.git branch", Allow},
		{"GIT_PAGER=cat git --git-dir=../other/.git log", Allow},
		{"GIT_TERMINAL_PROMPT=0 git --git-dir=../other/.git log", Allow},
		{"LC_ALL=C GIT_DIR=../other/.git git log", Allow},
		{"GIT_ALTERNATE_OBJECT_DIRECTORIES=../other/.git/objects git log", Allow},
		{"env LC_ALL=C git --git-dir=../other/.git log", Allow},
		// Judged by its operands, as `env LC_ALL=C git log -- .git` is: git
		// with a prefix assignment is not proven read-only.
		{"LC_ALL=C git log -- .git", Deny},
		{"env LC_ALL=C git log -- .git", Deny},
		{"LC_ALL=C git --git-dir=../other/.git log --output=.claude/settings.json", Deny},
		// A value that names a protected path.
		{"LC_ALL=.claude/settings.json git --git-dir=../other/.git log", Deny},
		{"GIT_INDEX_FILE=.claude/hooks/x git status", Deny},
		{"GIT_DIR=.claude git status", Deny},
		{"GIT_WORK_TREE=.claude git status", Deny},
		{"GIT_COMMON_DIR=.claude git log", Deny},
		// A variable-pinned git is judged only for reads, whatever else is set.
		{"LC_ALL=C GIT_DIR=../other/.git git branch newb", Deny},
		{"GIT_DIR=../other/.git git -C ../other log", Deny},
		// Variables set for the rest of the line, or on another command.
		// A variable that makes git run another program (GIT-001's).
		{"GIT_EXTERNAL_DIFF=/tmp/x git --git-dir=../other/.git diff", Deny},
		{"GIT_CONFIG_COUNT=1 git --git-dir=../other/.git log", Deny},
		{"export LC_ALL=C; git --git-dir=../other/.git log", Deny},
		{"LC_ALL=C; git --git-dir=../other/.git log", Deny},
		{"LD_PRELOAD=/tmp/x.so cat .claude/settings.json", Deny},
	}
	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			ctx := EvalContext{ToolName: "Bash", Command: tt.command, CWD: cwd}
			if got, reason := sp007.Evaluate(&ctx); got != tt.want {
				t.Errorf("sp007(%q) = %v %q, want %v", tt.command, got, reason, tt.want)
			}
		})
	}

	for _, command := range []string{
		"GIT_DIR=../other/.git GIT_WORK_TREE=.claude git checkout -f HEAD",
		"GIT_DIR=.claude git branch newb",
	} {
		t.Run("reason/"+command, func(t *testing.T) {
			t.Parallel()
			ctx := EvalContext{ToolName: "Bash", Command: command, CWD: cwd}
			got, reason := sp007.Evaluate(&ctx)
			if got != Deny || strings.Contains(reason, "--git-dir") {
				t.Errorf("sp007(%q) = %v %q, want a deny that names no --git-dir workaround", command, got, reason)
			}
		})
	}
}
