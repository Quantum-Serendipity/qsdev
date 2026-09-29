package rules

import (
	"path/filepath"
	"strings"
	"testing"
)

// TestSP008_SettingsOverride covers F150: SP-008 guards the surfaces that do
// change which hooks run — a Claude Code session started from Bash with
// options or environment that drop or replace the hook settings, a subcommand
// that writes settings, and a write through $CLAUDE_CONFIG_DIR — instead of
// variables nothing reads. Exporting a variable in a Bash call cannot reach
// the running session or its hooks, so plain exports are allowed.
func TestSP008_SettingsOverride(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		command string
		want    Verdict
	}{
		// Hook-dropping options.
		{"bare", `claude --bare -p "hi"`, Deny},
		{"safe mode after prompt flag", "claude -p x --safe-mode", Deny},
		{"restricted", "claude --restricted -p x", Deny},
		{"absolute binary path", "/usr/local/bin/claude --bare -p x", Deny},
		{"windows exe", `claude.exe --bare -p x`, Deny},
		{"nohup wrapper", "nohup claude --bare -p x &", Deny},
		{"npx package", "npx @anthropic-ai/claude-code --bare -p x", Deny},
		{"node entry script", "node /usr/lib/node_modules/@anthropic-ai/claude-code/cli.js --bare -p x", Deny},
		{"native installer release", "~/.local/share/claude/versions/2.1.3 --safe-mode -p x", Deny},
		{"npx pinned package with setting sources", "npx -y @anthropic-ai/claude-code@2.1.0 --setting-sources user -p x", Deny},
		{"sh -c script", `sh -c 'claude --bare -p x'`, Deny},
		{"nested eval in bash -c", `bash -c "eval 'claude --safe-mode -p x'"`, Deny},
		{"program from expansion", "c=claude; $c --bare -p x", Deny},
		{"flag from expansion", `F=--bare; claude "$F" -p x`, Deny},
		{"flag through xargs", "echo --bare | xargs claude -p", Deny},
		{"unparseable", `claude --bare -p "unterminated`, Deny},

		// Settings sources and inline settings.
		{"setting sources drop project", "claude --setting-sources user -p x", Deny},
		{"setting sources equals form", "claude --setting-sources=project,local -p x", Deny},
		{"inline disableAllHooks", `claude --settings '{"disableAllHooks": true}' -p x`, Deny},
		{"inline hooks block", `claude --settings='{"hooks":{}}' -p x`, Deny},
		{"inline env block", `claude --settings '{"env":{"PATH":"/tmp/evil"}}' -p x`, Deny},
		{"settings file", "claude --settings /tmp/s.json -p x", Deny},
		{"settings from expansion", `claude --settings "$S" -p x`, Deny},
		{"invalid inline JSON", `claude --settings '{bad' -p x`, Deny},
		{"settings without value", "claude --settings", Deny},

		// Environment overrides.
		{"prefix config dir", "CLAUDE_CONFIG_DIR=/tmp/cc claude -p x", Deny},
		{"exported config dir", "export CLAUDE_CONFIG_DIR=/tmp/cc; claude -p x", Deny},
		{"env simple mode", "env CLAUDE_CODE_SIMPLE=1 claude -p x", Deny},
		{"sudo safe mode", "sudo CLAUDE_CODE_SAFE_MODE=1 claude -p x", Deny},
		{"relocated home", "HOME=/tmp/empty claude -p x", Deny},
		{"exported home", "export HOME=/tmp/empty && claude -p x", Deny},

		// Settings-writing subcommand.
		{"import", "claude import codex --yes", Deny},

		// Writes through $CLAUDE_CONFIG_DIR.
		{"redirect into config dir", `echo '{}' > "$CLAUDE_CONFIG_DIR/settings.json"`, Deny},
		{"copy into config dir", `cp /tmp/s.json "${CLAUDE_CONFIG_DIR}/settings.json"`, Deny},
		{"cd into config dir", `cd "$CLAUDE_CONFIG_DIR" && echo '{}' > settings.json`, Deny},

		// Allowed.
		{"plain print", `claude -p "summarize the diff"`, Allow},
		{"benign inline settings", `claude --settings '{"model":"sonnet"}' -p x`, Allow},
		{"all setting sources", "claude --setting-sources user,project,local -p x", Allow},
		{"import dry run", "claude import codex --dry-run", Allow},
		{"mcp list", "claude mcp list", Allow},
		{"version", "claude --version", Allow},
		{"flag text inside prompt", `claude -p "explain --bare mode"`, Allow},
		{"flag after end of options", "claude -p -- --bare", Allow},
		{"commit message", `git commit -m "document claude --bare"`, Allow},
		{"grep for flag", "grep -rn -- --bare docs/claude.md", Allow},
		{"read settings", "cat .claude/settings.json", Allow},
		{"print config dir", `echo "$CLAUDE_CONFIG_DIR"`, Allow},
		{"list config dir", `ls "$CLAUDE_CONFIG_DIR"`, Allow},
		{"xargs without override", "echo hi | xargs claude -p", Allow},
		{"other tool with same flag", "mytool --settings conf.json", Allow},
		{"export unrelated", "export PATH=/usr/bin:$PATH", Allow},
		{"home in prompt expansion", `claude -p "$(cat "$HOME/prompt.txt")"`, Allow},
		{"node running other script", "node scripts/claude-report.js --bare", Allow},
		{"git bare clone", "git clone --bare https://example.com/r.git", Allow},
		// These variables are read by nothing, so setting them is harmless.
		{"former placebo export", "export QSDEV_BYPASS_ALL=1", Allow},
		{"former placebo unset", "unset GDEV_SELF_PROTECTION", Allow},
		{"claude env var outside a launch", "export CLAUDE_CODE_SIMPLE=1", Allow},
	}
	cwd := filepath.Join(homeDir(t), "project")
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ctx := EvalContext{ToolName: "Bash", Command: tt.command, CWD: cwd}
			if v, reason := sp008.Evaluate(&ctx); v != tt.want {
				t.Errorf("sp008(%q) = %v (%s), want %v", tt.command, v, reason, tt.want)
			}
		})
	}
}

