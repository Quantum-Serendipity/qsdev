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
	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// fakeGit is a gitrepo.Runner that answers from canned output keyed by the
// query (fakeKey) and records every call, so the worktree-rewrite rule can be
// judged without a repository and its git spawns counted.
type fakeGit struct {
	top     string            // the toplevel rev-parse prints; "" means not a repository
	answers map[string]string // fakeKey -> NUL-separated paths
	tracked string            // the NUL-separated paths ls-files prints
	err     error             // returned for every query when set
	// fetchHead is the FETCH_HEAD file `rev-parse --git-path FETCH_HEAD`
	// prints; "" makes that query fail.
	fetchHead string

	mu    sync.Mutex
	calls []gitrepo.Call
}

// fakeKey names a query by the arguments that matter to it: "toplevel",
// "fetch-head", "tracked", "diff <from> [<to>] -- [<pathspec>...]", "stash
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
		return "tracked"
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
		{"echo first, evil", "Bash", "echo hi; git reset --hard HEAD", top, evilRepo(), Deny, tracksOwnCall},
		{"second rewrite on the line, evil", "Bash", "git checkout -- src/x.go && git reset --hard evil", top, evilRepo(), Deny, tracksOwnCall},
		{"commit then benign switch, evil", "Bash", "git add -A && git commit -m x && git switch main", top, evilRepo(), Deny, tracksOwnCall},
		{"stash around a benign switch, evil", "Bash", "git stash && git switch main && git stash pop", top, evilRepo(), Deny, tracksOwnCall},
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
		{"stash push after HEAD moved", "Bash", "git stash", top, headMovedRepo(top), Deny, "git stash " + overwrite},
		{"stash -u after HEAD moved", "Bash", "git stash -u", top, headMovedRepo(top), Deny, overwrite},
		{"stash save after HEAD moved", "Bash", "git stash save wip", top, headMovedRepo(top), Deny, "git stash save " + overwrite},
		{"stash of another path after HEAD moved", "Bash", "git stash push -- src/x.go", top, headMovedRepo(top), Allow, ""},
		{"new branch from HEAD", "Bash", "git checkout -b feature", top, evilRepo(), Allow, ""},
		{"switch to an unchanged branch", "Bash", "git switch main", top, evilRepo(), Allow, ""},
		{"outside a repository", "Bash", "git reset --hard evil", top, &fakeGit{}, Allow, ""},
		{"unknown revision", "Bash", "git reset --hard nosuch", top, evilRepo(), Allow, ""},
		{"read-only git after a rewrite", "Bash", "git switch main && git log", top, evilRepo(), Allow, ""},
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

	tests := []struct {
		command string
		want    worktreeRewrite
		ok      bool
	}{
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
