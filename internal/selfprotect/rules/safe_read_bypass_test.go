package rules

import (
	"path/filepath"
	"testing"
)

// TestTier1_ArgDependentVerbsCannotClearDeny pins that a triggered deny is not
// cleared by a command word that is read-only only for some arguments. git, go,
// sort and uniq used to be name-only "safe reads", so `git rm` cleared the
// SP-003 delete trigger and `uniq IN OUT` / `sort -o OUT` cleared SP-007.
func TestTier1_ArgDependentVerbsCannotClearDeny(t *testing.T) {
	t.Parallel()
	home := homeDir(t)
	cwd := filepath.Join(home, "project")

	tests := []struct {
		command string
		want    Verdict
	}{
		{"git rm -f .claude/settings.json", Deny},
		{"git rm -rf .claude/hooks", Deny},
		{"git rm -f ~/.claude/settings.json", Deny},
		{"git rm -f .qsdev/audit/log.jsonl", Deny},
		{"git mv -f .claude/settings.json x", Deny},
		{"cp /tmp/a /tmp/b; uniq /tmp/evil .claude/hooks/tool-gates.py", Deny},
		{"cp /tmp/a /tmp/b; sort -o .claude/settings.json /tmp/evil", Deny},

		// Variables set on or before a read-only command can make it run code
		// (git's external diff, the dynamic loader, the command search path).
		{"GIT_EXTERNAL_DIFF='rm -rf .claude/hooks' git diff", Deny},
		{"export GIT_EXTERNAL_DIFF='rm -rf .claude/hooks'; git diff", Deny},
		{"GIT_EXTERNAL_DIFF='rm -rf .claude/hooks'; export GIT_EXTERNAL_DIFF; git diff", Deny},
		{"LD_PRELOAD=/tmp/rm.so cat .claude/settings.json", Deny},
		{"PATH=/tmp/rm:/usr/bin; cat .claude/settings.json", Deny},

		// Read-only git subcommands still clear an unrelated trigger.
		{"rm -rf /tmp/build && git diff .claude/settings.json", Allow},
		{"rm -rf /tmp/build && git log -p -- .claude/settings.json", Allow},
		{"cp /tmp/a /tmp/b && sort .claude/settings.json", Allow},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			t.Parallel()
			ctx := EvalContext{ToolName: "Bash", Command: tt.command, CWD: cwd}
			if got, matches := Tier1Rules.EvaluateAll(&ctx); got != tt.want {
				t.Errorf("EvaluateAll(%q) = %v %v, want %v", tt.command, got, matches, tt.want)
			}
		})
	}
}

// TestSP007_AppendAndCombinedRedirects pins that the append (>>) and combined
// (&>, &>>) redirect operators count as writes: SP-007 only sees write
// redirects, so reclassifying one as a read would let it clobber settings.
func TestSP007_AppendAndCombinedRedirects(t *testing.T) {
	t.Parallel()
	cwd := filepath.Join(homeDir(t), "project")

	for _, cmd := range []string{
		"echo x >> .claude/settings.local.json",
		"echo x &> .claude/settings.json",
		"echo x &>> .claude/settings.json",
		"echo x >| .claude/settings.json",
	} {
		t.Run(cmd, func(t *testing.T) {
			t.Parallel()
			ctx := EvalContext{ToolName: "Bash", Command: cmd, CWD: cwd}
			if v, _ := sp007.Evaluate(&ctx); v != Deny {
				t.Errorf("sp007(%q) = %v, want Deny", cmd, v)
			}
		})
	}
}

// TestSP001_ProjectControlFiles pins SP-001/SP-013 for the project-level files
// that register or feed enforcement. The Bash rules already denied
// `echo x > .claude/settings.json`; the Write/Edit side must agree.
func TestSP001_ProjectControlFiles(t *testing.T) {
	t.Parallel()
	proj := filepath.Join(homeDir(t), "Repos", "proj")

	tests := []struct {
		name string
		tool string
		path string
		rule string
	}{
		{"write project settings", "Write", filepath.Join(proj, ".claude", "settings.json"), "SP-001"},
		{"edit project local settings", "Edit", filepath.Join(proj, ".claude", "settings.local.json"), "SP-001"},
		{"multiedit project command", "MultiEdit", filepath.Join(proj, ".claude", "commands", "x.md"), "SP-001"},
		{"write project policy", "Write", filepath.Join(proj, ".qsdev", "policy.yaml"), "SP-001"},
		{"write project audit", "Write", filepath.Join(proj, ".qsdev", "audit", "x.jsonl"), "SP-013"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := EvalContext{ToolName: tt.tool, CanonicalPath: tt.path}
			verdict, matches := Tier1Rules.EvaluateAll(&ctx)
			if verdict != Deny {
				t.Fatalf("EvaluateAll(%s %s) = %v, want Deny", tt.tool, tt.path, verdict)
			}
			if id := firstRuleID(matches); id != tt.rule {
				t.Errorf("EvaluateAll(%s %s) first rule = %s, want %s", tt.tool, tt.path, id, tt.rule)
			}
		})
	}
}
