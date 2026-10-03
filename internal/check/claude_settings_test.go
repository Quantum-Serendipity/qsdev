package check

import (
	"cmp"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/claudesettings"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// generatedSettings mirrors the settings.json qsdev generates at the standard
// and full tiers.
const generatedSettings = `{
  "permissions": {"defaultMode": "default", "disableBypassPermissionsMode": "disable", "allow": [], "deny": []},
  "hooks": {"PreToolUse": [
    {"matcher": "*", "hooks": [{"type": "command", "command": "qsdev selfprotect"}]},
    {"matcher": "Bash", "hooks": [{"type": "command", "command": "\"${CLAUDE_PROJECT_DIR}\"/.claude/hooks/package-guard.py"}]}
  ]}
}`

// TestCheckClaudeSettingsPosture is the regression test for `qsdev check`
// passing after the guard hooks were removed from settings.json and
// bypassPermissions was made the default mode.
func TestCheckClaudeSettingsPosture(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		actual   string
		expected string
		local    string // .claude/settings.local.json; absent when empty
		user     string // the user settings.json; absent when empty
		noScript bool
		wantFail []string                 // failing result names, sorted
		failSev  CheckSeverity            // severity of every failure; high when empty
		checkSev map[string]CheckSeverity // per-check exceptions to failSev
		wantWarn []string                 // warning result names, sorted
	}{
		{name: "intact", actual: generatedSettings, expected: generatedSettings},
		{
			name:     "hooks stripped and bypass enabled",
			actual:   `{"permissions": {"defaultMode": "bypassPermissions", "allow": [], "deny": []}}`,
			expected: generatedSettings,
			wantFail: []string{"claude_bypass_permissions_mode", "claude_disable_bypass_missing", "claude_hook_missing", "claude_hook_missing"},
		},
		{
			name:     "package guard dropped only",
			actual:   strings.Replace(generatedSettings, `.claude/hooks/package-guard.py`, `.claude/hooks/other.py`, 1),
			expected: generatedSettings,
			wantFail: []string{"claude_hook_missing", "claude_hook_script_missing"},
		},
		{
			name:     "registered script deleted",
			actual:   generatedSettings,
			expected: generatedSettings,
			noScript: true,
			wantFail: []string{"claude_hook_script_missing"},
		},
		{
			name:     "no expected settings still flags bypass",
			actual:   `{"permissions": {"defaultMode": "bypassPermissions"}}`,
			wantFail: []string{"claude_bypass_permissions_mode"},
		},
		{
			name:     "tier without disableBypass does not require it",
			actual:   strings.Replace(generatedSettings, `"disableBypassPermissionsMode": "disable", `, "", 1),
			expected: strings.Replace(generatedSettings, `"disableBypassPermissionsMode": "disable", `, "", 1),
		},
		{
			name:     "all hooks disabled",
			actual:   strings.Replace(generatedSettings, `"hooks": {`, `"disableAllHooks": true, "hooks": {`, 1),
			expected: generatedSettings,
			wantFail: []string{"claude_all_hooks_disabled"},
			failSev:  SeverityCritical,
		},
		{
			name:     "guard narrowed by an if condition",
			actual:   strings.Replace(generatedSettings, `"command": "qsdev selfprotect"`, `"command": "qsdev selfprotect", "if": "Bash(never *)"`, 1),
			expected: generatedSettings,
			wantFail: []string{"claude_hook_missing"},
		},
		{
			// encoding/json matches struct fields case-insensitively; Claude
			// Code reads "hooks" and "permissions" exactly, so decoy keys in
			// another case must not satisfy the check.
			name: "decoy keys in another case",
			actual: `{"permissions": {"defaultMode": "bypassPermissions"}, "hooks": {},
  "Permissions": {"defaultMode": "default", "disableBypassPermissionsMode": "disable"},
  "Hooks": ` + generatedSettings[strings.Index(generatedSettings, `{"PreToolUse"`):],
			expected: generatedSettings,
			wantFail: []string{
				"claude_bypass_permissions_mode", "claude_disable_bypass_missing", "claude_hook_missing", "claude_hook_missing",
				"claude_settings_unloadable",
			},
			// Claude Code refuses a file declaring PreToolUse hooks outside
			// "hooks", so the decoy also makes it unloadable.
			checkSev: map[string]CheckSeverity{"claude_settings_unloadable": SeverityCritical},
		},
		{
			name:     "hook policy env intact, user variable added",
			actual:   withEnv(`{"TOOL_GATES_DENIED": "WebFetch", "MY_VAR": "x"}`),
			expected: withEnv(`{"TOOL_GATES_DENIED": "WebFetch"}`),
		},
		{
			name:     "hook policy env removed",
			actual:   generatedSettings,
			expected: withEnv(`{"TOOL_GATES_DENIED": "WebFetch"}`),
			wantFail: []string{"claude_hook_env_changed"},
		},
		{
			name:     "hook policy env emptied",
			actual:   withEnv(`{"TOOL_GATES_DENIED": ""}`),
			expected: withEnv(`{"TOOL_GATES_DENIED": "WebFetch"}`),
			wantFail: []string{"claude_hook_env_changed"},
		},
		{
			name:     "hook policy env not a string",
			actual:   withEnv(`{"TOOL_GATES_DENIED": ["WebFetch"]}`),
			expected: withEnv(`{"TOOL_GATES_DENIED": "WebFetch"}`),
			wantFail: []string{"claude_hook_env_changed"},
		},
		{
			name:     "unparseable settings",
			actual:   `{"permissions": `,
			expected: generatedSettings,
			wantFail: []string{"claude_settings_parse"},
		},
		{
			name:     "no local file unchanged",
			actual:   withEnv(`{"TOOL_GATES_DENIED": "WebFetch"}`),
			expected: withEnv(`{"TOOL_GATES_DENIED": "WebFetch"}`),
		},
		{
			name:     "local disableAllHooks",
			actual:   generatedSettings,
			expected: generatedSettings,
			local:    `{"disableAllHooks": true}`,
			wantFail: []string{"claude_settings_local_override"},
			failSev:  SeverityCritical,
		},
		{
			// Claude Code honours a user-level disableAllHooks for every
			// project, so it switches the guards off just as a local one does.
			name:     "user disableAllHooks",
			actual:   generatedSettings,
			expected: generatedSettings,
			user:     `{"disableAllHooks": true}`,
			wantFail: []string{"claude_settings_user_override"},
			failSev:  SeverityCritical,
		},
		{
			name:     "user settings without override",
			actual:   generatedSettings,
			expected: generatedSettings,
			user:     `{"permissions": {"allow": ["Bash(ls)"]}}`,
		},
		{
			name:     "user parse error",
			actual:   generatedSettings,
			expected: generatedSettings,
			user:     `{"disableAllHooks": `,
			wantFail: []string{"claude_settings_parse"},
		},
		{
			name:     "local bypassPermissions",
			actual:   generatedSettings,
			expected: generatedSettings,
			local:    `{"permissions": {"defaultMode": "bypassPermissions"}}`,
			wantFail: []string{"claude_settings_local_override"},
		},
		{
			name:     "local env policy override warns",
			actual:   withEnv(`{"TOOL_GATES_DENIED": "WebFetch"}`),
			expected: withEnv(`{"TOOL_GATES_DENIED": "WebFetch"}`),
			local:    `{"env": {"TOOL_GATES_DENIED": "", "MY_VAR": "x"}}`,
			wantWarn: []string{"claude_settings_local_override"},
		},
		{
			name:     "local launch env",
			actual:   generatedSettings,
			expected: generatedSettings,
			local:    `{"env": {"PYTHONPATH": ".claude/rules", "BASH_ENV": "x"}}`,
			wantFail: []string{"claude_settings_local_override", "claude_settings_local_override"},
			failSev:  SeverityCritical,
		},
		{
			name:     "committed launch env",
			actual:   withEnv(`{"LD_PRELOAD": "/tmp/x.so"}`),
			expected: generatedSettings,
			wantFail: []string{"claude_hook_launch_env"},
			failSev:  SeverityCritical,
		},
		{
			name:     "committed bash function and nix python path",
			actual:   withEnv(`{"BASH_FUNC_qsdev%%": "() { exit 0; }", "NIX_PYTHONPATH": "tools"}`),
			expected: generatedSettings,
			wantFail: []string{"claude_hook_launch_env", "claude_hook_launch_env"},
			failSev:  SeverityCritical,
		},
		{
			// Names no shell imports pass, as on main, though they are not
			// shell identifiers.
			name:     "committed inert non-identifier env",
			actual:   withEnv(`{"my-var": "x", "ProgramFiles(x86)": "C:/Program Files (x86)"}`),
			expected: withEnv(`{"my-var": "x", "ProgramFiles(x86)": "C:/Program Files (x86)"}`),
		},
		{
			name:     "local pyvenv launcher",
			actual:   generatedSettings,
			expected: generatedSettings,
			local:    `{"env": {"__PYVENV_LAUNCHER__": "tools/python3"}}`,
			wantFail: []string{"claude_settings_local_override"},
			failSev:  SeverityCritical,
		},
		{
			name:     "user nix python path",
			actual:   generatedSettings,
			expected: generatedSettings,
			user:     `{"env": {"NIX_PYTHONPATH": "/tmp/x"}}`,
			wantFail: []string{"claude_settings_user_override"},
			failSev:  SeverityCritical,
		},
		{
			// Claude Code refuses a file declaring PreToolUse hooks outside
			// "hooks", so the committed guards beside them are off.
			name:     "guard event at the top level",
			actual:   strings.Replace(generatedSettings, `"hooks": {`, `"PreToolUse": true, "hooks": {`, 1),
			expected: generatedSettings,
			wantFail: []string{"claude_settings_unloadable"},
			failSev:  SeverityCritical,
		},
		{
			name:     "matcher under another key",
			actual:   strings.Replace(generatedSettings, `"hooks": {`, `"x": {"hooks": [1]}, "hooks": {`, 1),
			expected: generatedSettings,
			wantFail: []string{"claude_settings_unloadable"},
			failSev:  SeverityCritical,
		},
		{
			name:     "guard event under permissions",
			actual:   strings.Replace(generatedSettings, `"allow": []`, `"allow": [], "PreToolUse": [1]`, 1),
			expected: generatedSettings,
			wantFail: []string{"claude_settings_unloadable"},
			failSev:  SeverityCritical,
		},
		{
			name:     "exempt key holding guard-shaped data",
			actual:   strings.Replace(generatedSettings, `"hooks": {`, `"mcpServers": {"s": {"PreToolUse": [1]}}, "hooks": {`, 1),
			expected: generatedSettings,
		},
		{
			// A mistyped posture key fails Claude Code's settings schema,
			// so it refuses the file rather than reading "false" as false.
			name:     "disableAllHooks not a boolean",
			actual:   strings.Replace(generatedSettings, `"hooks": {`, `"disableAllHooks": "false", "hooks": {`, 1),
			expected: generatedSettings,
			wantFail: []string{"claude_settings_unloadable"},
			failSev:  SeverityCritical,
		},
		{
			name:     "local guard hooks outside hooks warns",
			actual:   generatedSettings,
			expected: generatedSettings,
			local:    `{"PreToolUse": [{"matcher": "*", "hooks": [{"type": "command", "command": "true"}]}]}`,
			wantWarn: []string{"claude_settings_unloadable"},
		},
		{
			// "disable" is the only value Claude Code's schema accepts; any
			// other makes it refuse the whole local file.
			name:     "local disableBypassPermissionsMode invalid is unloadable",
			actual:   generatedSettings,
			expected: generatedSettings,
			local:    `{"permissions": {"disableBypassPermissionsMode": "allow"}}`,
			wantWarn: []string{"claude_settings_unloadable"},
		},
		{
			name:     "local additions that do not weaken pass",
			actual:   withEnv(`{"TOOL_GATES_DENIED": "WebFetch"}`),
			expected: withEnv(`{"TOOL_GATES_DENIED": "WebFetch"}`),
			local: `{"disableAllHooks": false, "env": {"TOOL_GATES_DENIED": "WebFetch", "MY_VAR": "x"},
  "permissions": {"defaultMode": "acceptEdits", "disableBypassPermissionsMode": "disable", "deny": ["Read(./.env)"]}}`,
		},
		{
			// The decoy-key rule applies to the local file too.
			name:     "local decoy keys ignored",
			actual:   generatedSettings,
			expected: generatedSettings,
			local:    `{"DisableAllHooks": true, "Permissions": {"defaultMode": "bypassPermissions"}}`,
		},
		{
			// Claude Code applies none of a file holding a PreToolUse entry
			// it cannot load: the intact guard beside it is off.
			name:     "unloadable sibling: if number",
			actual:   withPreToolUse(`{"matcher": "Write", "hooks": [{"type": "command", "command": "true", "if": 1}]}`),
			expected: generatedSettings,
			wantFail: []string{"claude_settings_unloadable"},
			failSev:  SeverityCritical,
		},
		{
			name:     "unloadable sibling: timeout string",
			actual:   withPreToolUse(`{"matcher": "Write", "hooks": [{"type": "command", "command": "true", "timeout": "x"}]}`),
			expected: generatedSettings,
			wantFail: []string{"claude_settings_unloadable"},
			failSev:  SeverityCritical,
		},
		{
			name:     "unloadable sibling: hook not an object",
			actual:   withPreToolUse(`{"matcher": "Write", "hooks": [5]}`),
			expected: generatedSettings,
			wantFail: []string{"claude_settings_unloadable"},
			failSev:  SeverityCritical,
		},
		{
			name:     "unloadable sibling: unknown hook type",
			actual:   withPreToolUse(`{"matcher": "Write", "hooks": [{"type": "bogus"}]}`),
			expected: generatedSettings,
			wantFail: []string{"claude_settings_unloadable"},
			failSev:  SeverityCritical,
		},
		{
			name:     "unloadable sibling: hooks not an array",
			actual:   withPreToolUse(`{"matcher": "Write", "hooks": 1}`),
			expected: generatedSettings,
			wantFail: []string{"claude_settings_unloadable"},
			failSev:  SeverityCritical,
		},
		{
			name:     "unloadable sibling: PermissionRequest not an array",
			actual:   strings.Replace(generatedSettings, `"hooks": {`, `"hooks": {"PermissionRequest": "x", `, 1),
			expected: generatedSettings,
			wantFail: []string{"claude_settings_unloadable"},
			failSev:  SeverityCritical,
		},
		{
			name:     "bad entry of another event is only dropped",
			actual:   strings.Replace(generatedSettings, `"hooks": {`, `"hooks": {"PostToolUse": [{"hooks": [{"type": "bogus"}]}], `, 1),
			expected: generatedSettings,
		},
		{
			name:     "local unloadable warns",
			actual:   generatedSettings,
			expected: generatedSettings,
			local:    `{"hooks": {"PreToolUse": [{"hooks": [{"type": "bogus"}]}]}}`,
			wantWarn: []string{"claude_settings_unloadable"},
		},
		{
			// Status refuses the guard when the user file sets a launch-env
			// variable; check agrees.
			name:     "user launch env",
			actual:   generatedSettings,
			expected: generatedSettings,
			user:     `{"env": {"PYTHONPATH": "/tmp/x"}}`,
			wantFail: []string{"claude_settings_user_override"},
			failSev:  SeverityCritical,
		},
		{
			name:     "user launch env overridden by the project",
			actual:   withEnv(`{"HOME": ""}`),
			expected: generatedSettings,
			user:     `{"env": {"HOME": "fakehome"}}`,
		},
		{
			name:     "user inert env",
			actual:   generatedSettings,
			expected: generatedSettings,
			user:     `{"env": {"MY_VAR": "x"}}`,
		},
		{
			name:     "unloadable user file is not applied",
			actual:   generatedSettings,
			expected: generatedSettings,
			user:     `{"disableAllHooks": true, "hooks": {"PreToolUse": [5]}}`,
		},
		{
			name:     "local parse error",
			actual:   generatedSettings,
			expected: generatedSettings,
			local:    `{"disableAllHooks": `,
			wantFail: []string{"claude_settings_parse"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeTestFile(t, dir, ClaudeSettingsRelPath, tt.actual)
			if !tt.noScript {
				writeTestScript(t, dir, ".claude/hooks/package-guard.py", "#!/usr/bin/env python3\n")
			}
			if tt.local != "" {
				writeTestFile(t, dir, claudesettings.LocalRelPath, tt.local)
			}
			userDir := t.TempDir()
			if tt.user != "" {
				writeTestFile(t, userDir, "settings.json", tt.user)
			}
			wantSev := cmp.Or(tt.failSev, SeverityHigh)

			results := CheckClaudeSettingsPosture(CheckContext{
				ProjectRoot:            dir,
				ExpectedClaudeSettings: []byte(tt.expected),
				LookPath:               lookPathFound,
				ClaudeUserDir:          userDir,
			})
			var failed, warned []string
			passed := false
			for _, r := range results {
				switch r.Status {
				case StatusFail:
					if want := cmp.Or(tt.checkSev[r.Name], wantSev); r.Severity != want {
						t.Errorf("%s severity = %s, want %s", r.Name, r.Severity, want)
					}
					failed = append(failed, r.Name)
				case StatusWarn:
					if r.Severity != SeverityMedium {
						t.Errorf("%s severity = %s, want medium", r.Name, r.Severity)
					}
					warned = append(warned, r.Name)
				case StatusPass:
					passed = r.Name == "claude_settings_posture"
				}
				if tt.local != "" && (r.Name == "claude_settings_local_override" || r.Name == "claude_settings_parse") &&
					r.FilePath != claudesettings.LocalRelPath {
					t.Errorf("%s FilePath = %q, want %q", r.Name, r.FilePath, claudesettings.LocalRelPath)
				}
			}
			slices.Sort(failed)
			slices.Sort(warned)
			if !slices.Equal(failed, tt.wantFail) {
				t.Errorf("failed checks = %v, want %v\n%+v", failed, tt.wantFail, results)
			}
			if !slices.Equal(warned, tt.wantWarn) {
				t.Errorf("warned checks = %v, want %v\n%+v", warned, tt.wantWarn, results)
			}
			if wantPass := len(tt.wantFail)+len(tt.wantWarn) == 0; passed != wantPass {
				t.Errorf("claude_settings_posture pass = %v, want %v\n%+v", passed, wantPass, results)
			}
			if len(tt.wantFail) > 0 && !ShouldFail(results, AuditLevelLow) {
				t.Error("results do not fail the check at --audit-level low")
			}
		})
	}
}

