package claudecode_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/claudesettings"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestGeneratedGuardRecognized pins that the package-guard hook the generator
// emits, with and without the sandbox prefix, is the shape posture credits as
// running the guard (claudesettings.RunsScript), for every shell tool.
func TestGeneratedGuardRecognized(t *testing.T) {
	t.Parallel()
	for _, sandbox := range []bool{false, true} {
		gf := mustGenerateSettings(t, allHooksAnswers(sandbox), ecosystem.NewRegistry())
		s, err := claudesettings.Parse(gf.Content)
		if err != nil {
			t.Fatalf("sandbox=%v: parse generated settings: %v", sandbox, err)
		}
		for _, tool := range []string{"Bash", "PowerShell", "Monitor"} {
			if !s.RunsScript(claudesettings.EventPreToolUse, tool, ".claude/hooks/package-guard.py", branding.Get().AppName, claudesettings.GuardHookTimeout) {
				t.Errorf("sandbox=%v: generated guard not recognized as running for %s: %+v",
					sandbox, tool, s.Hooks[claudesettings.EventPreToolUse])
			}
		}
	}
}

// TestPackageGuardMatcher_ClaudeCodeSemantics pins which tools the generated
// package-guard matcher fires for under Claude Code's matcher semantics
// (claudesettings.RunsScript: exact names, or an unanchored JavaScript
// regular expression once the matcher holds other characters): every shell
// tool and the MCP server's nix_run tool under any server name, and not
// other tools. Destructive-prevention keeps the shell tools only.
func TestPackageGuardMatcher_ClaudeCodeSemantics(t *testing.T) {
	t.Parallel()
	gf := mustGenerateSettings(t, allHooksAnswers(false), ecosystem.NewRegistry())
	s, err := claudesettings.Parse(gf.Content)
	if err != nil {
		t.Fatalf("parse generated settings: %v", err)
	}
	runs := func(script, tool string) bool {
		return s.RunsScript(claudesettings.EventPreToolUse, tool, script, branding.Get().AppName, 1)
	}
	cases := []struct {
		tool              string
		guard, destroying bool
	}{
		{tool: "Bash", guard: true, destroying: true},
		{tool: "PowerShell", guard: true, destroying: true},
		{tool: "Monitor", guard: true, destroying: true},
		{tool: "mcp__qsdev__qsdev_nix_run", guard: true},
		{tool: "mcp__x__qsdev_nix_run", guard: true},
		{tool: "mcp__my-server__qsdev_nix_run", guard: true},
		{tool: "Write"},
		{tool: "Edit"},
		{tool: "mcp__x__other"},
		{tool: "mcp__x__qsdev_devenv_status"},
		{tool: "qsdev_nix_run"},
	}
	for _, tc := range cases {
		if got := runs(".claude/hooks/package-guard.py", tc.tool); got != tc.guard {
			t.Errorf("package-guard runs for %s = %v, want %v", tc.tool, got, tc.guard)
		}
		if got := runs(".claude/hooks/block-destructive.py", tc.tool); got != tc.destroying {
			t.Errorf("block-destructive runs for %s = %v, want %v", tc.tool, got, tc.destroying)
		}
	}
}
