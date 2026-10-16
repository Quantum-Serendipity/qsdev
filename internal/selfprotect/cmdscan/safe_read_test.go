package cmdscan

import (
	"slices"
	"strings"
	"testing"
)

// TestIsSafeReadCommand pins which parsed commands may clear a
// substring-triggered self-protection deny. A command that can write a named
// operand, delete, or execute a program must never qualify: `git rm`, `sort -o`
// and `uniq IN OUT` used to be "safe reads" and cleared SP-003/SP-007 denies.
func TestIsSafeReadCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		command string
		want    bool
	}{
		// Read-only for any arguments.
		{"cat .claude/settings.json", true},
		{"grep -r deny .claude/", true},
		{"jq . .claude/settings.json", true},
		{"printenv QSDEV_ORG_CONFIG", true},
		{"du -sh .git", true}, // du only sums sizes, whatever its options
		{"du --files0-from=list -c", true},

		// direnv: trust and status subcommands leave the .envrc untouched.
		{"direnv allow .envrc", true},
		{"direnv deny .envrc", true},
		{"direnv status", true},
		{"direnv edit .envrc", false},
		{"direnv exec . sh -c true", false},
		{"direnv", false},

		// git: only inspection subcommands, and only without output/pager options.
		{"git status", true},
		{"git diff .claude/settings.json", true},
		{"git log -p -- .claude/settings.json", true},
		{"git show HEAD:.claude/settings.json", true},
		{"git ls-files .claude", true},
		{"git blame .claude/settings.json", true},
		{"git rm -f .claude/settings.json", false},
		{"git rm -rf .claude/hooks", false},
		{"git mv -f .claude/settings.json x", false},
		{"git checkout HEAD~5 -- .claude/settings.json", false},
		{"git restore .claude/settings.json", false},
		{"git stash", false},
		{"git clean -fdx", false},
		{"git", false},
		{"git -C /tmp status", false},
		{"git -c alias.status=!sh status", false},
		{"git diff --output=.claude/settings.json", false},
		{"git log --output .claude/settings.json", false},
		{"git grep -Osh pattern", false},
		{"git grep --open-files-in-pager=sh x", false},
		{"git grep --open=sh x", false},
		{"git grep -nO pattern", false},
		{"git diff --out=.claude/settings.json", false},
		{"git log --oneline -- .claude/settings.json", true},
		// branch, tag, remote and stash read only in their list forms
		// (GitReadOnly).
		{"git branch", true},
		{"git branch -a", true},
		{"git tag -l 'v*'", true},
		{"git remote -v", true},
		{"git stash list", true},
		{"git rev-list HEAD", true},
		{"git cat-file -p HEAD", true},
		{"git for-each-ref", true},
		{"git branch newb", false},
		{"git branch -D x", false},
		{"git branch -m a b", false},
		{"git tag v1", false},
		{"git tag -d v1", false},
		{"git tag -v v1", false},
		{"git remote add o u", false},
		{"git stash drop", false},
		{"git -C x branch", false}, // a global option before the subcommand
		// --no-pager and -P only stop paging; any other global option fails
		// closed.
		{"git --no-pager log", true},
		{"git -P log", true},
		{"git --no-pager branch", true},
		{"git -P --no-pager status", true},
		{"git --no-pager branch newb", false},
		{"git --no-pager -c alias.log=!sh log", false},
		{"git -p log", false},
		{"git --no-pager", false},
		{"git worktree list", true},
		{"git symbolic-ref HEAD", true},

		// go runs code (go run, go generate, go test).
		{"go run .claude/hooks/x.go", false},
		{"go version", false},

		// sort: stdout only; -o/--output (and abbreviations) write a file and
		// --compress-program runs one.
		{"sort .claude/settings.json", true},
		{"sort -u -k2,2 f", true},
		{"sort -o .claude/settings.json /tmp/evil", false},
		{"sort -uo .claude/settings.json /tmp/evil", false},
		{"sort --output=.claude/settings.json /tmp/evil", false},
		{"sort --outp=.claude/settings.json /tmp/evil", false},
		{"sort --compress-program=sh f", false},

		// uniq's second operand is an output file.
		{"uniq /tmp/evil .claude/hooks/tool-gates.py", false},

		// rg: --pre and --hostname-bin run a program.
		{"rg deny .claude/", true},
		{"rg --pre sh deny .claude/", false},
		{"rg --pre=sh deny .claude/", false},
		{"rg --hostname-bin=sh x", false},

		// A prefix assignment can make any command run code.
		{"GIT_EXTERNAL_DIFF=/tmp/x git diff", false},
		{"LD_PRELOAD=/tmp/x.so cat f", false},

		// Wrappers, declaration builtins and unknown commands stay opaque.
		{"sudo cat x", false},
		{"export GIT_EXTERNAL_DIFF=/tmp/x", false},
		{"sed -i s/a/b/ .claude/settings.json", false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.command)
			if err != nil || len(cmds) != 1 {
				t.Fatalf("Parse(%q) = %+v, %v; want one command", tt.command, cmds, err)
			}
			if got := IsSafeReadCommand(cmds[0]); got != tt.want {
				t.Errorf("IsSafeReadCommand(%q) = %v, want %v", tt.command, got, tt.want)
			}
		})
	}
}