// withPreToolUse returns generatedSettings with entry appended to its
// PreToolUse matchers.
func withPreToolUse(entry string) string {
	return strings.Replace(generatedSettings, "\n  ]}", ",\n    "+entry+"\n  ]}", 1)
}

// withEnv returns generatedSettings with the given "env" object.
func withEnv(env string) string {
	return strings.Replace(generatedSettings, "{\n", "{\n  \"env\": "+env+",\n", 1)
}

// TestCheckClaudeSettingsPosture_HooksWithoutPolicy guards W046: an enabled
// hook without the policy it enforces (tool-gates with no allow or deny list)
// is reported as "enabled (no policy)" instead of passing as a control.
func TestCheckClaudeSettingsPosture_HooksWithoutPolicy(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		hooks    []HookWithoutPolicy
		wantWarn []string
	}{
		{name: "every hook has its policy"},
		{
			name:     "tool-gates without policy",
			hooks:    []HookWithoutPolicy{{Name: "tool-gates", PolicyKey: "hooks.tool_gates"}},
			wantWarn: []string{"claude_hook_no_policy"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeTestFile(t, dir, ClaudeSettingsRelPath, generatedSettings)
			writeTestScript(t, dir, ".claude/hooks/package-guard.py", "#!/usr/bin/env python3\n")

			results := CheckClaudeSettingsPosture(CheckContext{
				ProjectRoot:            dir,
				ExpectedClaudeSettings: []byte(generatedSettings),
				HooksWithoutPolicy:     tt.hooks,
				LookPath:               lookPathFound,
			})
			var warned []string
			for _, r := range results {
				switch r.Status {
				case StatusWarn:
					warned = append(warned, r.Name)
					if !strings.Contains(r.Message, "enabled (no policy)") || !strings.Contains(r.Remediation, "hooks.tool_gates") {
						t.Errorf("warning %q / %q does not name the missing policy", r.Message, r.Remediation)
					}
				case StatusFail:
					t.Errorf("unexpected failure %s: %s", r.Name, r.Message)
				}
			}
			if !slices.Equal(warned, tt.wantWarn) {
				t.Errorf("warnings = %v, want %v", warned, tt.wantWarn)
			}
		})
	}
}

