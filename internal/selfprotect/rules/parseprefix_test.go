package rules

import (
	"testing"
)

// TestEvaluateAll_DeniesMutationBeforeSyntaxError is the regression test for
// the parse-error bypass: bash runs every complete line before a syntax
// error, so a protected mutation followed by a stray closing word on the next
// line still runs and must still be denied, for every protected category
// (org overlay, answers, Claude settings and hooks, the sandbox policy under
// the state directory, .envrc and the audit trail).
func TestEvaluateAll_DeniesMutationBeforeSyntaxError(t *testing.T) {
	t.Parallel()
	mutations := []string{
		// org overlay, reached through cd so no literal path is present
		"cd ~/.config && cp /tmp/x qsdev/defaults.yaml",
		"cd ~/.config && rm -rf qsdev",
		"cd ~/.config && mv /tmp/evil qsdev",
		"cd ~/.config; ln -sfn /tmp/evil qsdev",
		// answers
		"cd .devinit && cp /tmp/x .qsdev-init-answers.yaml",
		// Claude settings and hooks
		"cd .claude && rm settings.json",
		"cd .claude/hooks && cp /tmp/x package-guard.py",
		// sandbox policy
		"cd .qsdev && cp /tmp/x policy.nix",
		// .envrc
		"cd sub && echo evil > ../.envrc",
		// audit trail
		"cd .claude && rm hook-audit.log",
	}
	trailers := []string{"", "\nfi", "\ndone", "\nesac", "\n)", "\nfi\ntouch later"}
	for _, m := range mutations {
		for _, tail := range trailers {
			command := m + tail
			t.Run(command, func(t *testing.T) {
				t.Parallel()
				ctx := &EvalContext{ToolName: "Bash", Command: command, CWD: t.TempDir()}
				if v, _ := Tier1Rules.EvaluateAll(ctx); v != Deny {
					t.Errorf("EvaluateAll(%q) = %v, want deny", command, v)
				}
			})
		}
	}
}

// TestEvaluateAll_UnparseableHarmlessAllowed pins that the executed-prefix
// pass adds no denial of its own: a harmless prefix before a syntax error is
// allowed.
func TestEvaluateAll_UnparseableHarmlessAllowed(t *testing.T) {
	t.Parallel()
	for _, command := range []string{"ls -la\nfi", "cd /tmp && echo ok\ndone", "fi"} {
		ctx := &EvalContext{ToolName: "Bash", Command: command, CWD: t.TempDir()}
		if v, m := Tier1Rules.EvaluateAll(ctx); v != Allow {
			t.Errorf("EvaluateAll(%q) = %v (%v), want allow", command, v, m)
		}
	}
}
