package check

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
				writeTestScript(t, dir, ".claude/hooks/package-guard.py", "#!/usr/bin/env python3\n")
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
		{name: "unresolvable interpreter of a project script", command: `nonexistent-bin "${CLAUDE_PROJECT_DIR}"/.claude/hooks/x.py`, event: "PreToolUse", wantSev: SeverityHigh, wantProg: "nonexistent-bin"},
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
		{name: "not executable", content: "#!/usr/bin/env python3\n", noExec: true, posix: true, wantProg: "python3"},
		{name: "absolute interpreter is a directory", content: "#!" + filepath.Dir(missingAbs) + "\n", posix: true, wantProg: filepath.Dir(missingAbs)},
		{name: "absolute interpreter not executable", content: "#!" + notExecInterp + "\n", posix: true, wantProg: notExecInterp},
		{name: "CRLF interpreter line", content: "#!/usr/bin/env python3\r\nprint(1)\r\n", posix: true, wantProg: "python3"},
		{name: "env with unsplit option", content: "#!/usr/bin/env python3 -u\n", goos: "linux", wantProg: "python3 -u"},
		{name: "env option without -S", content: "#!/usr/bin/env -i python3\n", goos: "linux", wantProg: "/usr/bin/env"},
		{name: "env -S with options resolvable", content: "#!/usr/bin/env -S python3 -u\n"},
		{name: "env path missing", content: "#!/nonexistent/env python3\n", posix: true, wantProg: "/nonexistent/env"},
		{name: "relative env path missing", content: "#!env python3\n", posix: true, wantProg: "env"},
		{name: "executable by others only", content: "#!/usr/bin/env python3\n", mode: 0o645, notRoot: true, posix: true, wantProg: "python3"},
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
