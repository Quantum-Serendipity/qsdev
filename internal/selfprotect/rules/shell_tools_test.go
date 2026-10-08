package rules

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/selfprotect/canon"
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
	testShellRewritesFileEveryShellTool(t)
	testGitCodeExecutionEveryShellTool(t)
}

// shellRewriteCase is a command ShellRewritesFile judges against names, and
// the name it must report ("" for a command that rewrites none of them).
type shellRewriteCase struct {
	command string
	names   []string
	want    string
}

// testShellRewritesFileEveryShellTool checks the guarded-file and
// environment-file rewrite checks for every shell tool (U18-07, SP-015): a
// Bash-only check let PowerShell and Monitor rewrite .npmrc or ~/.bashrc.
// The modifier forms strip a suffix or prefix that hides the name from a
// word match at run time, so they must stay caught by the expansion fallback.
func testShellRewritesFileEveryShellTool(t *testing.T) {
	t.Helper()
	guarded := canon.GuardedConfigFiles()
	env := canon.EnvSourceFiles()
	cases := []shellRewriteCase{
		{"echo ignore-scripts=false >> .npmrc", guarded, ".npmrc"},
		{"rm .npmrc", guarded, ".npmrc"},
		{"echo 'repos: []' > .pre-commit-config.yaml", guarded, ".pre-commit-config.yaml"},
		{"printf x >> ~/.bashrc", env, ".bashrc"},
		{"sed -i 's/enable = true/enable = false/' devenv.nix", env, "devenv.nix"},
		{"f=.npmrc-; echo ignore-scripts=false >> ${f%-}", guarded, ".npmrc"},
		{"f=.npmrc.orig; cp /dev/null ${f%.orig}", guarded, ".npmrc"},
		{"f=.npmrc_x; echo ignore-scripts=false >> ${f%_x}", guarded, ".npmrc"},
		{"f=x.npmrc; echo x > ${f#x}", guarded, ".npmrc"},
		{"f=pnpm-workspace.yaml0; echo x >> ${f%0}", guarded, "pnpm-workspace.yaml"},
		{"f=.pre-commit-config.yaml_; rm ${f%_}", guarded, ".pre-commit-config.yaml"},
		{"f=devenv.nix_; sed -i s/true/false/ ${f%_}", env, "devenv.nix"},
		{"f=.bashrc.bak; printf x >> ~/${f%.bak}", env, ".bashrc"},
		{"sed -n 1p .npmrc", guarded, ""},
		{"git add .npmrc", guarded, ""},
		{"cat .npmrc", guarded, ""},
		{"head .npmrc", guarded, ""},
		{"git diff devenv.nix", env, ""},
	}
	cwd := filepath.Join(homeDir(t), "project")
	for _, tool := range cmdscan.ShellTools {
		for _, tt := range cases {
			t.Run("ShellRewritesFile/"+tool+"/"+tt.command, func(t *testing.T) {
				t.Parallel()
				ctx := EvalContext{ToolName: tool, Command: tt.command, CWD: cwd}
				if name, ok := ShellRewritesFile(&ctx, tt.names); name != tt.want || ok != (tt.want != "") {
					t.Errorf("ShellRewritesFile = (%q, %v), want (%q, %v)", name, ok, tt.want, tt.want != "")
				}
			})
		}
	}
	t.Run("ShellRewritesFile/non-shell tool", func(t *testing.T) {
		t.Parallel()
		ctx := EvalContext{ToolName: "WebFetch", Command: "rm .npmrc", CWD: cwd}
		if name, ok := ShellRewritesFile(&ctx, guarded); ok {
			t.Errorf("ShellRewritesFile = (%q, true), want a non-shell tool not evaluated", name)
		}
	})
}

