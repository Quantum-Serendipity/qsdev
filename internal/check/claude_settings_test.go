package check

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
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
		noScript bool
		wantFail []string // failing result names, sorted
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
			wantFail: []string{"claude_bypass_permissions_mode", "claude_disable_bypass_missing", "claude_hook_missing", "claude_hook_missing"},
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeTestFile(t, dir, ClaudeSettingsRelPath, tt.actual)
			if !tt.noScript {
				writeTestFile(t, dir, ".claude/hooks/package-guard.py", "#!/usr/bin/env python3\n")
			}

			results := CheckClaudeSettingsPosture(CheckContext{ProjectRoot: dir, ExpectedClaudeSettings: []byte(tt.expected)})
			var failed []string
			for _, r := range results {
				if r.Status == StatusFail {
					if r.Severity != SeverityHigh {
						t.Errorf("%s severity = %s, want high", r.Name, r.Severity)
					}
					failed = append(failed, r.Name)
				}
			}
			slices.Sort(failed)
			if !slices.Equal(failed, tt.wantFail) {
				t.Errorf("failed checks = %v, want %v\n%+v", failed, tt.wantFail, results)
			}
			if len(tt.wantFail) > 0 && !ShouldFail(results, AuditLevelLow) {
				t.Error("results do not fail the check at --audit-level low")
			}
		})
	}
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
			writeTestFile(t, dir, ".claude/hooks/package-guard.py", "#!/usr/bin/env python3\n")

			results := CheckClaudeSettingsPosture(CheckContext{
				ProjectRoot:            dir,
				ExpectedClaudeSettings: []byte(generatedSettings),
				HooksWithoutPolicy:     tt.hooks,
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

func writeTestFile(t *testing.T, dir, rel, content string) {
	t.Helper()
	abs := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