// TestGitReadOnly pins the one read-only git model by subcommand and form:
// branch and tag read only in their list form, remote only when it lists or
// prints a URL, stash only for list and show, and an option that writes a
// file or runs a program makes any form a write.
func TestGitReadOnly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		args string // the subcommand and its arguments
		want bool
	}{
		// Always read-only subcommands.
		{"status", true},
		{"diff HEAD~1", true},
		{"log --oneline -5", true},
		{"show HEAD:f", true},
		{"grep -n x", true},
		{"ls-files -s", true},
		{"blame f", true},
		{"rev-parse --git-dir", true},
		{"ls-tree HEAD", true},
		{"cat-file -p HEAD", true},
		{"describe --always", true},
		{"shortlog -s HEAD", true},
		{"rev-list HEAD", true},
		{"for-each-ref", true},
		{"merge-base HEAD HEAD", true},
		{"name-rev HEAD", true},
		{"show-ref", true},
		{"count-objects -v", true},
		{"var -l", true},
		{"check-ignore x", true},
		{"whatchanged", true},
		{"range-diff a...b", true},
		// They contact a remote, which may run a transport program.
		{"ls-remote .", false},
		{"remote show origin", false},

		// reflog, worktree and notes: their read subcommands.
		{"reflog", true},
		{"reflog show", true},
		{"reflog show HEAD", true},
		{"reflog list", true},
		{"reflog expire --all", false},
		{"reflog delete HEAD@{1}", false},
		{"reflog -5", false}, // not proven to stay show
		{"worktree list", true},
		{"worktree list --porcelain", true},
		{"worktree", false},
		{"worktree add x", false},
		{"worktree remove x", false},
		{"notes", true},
		{"notes list", true},
		{"notes show HEAD", true},
		{"notes add -m x", false},
		{"notes remove", false},

		// symbolic-ref: printing one ref.
		{"symbolic-ref HEAD", true},
		{"symbolic-ref --short -q HEAD", true},
		{"symbolic-ref", false},
		{"symbolic-ref HEAD refs/heads/x", false},
		{"symbolic-ref -d HEAD", false},
		{"symbolic-ref -m why HEAD refs/heads/x", false},

		// branch: the list form only.
		{"branch", true},
		{"branch -a", true},
		{"branch -r", true},
		{"branch -v", true},
		{"branch -vv", true},
		{"branch -av", true},
		{"branch --all --verbose", true},
		{"branch --list 'feat/*'", true},
		{"branch -l p q", true},
		{"branch --contains X", true},
		{"branch --contains", true},
		{"branch --no-merged=main", true},
		{"branch --format=%(refname)", true},
		{"branch --format %(refname)", true},
		{"branch --sort=-committerdate", true},
		{"branch --points-at HEAD", true},
		{"branch --show-current", true},
		{"branch --color=always --no-column --abbrev=8", true},
		{"branch -i --list P", true},
		{"branch newb", false},
		{"branch -D x", false},
		{"branch -d x", false},
		{"branch -m a b", false},
		{"branch -M a b", false},
		{"branch -c a b", false},
		{"branch -C a b", false},
		{"branch -u o/x", false},
		{"branch --set-upstream-to=o", false},
		{"branch --unset-upstream", false},
		{"branch --edit-description", false},
		{"branch -f newb HEAD", false},
		{"branch --contains X newb", false}, // a positional without --list
		{"branch --unknown-option", false},
		{"branch --li p", false}, // an abbreviation is not judged
		{"branch -ad x", false},
		{"branch -- x", false},

		// tag: the list form only; -v runs gpg.
		{"tag", true},
		{"tag -l 'v*'", true},
		{"tag --list", true},
		{"tag -n3", true},
		{"tag -n", true},
		{"tag -l -n2 'v*'", true},
		{"tag --points-at X", true},
		{"tag --contains=X --sort=-v:refname", true},
		{"tag --merged main --format %(refname)", true},
		{"tag -i -l V*", true},
		{"tag --column=never", true},
		{"tag v1", false},
		{"tag -d v1", false},
		{"tag -v v1", false},
		{"tag --verify v1", false},
		{"tag -a v1 -m msg", false},
		{"tag -f v1", false},
		{"tag -n3 v1", false}, // a positional without --list

		// remote: listing and get-url.
		{"remote", true},
		{"remote -v", true},
		{"remote --verbose", true},
		{"remote get-url o", true},
		{"remote get-url --push o", true},
		{"remote add o u", false},
		{"remote remove o", false},
		{"remote set-url o u", false},
		{"remote -v add o u", false},
		{"remote prune o", false},

		// stash: list and show.
		{"stash list", true},
		{"stash show -p", true},
		{"stash show stash@{1}", true},
		{"stash", false},
		{"stash drop", false},
		{"stash pop", false},
		{"stash push -m x", false},

		// An option that writes a file or runs a program makes any form a
		// write.
		{"log --output=x", false},
		{"log --out x", false},
		{"grep -O x", false},
		{"grep -nO x", false},
		{"grep --open-files-in-pager=sh x", false},
		{"stash list --output=x", false},
		{"stash show --output=x", false},
		{"branch --format --output=x", false},
		{"remote get-url --output=x o", false},

		// Anything else is not proven read-only.
		{"checkout x", false},
		{"apply --check p", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.args, func(t *testing.T) {
			t.Parallel()
			words := strings.Fields(tt.args)
			sub, args := "", []string(nil)
			if len(words) > 0 {
				sub, args = words[0], words[1:]
			}
			if got := GitReadOnly(sub, args); got != tt.want {
				t.Errorf("GitReadOnly(%q, %q) = %v, want %v", sub, args, got, tt.want)
			}
		})
	}
}

