package cmdscan

import (
	"slices"
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
// command, and a declaration builtin as a command named by the builtin, so no
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
		{"export GIT_EXTERNAL_DIFF=/tmp/x", "export", []string{"GIT_EXTERNAL_DIFF=/tmp/x"}, nil, false},
		{"declare -x X", "declare", []string{"-x", "X"}, nil, false},
		{"local v=\"$1\"", "local", []string{"v="}, nil, true},
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
