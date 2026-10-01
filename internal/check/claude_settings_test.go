package check

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/claudesettings"
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
		noScript bool
		wantFail []string // failing result names, sorted
		wantWarn []string // warning result names, sorted
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
			name:     "local disableBypassPermissionsMode changed warns",
			actual:   generatedSettings,
			expected: generatedSettings,
			local:    `{"permissions": {"disableBypassPermissionsMode": "allow"}}`,
			wantWarn: []string{"claude_settings_local_override"},
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
				writeTestFile(t, dir, ".claude/hooks/package-guard.py", "#!/usr/bin/env python3\n")
			}
			if tt.local != "" {
				writeTestFile(t, dir, claudesettings.LocalRelPath, tt.local)
			}

			results := CheckClaudeSettingsPosture(CheckContext{
				ProjectRoot:            dir,
				ExpectedClaudeSettings: []byte(tt.expected),
				LookPath:               lookPathFound,
			})
			var failed, warned []string
			passed := false
			for _, r := range results {
				switch r.Status {
				case StatusFail:
					if r.Severity != SeverityHigh {
						t.Errorf("%s severity = %s, want high", r.Name, r.Severity)
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
// not resolve on PATH exits 127, which Claude Code treats as a non-blocking
// error, so the guard silently stops applying.
func TestCheckHookPrograms(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		command  string
		event    string
		wantSev  CheckSeverity // empty: no finding
		wantProg string
	}{
		{name: "unresolvable self-protection", command: "nonexistent-bin selfprotect", event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		{name: "unresolvable other hook", command: "nonexistent-bin --flag", event: "PostToolUse", wantSev: SeverityHigh, wantProg: "nonexistent-bin"},
		{name: "unresolvable after assignment", command: "FOO=1 nonexistent-bin selfprotect", event: "PreToolUse", wantSev: SeverityCritical, wantProg: "nonexistent-bin"},
		{name: "resolvable", command: "sh -c true", event: "PreToolUse"},
		{name: "project script path", command: `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/x.py`, event: "PreToolUse"},
		{name: "variable command word", command: "$QSDEV_BIN selfprotect", event: "PreToolUse"},
		{name: "absolute path", command: "/nonexistent/qsdev selfprotect", event: "PreToolUse"},
		{name: "relative path", command: "./bin/qsdev selfprotect", event: "PreToolUse"},
		{name: "unparseable", command: "qsdev 'selfprotect", event: "PreToolUse"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd, err := json.Marshal(tt.command)
			if err != nil {
				t.Fatal(err)
			}
			actual, err := claudesettings.Parse([]byte(`{"hooks": {"` + tt.event + `": [{"matcher": "*", "hooks": [{"type": "command", "command": ` + string(cmd) + `}]}]}}`))
			if err != nil {
				t.Fatal(err)
			}
			results := checkHookPrograms(actual, lookPathFound)
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
			if !strings.Contains(r.Message, tt.wantProg) || r.Metadata["program"] != tt.wantProg {
				t.Errorf("finding %q (metadata %v) does not name %s", r.Message, r.Metadata, tt.wantProg)
			}
			if !ShouldFail(results, AuditLevelHigh) {
				t.Error("finding does not fail the check at --audit-level high")
			}
		})
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

// lookPathFound resolves every name except "nonexistent-bin", so tests do
// not depend on the programs installed on PATH.
func lookPathFound(file string) (string, error) {
	if file == "nonexistent-bin" {
		return "", exec.ErrNotFound
	}
	return filepath.Join("/usr/bin", file), nil
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

// TestCheckHookScripts_NamesCommand checks that a missing hook script is
// reported once per hook command that runs it, quoting that command.
func TestCheckHookScripts_NamesCommand(t *testing.T) {
	t.Parallel()
	actual, err := claudesettings.Parse([]byte(generatedSettings))
	if err != nil {
		t.Fatal(err)
	}
	results := checkHookScripts(t.TempDir(), actual)
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