// TestIsSafeReadVerb_ArgDependentCommandsExcluded verifies that the name-only
// check never vouches for a command whose effect depends on its arguments.
func TestIsSafeReadVerb_ArgDependentCommandsExcluded(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"git", "go", "sort", "uniq", "rg", "sed", "awk", "eval", "sh"} {
		if IsSafeReadVerb(name) {
			t.Errorf("IsSafeReadVerb(%q) = true; the name alone does not prove it read-only", name)
		}
	}
	for name := range argReadOnly {
		if safeReadVerbs[name] {
			t.Errorf("%q is in both safeReadVerbs and argReadOnly; its arguments would be ignored", name)
		}
	}
}

// TestParse_WriteRedirectOperators pins isWriteOp: every operator that creates
// or clobbers its target must be reported as a write redirect (SP-007 only sees
// WriteRedirects), and input operators must not be.
func TestParse_WriteRedirectOperators(t *testing.T) {
	t.Parallel()

	const target = ".claude/settings.json"
	tests := []struct {
		command   string
		wantWrite bool
	}{
		{"echo x > " + target, true},
		{"echo x >> " + target, true},
		{"echo x &> " + target, true},
		{"echo x &>> " + target, true},
		{"echo x >| " + target, true},
		{"cat <> " + target, true},
		{"echo x 2> " + target, true},
		{"cat < " + target, false},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.command)
			if err != nil || len(cmds) != 1 {
				t.Fatalf("Parse(%q) = %+v, %v; want one command", tt.command, cmds, err)
			}
			gotWrite := slices.Contains(cmds[0].WriteRedirects, target)
			gotRead := slices.Contains(cmds[0].ReadRedirects, target)
			if gotWrite != tt.wantWrite || gotRead == tt.wantWrite {
				t.Errorf("Parse(%q): write=%v read=%v, want write=%v",
					tt.command, gotWrite, gotRead, tt.wantWrite)
			}
		})
	}
}

// TestParse_Assignments pins that variable assignments are reported: a prefix
// assignment on its command, a bare assignment statement as a nameless
// command, and a declaration builtin as a command named by the builtin that
// sets the names it is given (see TestParse_OtherAssignForms), so no
// consumer mistakes them for inert text.
func TestParse_Assignments(t *testing.T) {
	t.Parallel()

	tests := []struct {
		command     string
		wantName    string
		wantArgs    []string
		wantAssigns []string
		wantExp     bool
	}{
		{"A=1 B=2 git diff", "git", []string{"diff"}, []string{"A", "B"}, false},
		{"PATH=/tmp/x", "", nil, []string{"PATH"}, false},
		{"X=$(id)", "", nil, []string{"X"}, true},
		{"arr=(a b)", "", nil, []string{"arr"}, true},
		{"export GIT_EXTERNAL_DIFF=/tmp/x", "export", []string{"GIT_EXTERNAL_DIFF=/tmp/x"}, []string{"GIT_EXTERNAL_DIFF"}, false},
		{"declare -x X", "declare", []string{"-x", "X"}, []string{"X"}, false},
		{"local v=\"$1\"", "local", []string{"v="}, []string{"v"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			cmds, err := Parse(tt.command)
			if err != nil || len(cmds) == 0 {
				t.Fatalf("Parse(%q) = %+v, %v; want a command", tt.command, cmds, err)
			}
			c := cmds[0]
			if c.Name != tt.wantName || !slices.Equal(c.Args, tt.wantArgs) ||
				!slices.Equal(c.Assigns, tt.wantAssigns) || c.HasExpansion != tt.wantExp {
				t.Errorf("Parse(%q)[0] = name %q args %q assigns %q exp %v; want %q %q %q %v",
					tt.command, c.Name, c.Args, c.Assigns, c.HasExpansion,
					tt.wantName, tt.wantArgs, tt.wantAssigns, tt.wantExp)
			}
		})
	}
}