// TestCheckHookPrograms guards U18-V04: a hook whose bare command word does
// not resolve on PATH exits 127, and one whose program path does not exist
// or cannot run exits 127 or 126; Claude Code treats each as a non-blocking
// error, so the guard silently stops applying. Shell builtins (cd, source)
// are never looked up on PATH, and project hook scripts are left to
// checkHookScripts.
func TestCheckHookPrograms(t *testing.T) {
	t.Parallel()
	const rootVar = "<root>" // replaced by the project directory in wantProg
	tests := []struct {
		name     string
		command  string
		event    string
		scripts  map[string]os.FileMode // project files to create, by mode
		posix    bool                   // the case only applies outside Windows
		wantSev  CheckSeverity          // empty: no finding
		wantProg string
		wantMsg  string // the program the message names, when not wantProg
	}{
		{name: "unresolvable self-protection", command: "nonexistent-bin selfprotect", event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		{name: "unresolvable other hook", command: "nonexistent-bin --flag", event: "PostToolUse", wantSev: SeverityHigh, wantProg: "nonexistent-bin"},
		{name: "unresolvable after assignment", command: "FOO=1 nonexistent-bin selfprotect", event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		{name: "resolvable", command: "sh -c true", event: "PreToolUse"},
		{name: "project script path", command: `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/x.py`, event: "PreToolUse"},
		{name: "variable command word", command: "$QSDEV_BIN selfprotect", event: "PreToolUse"},
		{name: "unknown variable path", command: `"$OTHER"/tools/guard.py`, event: "PreToolUse"},
		{name: "unparseable", command: "qsdev 'selfprotect", event: "PreToolUse"},
		{name: "unresolvable interpreter of a project script", command: `nonexistent-bin "${CLAUDE_PROJECT_DIR}"/.claude/hooks/x.py`, event: "PreToolUse", wantSev: SeverityHigh, wantProg: "nonexistent-bin"},
		{name: "cd builtin then resolvable program", command: `cd "$CLAUDE_PROJECT_DIR" && gofmt -l .`, event: "PostToolUse"},
		{name: "cd builtin then unresolvable program", command: `cd "$CLAUDE_PROJECT_DIR" && nonexistent-bin`, event: "PostToolUse", wantSev: SeverityHigh, wantProg: "nonexistent-bin"},
		{name: "cd builtin then project script", command: `cd "$CLAUDE_PROJECT_DIR" && ./.claude/hooks/x.py`, event: "PreToolUse"},
		{name: "source builtin", command: `source "${CLAUDE_PROJECT_DIR}"/.env; gofmt -l .`, event: "PostToolUse"},
		{name: "exec wrapper of a project script", command: `exec "${CLAUDE_PROJECT_DIR}"/.claude/hooks/x.py`, event: "PreToolUse"},
		{name: "exec wrapper of an unresolvable program", command: `exec nonexistent-bin selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		{name: "absolute path missing", command: "/nonexistent/qsdev selfprotect", event: "PreToolUse", posix: true, wantSev: SeverityCritical, wantProg: "/nonexistent/qsdev"},
		{name: "relative path missing", command: "./bin/qsdev selfprotect", event: "PreToolUse", wantSev: SeverityCritical, wantProg: "./bin/qsdev"},
		{name: "relative path present", command: "./bin/qsdev selfprotect", event: "PreToolUse", scripts: map[string]os.FileMode{"bin/qsdev": 0o755}},
		{name: "project path missing", command: `"${CLAUDE_PROJECT_DIR}"/tools/guard.py`, event: "PreToolUse", wantSev: SeverityHigh, wantProg: rootVar + "/tools/guard.py"},
		{name: "project path present", command: `"${CLAUDE_PROJECT_DIR}"/tools/guard.py`, event: "PreToolUse", scripts: map[string]os.FileMode{"tools/guard.py": 0o755}},
		{name: "project path not executable", command: `"${CLAUDE_PROJECT_DIR}"/tools/guard.py`, event: "PreToolUse", scripts: map[string]os.FileMode{"tools/guard.py": 0o644}, posix: true, wantSev: SeverityHigh, wantProg: rootVar + "/tools/guard.py"},
		{name: "exec redirect then unresolvable program", command: `exec 2>/dev/null; nonexistent-bin`, event: "PreToolUse", wantSev: SeverityHigh, wantProg: "nonexistent-bin"},
		{name: "exec redirect then resolvable program", command: `exec 2>/dev/null; gofmt -l .`, event: "PostToolUse"},
		{name: "env split string", command: `env -S 'nonexistent-bin -u'`, event: "PreToolUse"},
		{name: "test-guarded optional program", command: `test -x .venv/bin/ruff && .venv/bin/ruff check`, event: "PostToolUse"},
		{name: "bracket-guarded optional program", command: `[ -x "$CLAUDE_PROJECT_DIR/tools/opt" ] && "$CLAUDE_PROJECT_DIR/tools/opt" || true`, event: "PostToolUse"},
		{name: "double-bracket-guarded optional program", command: `[[ -x .venv/bin/ruff ]] && .venv/bin/ruff check`, event: "PostToolUse"},
		{name: "command -v guard then exit", command: `command -v nonexistent-bin >/dev/null || exit 0; nonexistent-bin`, event: "PostToolUse"},
		{name: "command -v and-guard", command: `command -v nonexistent-bin >/dev/null && nonexistent-bin`, event: "PostToolUse"},
		{name: "or branch", command: `true || nonexistent-bin`, event: "PostToolUse"},
		{name: "if branch", command: `if [ -x ./bin/qsdev ]; then ./bin/qsdev selfprotect; fi`, event: "PreToolUse"},
		{name: "function body", command: `f() { nonexistent-bin; }; :`, event: "PreToolUse"},
		{name: "cd and-chain then unresolvable program", command: `cd "$CLAUDE_PROJECT_DIR" && cd .claude && nonexistent-bin selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		{name: "cd or-exit then unresolvable program", command: `cd "$CLAUDE_PROJECT_DIR" || exit 1; nonexistent-bin selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		{name: "optional env file then missing absolute program", command: `cd "$CLAUDE_PROJECT_DIR"; [ -f .env ] && . ./.env; "$CLAUDE_PROJECT_DIR"/bin/missing selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: rootVar + "/bin/missing"},
		{name: "optional env file may change directory", command: `cd "$CLAUDE_PROJECT_DIR"; [ -f .env ] && . ./.env; ./bin/missing selfprotect`, event: "PreToolUse"},
		{name: "optional env file then bare program", command: `cd "$CLAUDE_PROJECT_DIR"; [ -f .env ] && . ./.env; nonexistent-bin selfprotect`, event: "PreToolUse"},
		{name: "if-guarded exit then program", command: `if ! command -v nonexistent-bin >/dev/null; then exit 0; fi; nonexistent-bin`, event: "PostToolUse"},
		{name: "test-guarded exit then program", command: `[ -x ./bin/qsdev ] || exit 0; ./bin/qsdev selfprotect`, event: "PreToolUse"},
		{name: "cd and-exit ends the hook", command: `cd "$CLAUDE_PROJECT_DIR" && exit 0; nonexistent-bin`, event: "PostToolUse"},
		{name: "exit ends the hook", command: `exit 0; nonexistent-bin`, event: "PostToolUse"},
		{name: "exec ends the hook", command: `exec gofmt -l .; nonexistent-bin`, event: "PostToolUse"},
		{name: "or-exec fallback after cd then program", command: `cd "$CLAUDE_PROJECT_DIR" || exec gofmt; nonexistent-bin`, event: "PostToolUse", wantSev: SeverityHigh, wantProg: "nonexistent-bin"},
		{name: "exec redirect then cd and-chain", command: `exec 2>/dev/null; cd "$CLAUDE_PROJECT_DIR" && nonexistent-bin`, event: "PostToolUse", wantSev: SeverityHigh, wantProg: "nonexistent-bin"},
		{name: "earlier test then cd and-chain", command: `test -f x; cd "$CLAUDE_PROJECT_DIR" && nonexistent-bin`, event: "PostToolUse", wantSev: SeverityHigh, wantProg: "nonexistent-bin"},
		{name: "program and-guards the next", command: `gofmt -l . && nonexistent-bin`, event: "PostToolUse"},
		{name: "builtin behind an external wrapper", command: `timeout 5 cd /tmp && gofmt -l .`, event: "PostToolUse", wantSev: SeverityHigh, wantProg: "cd"},
		{name: "builtin behind exec", command: `exec cd /tmp`, event: "PostToolUse", wantSev: SeverityHigh, wantProg: "cd"},
		{name: "builtin behind command", command: `command cd /tmp && gofmt -l .`, event: "PostToolUse"},
		{name: "mapfile builtin", command: `mapfile -t a < /dev/null; gofmt -l .`, event: "PostToolUse"},
		{name: "enable builtin", command: `enable -n test; gofmt -l .`, event: "PostToolUse"},
		{name: "second unconditional program", command: `gofmt -l .; nonexistent-bin`, event: "PostToolUse", wantSev: SeverityHigh, wantProg: "nonexistent-bin"},
		{name: "second program after a pipeline", command: `gofmt -l . | cat; nonexistent-bin selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		// A program looked up after the hook changes PATH may be found there.
		{name: "export PATH then bare program", command: `export PATH="$CLAUDE_PROJECT_DIR/.venv/bin:$PATH"; nonexistent-bin check`, event: "PostToolUse"},
		{name: "export node_modules PATH then bare program", command: `export PATH="$CLAUDE_PROJECT_DIR/node_modules/.bin:$PATH"; nonexistent-bin .`, event: "PostToolUse"},
		{name: "prefix PATH then bare program", command: `PATH="$CLAUDE_PROJECT_DIR/.venv/bin:$PATH" nonexistent-bin check`, event: "PostToolUse"},
		{name: "prefix PATH sets it for its command only", command: `PATH="$CLAUDE_PROJECT_DIR/.venv/bin:$PATH" gofmt -l .; nonexistent-bin selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		{name: "bare PATH assignment then bare program", command: `PATH="$CLAUDE_PROJECT_DIR/.venv/bin:$PATH"; nonexistent-bin check`, event: "PostToolUse"},
		{name: "sourced venv then bare program", command: `cd "$CLAUDE_PROJECT_DIR" && . .venv/bin/activate && nonexistent-bin check`, event: "PostToolUse"},
		{name: "eval then bare program", command: `eval "$(direnv export bash)"; nonexistent-bin check`, event: "PostToolUse"},
		{name: "env -i then bare program", command: `env -i nonexistent-bin check`, event: "PostToolUse"},
		{name: "env -u PATH then bare program", command: `env -u PATH nonexistent-bin check`, event: "PostToolUse"},
		{name: "env PATH operand then bare program", command: `env PATH=/opt/bin nonexistent-bin check`, event: "PostToolUse"},
		{name: "env other operand then bare program", command: `env A=1 nonexistent-bin selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		// A PATH a script shell or eval is started with reaches its script.
		{name: "prefix PATH on bash -c", command: `PATH="$CLAUDE_PROJECT_DIR/.venv/bin:$PATH" bash -c 'nonexistent-bin check'`, event: "PostToolUse"},
		{name: "prefix PATH on sh -c", command: `PATH="$CLAUDE_PROJECT_DIR/.venv/bin:$PATH" sh -c 'nonexistent-bin check'`, event: "PostToolUse"},
		{name: "prefix PATH on exec bash -c", command: `PATH="$CLAUDE_PROJECT_DIR/.venv/bin:$PATH" exec bash -c 'nonexistent-bin check'`, event: "PostToolUse"},
		{name: "prefix PATH on timeout bash -c", command: `PATH="$CLAUDE_PROJECT_DIR/.venv/bin:$PATH" timeout 5 bash -c 'nonexistent-bin check'`, event: "PostToolUse"},
		{name: "prefix PATH on eval", command: `PATH="$CLAUDE_PROJECT_DIR/.venv/bin:$PATH" eval 'nonexistent-bin check'`, event: "PostToolUse"},
		{name: "prefix PATH on eval persists", command: `PATH="$CLAUDE_PROJECT_DIR/.venv/bin:$PATH" eval true; nonexistent-bin check`, event: "PostToolUse"},
		{name: "env PATH operand on sh -c", command: `env PATH="$CLAUDE_PROJECT_DIR/.venv/bin:/usr/bin" sh -c 'nonexistent-bin check'`, event: "PostToolUse"},
		{name: "env -i on sh -c", command: `env -i sh -c 'nonexistent-bin check'`, event: "PostToolUse"},
		{name: "env -u PATH on bash -c", command: `env -u PATH bash -c 'nonexistent-bin check'`, event: "PostToolUse"},
		{name: "prefix PATH on sh -c stays in the child", command: `PATH="$CLAUDE_PROJECT_DIR/.venv/bin:$PATH" sh -c 'gofmt -l .'; nonexistent-bin selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		{name: "prefix PATH on sh -c then missing path program", command: `PATH="$CLAUDE_PROJECT_DIR/.venv/bin:$PATH" sh -c './bin/missing selfprotect'`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "./bin/missing"},
		{name: "env -C on sh -c then relative program", command: `env -C /tmp sh -c './bin/missing selfprotect'`, event: "PreToolUse"},
		{name: "export PATH then missing path program", command: `export PATH="$CLAUDE_PROJECT_DIR/.venv/bin:$PATH"; ./bin/missing selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "./bin/missing"},
		// A shell function the hook defines is no program.
		{name: "function call", command: `nonexistent-bin() { gofmt -l .; }; nonexistent-bin`, event: "PostToolUse"},
		{name: "command bypasses the function", command: `nonexistent-bin() { :; }; command nonexistent-bin`, event: "PostToolUse", wantSev: SeverityHigh, wantProg: "nonexistent-bin"},
		// Only an exit of the hook's own shell ends it.
		{name: "exit in a subshell", command: `(exit 0); nonexistent-bin selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		{name: "exit in a pipeline stage", command: `exit 0 | cat; nonexistent-bin selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		{name: "exit in a command substitution", command: `x=$(exit 1); nonexistent-bin selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		{name: "top-level return", command: `return 0; nonexistent-bin selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		{name: "exit in sh -c", command: `sh -c 'exit 0'; nonexistent-bin selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		{name: "eval exit ends the hook", command: `eval exit 0; nonexistent-bin`, event: "PostToolUse"},
		{name: "program in sh -c", command: `sh -c 'nonexistent-bin selfprotect'`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		{name: "program in eval", command: `eval nonexistent-bin selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		{name: "test in a block after cd", command: `cd /tmp && { test -f y; nonexistent-bin selfprotect; }`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		{name: "cd in a block after a test", command: `test -x ./bin/qsdev && { cd /tmp; nonexistent-bin selfprotect; }`, event: "PreToolUse"},
		// A relative program path resolves in the directory the hook is in.
		{name: "cd to project subdirectory then present relative program", command: `cd "$CLAUDE_PROJECT_DIR/.venv" && ./bin/ruff check`, event: "PostToolUse", scripts: map[string]os.FileMode{".venv/bin/ruff": 0o755}},
		{name: "cd to project subdirectory then missing relative program", command: `cd "$CLAUDE_PROJECT_DIR/.venv" && ./bin/missing selfprotect`, event: "PreToolUse", scripts: map[string]os.FileMode{".venv/bin/ruff": 0o755}, wantSev: SeverityCritical, wantProg: "./bin/missing"},
		// A cd to a directory that is not there fails and leaves the shell where it was.
		{name: "failed cd then relative program", command: `cd "$CLAUDE_PROJECT_DIR/nonexist" 2>/dev/null; ./.venv/bin/ruff check`, event: "PostToolUse", scripts: map[string]os.FileMode{".venv/bin/ruff": 0o755}},
		{name: "failed cd then missing absolute program", command: `cd "$CLAUDE_PROJECT_DIR/nonexist" 2>/dev/null; "$CLAUDE_PROJECT_DIR"/bin/missing selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: rootVar + "/bin/missing"},
		{name: "cd to frontend then present node program", command: `cd "$CLAUDE_PROJECT_DIR/frontend" && ./node_modules/.bin/eslint .`, event: "PostToolUse", scripts: map[string]os.FileMode{"frontend/node_modules/.bin/eslint": 0o755}},
		{name: "cd to project root then missing relative program", command: `cd "$CLAUDE_PROJECT_DIR" && ./bin/missing selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "./bin/missing"},
		{name: "relative cd then relative program", command: `cd frontend && ./node_modules/.bin/eslint .`, event: "PostToolUse"},
		{name: "cd to unknown directory then relative program", command: `cd "$OTHER" && ./bin/missing selfprotect`, event: "PreToolUse"},
		{name: "pushd and popd then relative program", command: `pushd "$CLAUDE_PROJECT_DIR/sub" && popd && ./bin/missing selfprotect`, event: "PreToolUse"},
		{name: "conditional cd then relative program", command: `[ -d "$CLAUDE_PROJECT_DIR/sub" ] && cd "$CLAUDE_PROJECT_DIR/sub"; ./bin/missing selfprotect`, event: "PreToolUse"},
		{name: "cd in a function then relative program", command: `f() { cd /tmp; }; f; ./bin/missing selfprotect`, event: "PreToolUse"},
		{name: "cd in a subshell then relative program", command: `(cd /tmp); ./bin/missing selfprotect`, event: "PreToolUse"},
		{name: "cd elsewhere then missing absolute program", command: `cd /tmp && "$CLAUDE_PROJECT_DIR"/bin/missing selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: rootVar + "/bin/missing"},
		{name: "cd in sh -c stays in the child", command: `sh -c 'cd /tmp'; ./bin/missing selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "./bin/missing"},
		{name: "eval cd then relative program", command: `eval cd /tmp; ./bin/missing selfprotect`, event: "PreToolUse"},
		{name: "env -C then relative program", command: `env -C /tmp ./bin/missing selfprotect`, event: "PreToolUse"},
		// A program whose failure the hook handles does not make it exit 127.
		{name: "version probe or-exit then program", command: `nonexistent-bin --version >/dev/null 2>&1 || exit 0; nonexistent-bin check`, event: "PostToolUse"},
		{name: "or-true program", command: `nonexistent-bin selfprotect || true`, event: "PreToolUse"},
		{name: "negated program", command: `! nonexistent-bin selfprotect`, event: "PreToolUse"},
		{name: "or-handled block runs its first program", command: `{ nonexistent-bin selfprotect; gofmt -l .; } || true`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		{name: "and-list left operand is not handled", command: `nonexistent-bin selfprotect && true`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		// hash -p binds a name without PATH.
		{name: "hash -p then bare program", command: `hash -p "$CLAUDE_PROJECT_DIR/.venv/bin/ruff" nonexistent-bin; nonexistent-bin check`, event: "PostToolUse"},
		{name: "hash -r then bare program", command: `hash -r; nonexistent-bin selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		// A script shell that reads startup files may change PATH first.
		{name: "bash -lc program", command: `bash -lc 'nonexistent-bin check'`, event: "PostToolUse"},
		{name: "bash --login -c program", command: `bash --login -c 'nonexistent-bin check'`, event: "PostToolUse"},
		{name: "zsh -c program", command: `zsh -c 'nonexistent-bin check'`, event: "PostToolUse"},
		{name: "bash -lc relative program", command: `bash -lc './bin/missing check'`, event: "PostToolUse"},
		{name: "bash -lc absolute program", command: `bash -lc '"$CLAUDE_PROJECT_DIR"/bin/missing selfprotect'`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: rootVar + "/bin/missing"},
		{name: "login shell startup stays in the child", command: `bash -lc 'nonexistent-bin check'; nonexistent-bin selfprotect`, event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.posix && runtime.GOOS == "windows" {
				t.Skip("Windows runs hooks through Git Bash, which maps POSIX paths and has no exec bit")
			}
			dir := t.TempDir()
			for rel, mode := range tt.scripts {
				writeTestFileMode(t, dir, rel, "#!/usr/bin/env python3\n", mode)
			}
			cmd, err := json.Marshal(tt.command)
			if err != nil {
				t.Fatal(err)
			}
			actual, err := claudesettings.Parse([]byte(`{"hooks": {"` + tt.event + `": [{"matcher": "*", "hooks": [{"type": "command", "command": ` + string(cmd) + `}]}]}}`))
			if err != nil {
				t.Fatal(err)
			}
			results := checkHookPrograms(dir, actual, lookPathNoBuiltins)
			if tt.wantSev == "" {
				if len(results) != 0 {
					t.Fatalf("unexpected findings: %+v", results)
				}
				return
			}
			if len(results) != 1 {
				t.Fatalf("got %d findings, want 1: %+v", len(results), results)
			}
			r := results[0]
			if r.Name != "claude_hook_unresolvable" || r.Status != StatusFail || r.Severity != tt.wantSev {
				t.Errorf("finding = %s/%s/%s, want claude_hook_unresolvable/fail/%s", r.Name, r.Status, r.Severity, tt.wantSev)
			}
			wantProg := strings.Replace(tt.wantProg, rootVar, dir, 1)
			wantMsg := strings.Replace(cmp.Or(tt.wantMsg, tt.wantProg), rootVar, dir, 1)
			if !strings.Contains(r.Message, wantMsg) || r.Metadata["program"] != wantProg {
				t.Errorf("finding %q (metadata %v) does not name %s (program %s)", r.Message, r.Metadata, wantMsg, wantProg)
			}
			if !ShouldFail(results, AuditLevelHigh) {
				t.Error("finding does not fail the check at --audit-level high")
			}
		})
	}
}

// TestCheckHookPrograms_HomeTilde pins that a program under ~/ resolves in
// the home directory, as the shell expands it, not under the project. It
// sets HOME, so it cannot run in parallel.
func TestCheckHookPrograms_HomeTilde(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows
	writeTestFileMode(t, home, "bin/guard", "#!/bin/sh\n", 0o755)
	tests := []struct {
		name     string
		command  string
		wantProg string // empty: no finding
	}{
		{name: "present", command: "~/bin/guard selfprotect"},
		{name: "missing", command: "~/bin/missing selfprotect", wantProg: filepath.Join(home, "bin", "missing")},
		{name: "other account", command: "~nonexistent-user-qsdev/bin/missing selfprotect"},
		{name: "quoted home variable", command: `"$HOME"/bin/missing selfprotect`},
		{name: "quoted tilde is literal", command: `"~/bin/guard" selfprotect`, wantProg: "~/bin/guard"},
		{name: "escaped tilde is literal", command: `\~/bin/guard selfprotect`, wantProg: "~/bin/guard"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := json.Marshal(tt.command)
			if err != nil {
				t.Fatal(err)
			}
			actual, err := claudesettings.Parse([]byte(`{"hooks": {"PreToolUse": [{"matcher": "*", "hooks": [{"type": "command", "command": ` + string(cmd) + `}]}]}}`))
			if err != nil {
				t.Fatal(err)
			}
			results := checkHookPrograms(t.TempDir(), actual, lookPathNoBuiltins)
			if tt.wantProg == "" {
				if len(results) != 0 {
					t.Fatalf("unexpected findings: %+v", results)
				}
				return
			}
			if len(results) != 1 || results[0].Metadata["program"] != tt.wantProg || results[0].Severity != SeverityCritical {
				t.Fatalf("findings = %+v, want one critical finding naming %s", results, tt.wantProg)
			}
		})
	}
}

// TestCheckHookPrograms_SymlinkDotDot pins that a relative program path
// resolves physically, as the kernel opens it: a ".." after a symlinked
// directory leaves the symlink's target, not the directory holding the
// symlink.
func TestCheckHookPrograms_SymlinkDotDot(t *testing.T) {
	t.Parallel()
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows, and Git Bash resolves paths in its own layer")
	}
	base := t.TempDir()
	project := filepath.Join(base, "proj")
	writeTestFileMode(t, base, "ext/tool", "#!/bin/sh\n", 0o755)
	writeTestFileMode(t, base, "ext/viainterp", "#!./link/../interp\n", 0o755)
	writeTestFileMode(t, base, "ext/interp", "#!/bin/sh\n", 0o755)
	if err := os.MkdirAll(filepath.Join(base, "ext", "inner"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(base, "ext", "inner"), filepath.Join(project, "link")); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name     string
		command  string
		wantProg string // empty: no finding
	}{
		{name: "cd into symlink then dot-dot program", command: `cd "$CLAUDE_PROJECT_DIR/link" && ../tool selfprotect`},
		{name: "dot-dot after symlink in program path", command: `./link/../tool selfprotect`},
		{name: "dot-dot after symlink in interpreter path", command: `cd "$CLAUDE_PROJECT_DIR" && ./link/../viainterp selfprotect`},
		{name: "dot-dot after symlink to missing program", command: `./link/../missing selfprotect`, wantProg: "./link/../missing"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd, err := json.Marshal(tt.command)
			if err != nil {
				t.Fatal(err)
			}
			actual, err := claudesettings.Parse([]byte(`{"hooks": {"PreToolUse": [{"matcher": "*", "hooks": [{"type": "command", "command": ` + string(cmd) + `}]}]}}`))
			if err != nil {
				t.Fatal(err)
			}
			results := checkHookPrograms(project, actual, lookPathNoBuiltins)
			if tt.wantProg == "" {
				if len(results) != 0 {
					t.Fatalf("unexpected findings: %+v", results)
				}
				return
			}
			if len(results) != 1 || results[0].Metadata["program"] != tt.wantProg {
				t.Fatalf("findings = %+v, want one naming %s", results, tt.wantProg)
			}
		})
	}
}

// TestCheckClaudeSettingsPosture_ExecWrappedScript pins that a project hook
// script started through exec is reported once, by checkHookScripts, and
// that exec is not itself looked up on PATH.
func TestCheckClaudeSettingsPosture_ExecWrappedScript(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeTestScript(t, dir, ".claude/hooks/x.py", "#!/usr/bin/env nonexistent-bin\n")
	writeTestFile(t, dir, ClaudeSettingsRelPath,
		`{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": "exec \"${CLAUDE_PROJECT_DIR}\"/.claude/hooks/x.py"}]}]}}`)
	var found []CheckResult
	for _, r := range CheckClaudeSettingsPosture(CheckContext{ProjectRoot: dir, LookPath: lookPathNoBuiltins}) {
		if r.Name == "claude_hook_unresolvable" {
			found = append(found, r)
		}
	}
	if len(found) != 1 || found[0].Metadata["program"] != "nonexistent-bin" {
		t.Errorf("claude_hook_unresolvable findings = %+v, want exactly one naming nonexistent-bin", found)
	}
}

// TestCheckClaudeSettingsPosture_UnresolvableHook runs the probe through the
// posture check with the real PATH lookup.
func TestCheckClaudeSettingsPosture_UnresolvableHook(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeTestFile(t, dir, ClaudeSettingsRelPath,
		`{"hooks": {"PreToolUse": [{"matcher": "*", "hooks": [{"type": "command", "command": "qsdev-nonexistent-bin-u18 selfprotect"}]}]}}`)
	var names []string
	for _, r := range CheckClaudeSettingsPosture(CheckContext{ProjectRoot: dir}) {
		names = append(names, r.Name)
	}
	if !slices.Contains(names, "claude_hook_unresolvable") {
		t.Errorf("results %v do not report claude_hook_unresolvable", names)
	}
}

// TestCheckHookScripts_Interpreter guards XS-WS1 A7: a registered hook
// script whose interpreter does not resolve exits 127, and one that is not
// executable exits 126; Claude Code treats both as non-blocking errors, so the
// guard silently stops applying.
func TestCheckHookScripts_Interpreter(t *testing.T) {
	t.Parallel()
	interpDir := t.TempDir()
	missingAbs := filepath.Join(interpDir, "no-such-python3")
	notExecInterp := filepath.Join(interpDir, "not-exec-python3")
	writeTestFile(t, interpDir, "not-exec-python3", "\x7fELF")
	tests := []struct {
		name     string
		command  string
		content  string
		noExec   bool
		mode     os.FileMode // when set, the script's exact mode
		notRoot  bool        // the case only applies to a non-root user
		posix    bool        // the case only applies where the kernel runs the shebang
		goos     string      // the case only applies on this platform
		wantProg string      // empty: no finding
	}{
		{name: "env interpreter missing", content: "#!/usr/bin/env nonexistent-bin\n", wantProg: "nonexistent-bin"},
		{name: "env -S interpreter missing", content: "#!/usr/bin/env -S nonexistent-bin -u\n", wantProg: "nonexistent-bin"},
		{name: "absolute interpreter missing", content: "#!" + missingAbs + "\n", posix: true, wantProg: missingAbs},
		{name: "not executable", content: "#!/usr/bin/env python3\n", noExec: true, posix: true, wantProg: ".claude/hooks/x.py"},
		{name: "absolute interpreter is a directory", content: "#!" + filepath.Dir(missingAbs) + "\n", posix: true, wantProg: filepath.Dir(missingAbs)},
		{name: "absolute interpreter not executable", content: "#!" + notExecInterp + "\n", posix: true, wantProg: notExecInterp},
		{name: "CRLF interpreter line", content: "#!/usr/bin/env python3\r\nprint(1)\r\n", posix: true, wantProg: "python3"},
		{name: "env with unsplit option", content: "#!/usr/bin/env python3 -u\n", goos: "linux", wantProg: "python3 -u"},
		{name: "env option without -S", content: "#!/usr/bin/env -i python3\n", goos: "linux", wantProg: "/usr/bin/env"},
		{name: "env -S with options resolvable", content: "#!/usr/bin/env -S python3 -u\n"},
		{name: "env path missing", content: "#!/nonexistent/env python3\n", posix: true, wantProg: "/nonexistent/env"},
		{name: "relative env path missing", content: "#!env python3\n", posix: true, wantProg: "env"},
		{name: "executable by others only", content: "#!/usr/bin/env python3\n", mode: 0o645, notRoot: true, posix: true, wantProg: ".claude/hooks/x.py"},
		{
			name:    "timeout wrapper with interpreter",
			command: `timeout 30 python3 "${CLAUDE_PROJECT_DIR}"/.claude/hooks/x.py`,
			content: "#!/usr/bin/env nonexistent-bin\n",
			noExec:  true,
		},
		{
			name:    "env assignment with interpreter",
			command: `env PYTHONSAFEPATH=1 python3 "${CLAUDE_PROJECT_DIR}"/.claude/hooks/x.py`,
			content: "#!" + missingAbs + "\n",
		},
		{
			name:     "exec wrapper",
			command:  `exec "${CLAUDE_PROJECT_DIR}"/.claude/hooks/x.py`,
			content:  "#!/usr/bin/env nonexistent-bin\n",
			wantProg: "nonexistent-bin",
		},
		{
			name:     "timeout wrapper",
			command:  `timeout 30 "${CLAUDE_PROJECT_DIR}"/.claude/hooks/x.py`,
			content:  "#!/usr/bin/env nonexistent-bin\n",
			wantProg: "nonexistent-bin",
		},
		{
			name:     "sh -c script",
			command:  `bash -c '"${CLAUDE_PROJECT_DIR}"/.claude/hooks/x.py'`,
			content:  "#!/usr/bin/env nonexistent-bin\n",
			wantProg: "nonexistent-bin",
		},
		{name: "resolvable", content: "#!/usr/bin/env python3\n"},
		{name: "no shebang runs under sh", content: "echo ok\n"},
		{
			name:     "sandboxed script missing interpreter",
			command:  `qsdev sandbox exec --category linter -- "${CLAUDE_PROJECT_DIR}"/.claude/hooks/x.py`,
			content:  "#!/usr/bin/env nonexistent-bin\n",
			wantProg: "nonexistent-bin",
		},
		{
			name:    "script passed to an interpreter after --",
			command: `python3 -- "${CLAUDE_PROJECT_DIR}"/.claude/hooks/x.py`,
			content: "#!/usr/bin/env nonexistent-bin\n",
			noExec:  true,
		},
		{
			name:    "sandboxed script passed to an interpreter",
			command: `qsdev sandbox exec --category linter -- python3 "${CLAUDE_PROJECT_DIR}"/.claude/hooks/x.py`,
			content: "#!/usr/bin/env nonexistent-bin\n",
			noExec:  true,
		},
		{
			name:     "sandboxed script through a wrapper",
			command:  `qsdev sandbox exec -- timeout 30 "${CLAUDE_PROJECT_DIR}"/.claude/hooks/x.py`,
			content:  "#!/usr/bin/env nonexistent-bin\n",
			wantProg: "nonexistent-bin",
		},
		{
			name:    "script passed to an interpreter",
			command: `sh "${CLAUDE_PROJECT_DIR}"/.claude/hooks/x.py`,
			content: "#!/usr/bin/env nonexistent-bin\n",
			noExec:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.posix && runtime.GOOS == "windows" {
				t.Skip("Windows runs hook scripts through Git Bash, which has no exec bit, tolerates CRLF and maps absolute interpreters itself")
			}
			if tt.goos != "" && runtime.GOOS != tt.goos {
				t.Skipf("the kernel's interpreter-line rule differs from %s here", tt.goos)
			}
			if tt.notRoot && os.Geteuid() == 0 {
				t.Skip("root may execute a file with any execute bit set")
			}
			dir := t.TempDir()
			command := tt.command
			if command == "" {
				command = `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/x.py || { rc=$?; exit 2; }`
			}
			switch {
			case tt.mode != 0:
				writeTestFileMode(t, dir, ".claude/hooks/x.py", tt.content, tt.mode)
				if err := os.Chmod(filepath.Join(dir, ".claude", "hooks", "x.py"), tt.mode); err != nil {
					t.Fatal(err)
				}
			case tt.noExec:
				writeTestFile(t, dir, ".claude/hooks/x.py", tt.content)
			default:
				writeTestScript(t, dir, ".claude/hooks/x.py", tt.content)
			}
			cmd, err := json.Marshal(command)
			if err != nil {
				t.Fatal(err)
			}
			actual, err := claudesettings.Parse([]byte(`{"hooks": {"PreToolUse": [{"matcher": "Bash", "hooks": [{"type": "command", "command": ` + string(cmd) + `}]}]}}`))
			if err != nil {
				t.Fatal(err)
			}
			results := checkHookScripts(dir, actual, lookPathFound)
			if tt.wantProg == "" {
				if len(results) != 0 {
					t.Fatalf("unexpected findings: %+v", results)
				}
				return
			}
			if len(results) != 1 {
				t.Fatalf("got %d findings, want 1: %+v", len(results), results)
			}
			r := results[0]
			if r.Name != "claude_hook_unresolvable" || r.Status != StatusFail || r.Severity != SeverityHigh {
				t.Errorf("finding = %s/%s/%s, want claude_hook_unresolvable/fail/high", r.Name, r.Status, r.Severity)
			}
			if r.Metadata["program"] != tt.wantProg || r.FilePath != ".claude/hooks/x.py" {
				t.Errorf("finding metadata %v, file %q; want program %s, file .claude/hooks/x.py", r.Metadata, r.FilePath, tt.wantProg)
			}
		})
	}
}

// lookPathFound resolves every name except "nonexistent-bin" and names no
// file has (with a blank or a carriage return), so tests do not depend on the
// programs installed on PATH.
func lookPathFound(file string) (string, error) {
	if file == "nonexistent-bin" || strings.ContainsAny(file, " \t\r") {
		return "", exec.ErrNotFound
	}
	return filepath.Join("/usr/bin", file), nil
}

// lookPathNoBuiltins is lookPathFound on a PATH that, like a typical one,
// holds no file named after the shell builtins cd, source, exec, mapfile or
// enable.
func lookPathNoBuiltins(file string) (string, error) {
	if file == "cd" || file == "source" || file == "exec" || file == "mapfile" || file == "enable" {
		return "", exec.ErrNotFound
	}
	return lookPathFound(file)
}

func writeTestFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	writeTestFileMode(t, dir, rel, content, 0o644)
}

// writeTestScript writes an executable hook script, as qsdev generates them.
func writeTestScript(t *testing.T, dir, rel, content string) {
	t.Helper()
	writeTestFileMode(t, dir, rel, content, 0o755)
}

func writeTestFileMode(t *testing.T, dir, rel, content string, mode os.FileMode) {
	t.Helper()
	abs := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

// TestCheckHookScripts_NamesCommand checks that a missing hook script is
// reported once per hook command that runs it, quoting that command.
func TestCheckHookScripts_NamesCommand(t *testing.T) {
	t.Parallel()
	actual, err := claudesettings.Parse([]byte(generatedSettings))
	if err != nil {
		t.Fatal(err)
	}
	results := checkHookScripts(t.TempDir(), actual, lookPathFound)
	if len(results) != 1 {
		t.Fatalf("got %d results, want 1: %+v", len(results), results)
	}
	r := results[0]
	const cmd = `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/package-guard.py`
	if r.Name != "claude_hook_script_missing" || r.Severity != SeverityHigh || r.FilePath != ".claude/hooks/package-guard.py" {
		t.Errorf("result = %+v, want claude_hook_script_missing at high for the guard", r)
	}
	if want := strconv.Quote(cmd); !strings.Contains(r.Message, want) {
		t.Errorf("message %q does not quote the hook command %s", r.Message, want)
	}
}

// TestCheckHookRegistrations_UnregisteredGuardIsCritical pins that a missing
// generated guard registration (a fail-closed PreToolUse hook such as the
// package guard) fails at critical, like a gutted guard script, while other
// missing hooks keep their severity.
func TestCheckHookRegistrations_UnregisteredGuardIsCritical(t *testing.T) {
	t.Parallel()
	guard := claudesettings.FailClosedCommand("package-guard", `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/package-guard.py`)
	expected := claudesettings.Settings{Hooks: map[string][]claudesettings.Matcher{
		claudesettings.EventPreToolUse: {
			{Matcher: "Bash", Hooks: []claudesettings.Hook{{Type: claudesettings.HookTypeCommand, Command: guard}}},
			{Matcher: "LSP", Hooks: []claudesettings.Hook{{Type: claudesettings.HookTypeCommand, Command: "qsdev lsp-guard"}}},
		},
		"PostToolUse": {
			{Matcher: "*", Hooks: []claudesettings.Hook{{Type: claudesettings.HookTypeCommand, Command: "qsdev audit"}}},
		},
	}}
	want := map[string]CheckSeverity{guard: SeverityCritical, "qsdev lsp-guard": SeverityHigh, "qsdev audit": SeverityMedium}
	results := checkHookRegistrations(claudesettings.Settings{}, expected)
	if len(results) != len(want) {
		t.Fatalf("got %d results, want %d: %+v", len(results), len(want), results)
	}
	for _, r := range results {
		if got := r.Severity; got != want[r.Metadata["command"]] {
			t.Errorf("missing %q: severity %s, want %s", r.Metadata["command"], got, want[r.Metadata["command"]])
		}
	}
}

// TestCheckExpectedGeneration pins that a project using Claude Code whose
// expected generator output is unknown fails critically, and that the
// settings posture check then claims nothing is intact.
func TestCheckExpectedGeneration(t *testing.T) {
	t.Parallel()
	genErr := errors.New(`unknown skill "no-such-skill"`)
	enabled := true
	tests := []struct {
		name         string
		err          error
		settings     bool
		cfg          *types.QsdevConfig
		wantStatus   CheckStatus
		wantSeverity CheckSeverity
	}{
		{name: "known output", err: nil, settings: true},
		{name: "settings on disk", err: genErr, settings: true, wantStatus: StatusFail, wantSeverity: SeverityCritical},
		{name: "claude code configured", err: genErr, cfg: &types.QsdevConfig{ClaudeCode: types.ClaudeCodeConfig{Enabled: &enabled}}, wantStatus: StatusFail, wantSeverity: SeverityCritical},
		{name: "no claude code", err: genErr, wantStatus: StatusWarn, wantSeverity: SeverityLow},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tt.settings {
				writeTestFile(t, dir, ClaudeSettingsRelPath, generatedSettings)
			}
			ctx := CheckContext{ProjectRoot: dir, QsdevConfig: tt.cfg, ExpectedGenerationErr: tt.err}
			got := CheckExpectedGeneration(ctx)
			if tt.wantStatus == "" {
				if len(got) != 0 {
					t.Fatalf("got %+v, want no result", got)
				}
				return
			}
			if len(got) != 1 || got[0].Name != "expected_generation_failed" || got[0].Status != tt.wantStatus || got[0].Severity != tt.wantSeverity {
				t.Fatalf("got %+v, want one expected_generation_failed %s/%s", got, tt.wantStatus, tt.wantSeverity)
			}
			if tt.settings && slices.ContainsFunc(CheckClaudeSettingsPosture(ctx), func(r CheckResult) bool { return r.Status == StatusPass }) {
				t.Error("claude_settings_posture passes without the expected settings")
			}
		})
	}
}