// TestSP008_OnlyShellTools checks that SP-008 judges shell commands only.
func TestSP008_OnlyShellTools(t *testing.T) {
	t.Parallel()
	for _, tool := range []string{"Write", "Read", "mcp__x__y"} {
		ctx := EvalContext{ToolName: tool, Command: "claude --bare -p x"}
		if v, _ := sp008.Evaluate(&ctx); v != Allow {
			t.Errorf("sp008 with tool %s = %v, want allow", tool, v)
		}
	}
	for _, tool := range []string{"PowerShell", "Monitor"} {
		ctx := EvalContext{ToolName: tool, Command: "claude --bare -p x"}
		if v, _ := sp008.Evaluate(&ctx); v != Deny {
			t.Errorf("sp008 with tool %s = %v, want deny", tool, v)
		}
	}
}

func TestClaudeArgsOverride(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
		want string // substring of the reason; "" for none
	}{
		{"none", []string{"-p", "hi"}, ""},
		{"bare", []string{"--bare"}, "--bare"},
		{"settings key", []string{"--settings", `{"disableAllHooks":false}`}, "disableAllHooks"},
		{"setting sources missing local", []string{"--setting-sources", "user,project"}, "local"},
		{"setting sources case and spaces", []string{"--setting-sources", " User , PROJECT,local"}, ""},
		{"import", []string{"import", "gemini"}, "import"},
		{"import is only a subcommand first", []string{"-p", "import"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := claudeArgsOverride(tt.args)
			if (tt.want == "") != (got == "") || !strings.Contains(got, tt.want) {
				t.Errorf("claudeArgsOverride(%q) = %q, want reason containing %q", tt.args, got, tt.want)
			}
		})
	}
}
