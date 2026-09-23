package rules

import (
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/cmdscan"
)

// TestShellToolsEvaluated verifies the command rules apply to every tool that
// runs a shell command (W033): PowerShell and Monitor send the same
// tool_input.command as Bash, so a Bash-only check let them bypass
// self-protection. A non-shell tool carrying a stray command field is not a
// shell command and stays allowed.
func TestShellToolsEvaluated(t *testing.T) {
	t.Parallel()
	commands := []string{
		"rm -rf ~/.qsdev/config.yaml",
		"unlink .claude/settings.json",
		"chmod 000 .claude/hooks/package-guard.py",
	}
	for _, tool := range cmdscan.ShellTools {
		for _, command := range commands {
			t.Run(tool+"/"+command, func(t *testing.T) {
				t.Parallel()
				ctx := EvalContext{ToolName: tool, Command: command, CWD: filepath.Join(homeDir(t), "project")}
				if got, _ := Tier1Rules.EvaluateAll(&ctx); got != Deny {
					t.Errorf("verdict = %v, want Deny", got)
				}
			})
		}
	}
	t.Run("non-shell tool", func(t *testing.T) {
		t.Parallel()
		ctx := EvalContext{ToolName: "WebFetch", Command: "rm -rf ~/.qsdev/config.yaml", CWD: filepath.Join(homeDir(t), "project")}
		if got, _ := Tier1Rules.EvaluateAll(&ctx); got != Allow {
			t.Errorf("verdict = %v, want Allow", got)
		}
	})
}

// TestAuditLogsProtected verifies the hook audit logs and the SOC 2 trail are
// audit trails (W047, W049): the agent may read them but not truncate,
// delete or relocate them with Write/Edit or a shell command.
func TestAuditLogsProtected(t *testing.T) {
	t.Parallel()
	home := homeDir(t)
	project := filepath.Join(home, "project")
	writes := []string{
		filepath.Join(project, ".claude", "logs", "hook-audit.jsonl"),
		filepath.Join(project, ".claude", "logs", "audit-2026-09-23.jsonl"),
		filepath.Join(project, ".claude", "hook-audit.log"),
		filepath.Join(home, ".claude", "audit", "claude-sessions-2026-09.jsonl"),
	}
	for _, path := range writes {
		t.Run("Write "+path, func(t *testing.T) {
			t.Parallel()
			ctx := EvalContext{ToolName: "Write", FilePath: path, CanonicalPath: path, CWD: project}
			if got, _ := Tier1Rules.EvaluateAll(&ctx); got != Deny {
				t.Errorf("verdict = %v, want Deny", got)
			}
		})
	}
	runBashCases(t, []bashCase{
		{name: "truncate hook log", command: ": > .claude/logs/hook-audit.jsonl"},
		{name: "delete hook logs", command: "rm -rf .claude/logs"},
		{name: "delete package-guard log", command: "rm .claude/hook-audit.log"},
		{name: "delete soc2 trail", command: "rm -rf ~/.claude/audit"},
		{name: "move hook log away", command: "mv .claude/logs/hook-audit.jsonl /tmp/x"},
	}, Deny)
	runBashCases(t, []bashCase{
		{name: "read hook log", command: "tail -n 20 .claude/logs/hook-audit.jsonl"},
	}, Allow)
}