// TestShellRewritesFile_Redefinitions verifies a command the line redefines
// as an alias or function is not trusted by its name (U18-WS3 review): the
// gh exemption, the read-only modelling (sed -n, cat) and git staging would
// otherwise let `alias gh='rm -f'; gh pr create .npmrc` write the file. The
// POSIX tools are checked here; PowerShell in TestShellRewritesFile_PowerShell.
func TestShellRewritesFile_Redefinitions(t *testing.T) {
	t.Parallel()
	guarded := slices.Concat(canon.GuardedConfigFiles(), []string{".qsdev.yaml"})
	cases := []struct{ command, want string }{
		{"alias gh='rm -f'; gh pr create .npmrc", ".npmrc"},
		{"shopt -s expand_aliases\nalias gh='tee -a'\necho ignore-scripts=false | gh pr create .npmrc", ".npmrc"},
		{"alias gh='tee -a'\necho ignore-scripts=false | gh pr create .npmrc", ".npmrc"},
		{"shopt -s expand_aliases\nalias gh='tee'\necho 'security: {}' | gh pr create .qsdev.yaml", ".qsdev.yaml"},
		{"alias cat='tee -a'; echo x | cat .npmrc", ".npmrc"},
		{"alias sed='sed -i'; sed -n 1p .npmrc", ".npmrc"},
		{"alias git='rm -f'; git add .npmrc", ".npmrc"},
		{"builtin alias gh='rm -f'; gh pr create .npmrc", ".npmrc"},
		{`alias "$a"='rm -f'; gh pr create .npmrc`, ".npmrc"},
		{`gh() { command rm -f "$@"; }; gh pr create .npmrc`, ".npmrc"},
		{"gh() { :; }; f=.npmrc; gh pr create \"$f\"", ".npmrc"},
		// A sourced script may define any function (U18-WS3 review).
		{". ./defs.sh; gh pr create .npmrc", ".npmrc"},
		{"source ./defs.sh && gh pr create --title x .qsdev.yaml", ".qsdev.yaml"},
		{"source .npmrc", ""},
		// An alias of another name leaves the real gh and cat alone.
		{"alias ll='ls -l'; gh pr create --title x --body .npmrc", ""},
		{"alias ll='ls -l'; cat .npmrc", ""},
	}
	cwd := filepath.Join(homeDir(t), "project")
	for _, tool := range []string{"Bash", "Monitor"} {
		for _, tt := range cases {
			t.Run(tool+"/"+tt.command, func(t *testing.T) {
				t.Parallel()
				ctx := EvalContext{ToolName: tool, Command: tt.command, CWD: cwd}
				if name, ok := ShellRewritesFile(&ctx, guarded); name != tt.want || ok != (tt.want != "") {
					t.Errorf("ShellRewritesFile = (%q, %v), want (%q, %v)", name, ok, tt.want, tt.want != "")
				}
			})
		}
	}
}

// testGitCodeExecutionEveryShellTool checks the git options that skip hooks
// or run programs for every shell tool (U18-V02).
func testGitCodeExecutionEveryShellTool(t *testing.T) {
	t.Helper()
	cases := []struct {
		command string
		deny    bool
	}{
		{"git commit --no-verify -m x", true},
		{"git -c core.hooksPath=/tmp commit -m x", true},
		{"git config core.hooksPath /tmp", true},
		{"git commit -m x", false},
		// The commit gate's integrity (U18-09).
		{"pre-commit uninstall", true},
		{"prek uninstall", true},
		{"pre-commit install -c /tmp/empty.yaml", true},
		{"pre-commit install --allow-missing-config", true},
		{"prek install --config=/tmp/e.yaml", true},
		{"git init --template=/tmp/t", true},
		{"git clone --templ=/tmp/t url", true},
		{"pre-commit install", false},
		{"prek install", false},
		{"pre-commit run --all-files", false},
	}
	for _, tool := range cmdscan.ShellTools {
		for _, tt := range cases {
			t.Run("GitCodeExecution/"+tool+"/"+tt.command, func(t *testing.T) {
				t.Parallel()
				if reason, deny := GitCodeExecution(&EvalContext{ToolName: tool, Command: tt.command}); deny != tt.deny {
					t.Errorf("GitCodeExecution = (%q, %v), want deny %v", reason, deny, tt.deny)
				}
			})
		}
	}
	// A commit-gate variable set for a commit, in each tool's own spelling.
	skipCommit := map[string]string{
		"Bash":       "SKIP=gitleaks git commit -m x",
		"Monitor":    "SKIP=gitleaks git commit -m x",
		"PowerShell": "$env:SKIP='gitleaks'; git commit -m x",
	}
	for _, tool := range cmdscan.ShellTools {
		command, ok := skipCommit[tool]
		if !ok {
			t.Errorf("no commit-gate variable row for shell tool %s", tool)
			continue
		}
		t.Run("GitCodeExecution/"+tool+"/"+command, func(t *testing.T) {
			t.Parallel()
			if reason, deny := GitCodeExecution(&EvalContext{ToolName: tool, Command: command}); !deny {
				t.Errorf("GitCodeExecution = (%q, false), want deny", reason)
			}
		})
	}
	t.Run("GitCodeExecution/non-shell tool", func(t *testing.T) {
		t.Parallel()
		if reason, deny := GitCodeExecution(&EvalContext{ToolName: "WebFetch", Command: "git commit --no-verify -m x"}); deny {
			t.Errorf("GitCodeExecution denied (%q), want a non-shell tool not evaluated", reason)
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
