package posture

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// pristineGuard stands in for the generated package-guard.py; fixtures record
// its real hash, so any edit on disk makes it a modified guard.
const pristineGuard = "#!/usr/bin/env python3\n\"\"\"Package guard.\"\"\"\nimport sys\n\nsys.exit(main())\n"

// testAssessOpts supplies pristineGuard as the generator's package guard.
var testAssessOpts = AssessOptions{PackageGuard: []byte(pristineGuard)}

// guardedFiles returns the files of a project whose package guard is in
// force, a pristine guard registered as a PreToolUse hook in settings.json,
// with extra added over them.
func guardedFiles(extra map[string]string) map[string]string {
	files := map[string]string{
		packageGuardPath:        pristineGuard,
		".claude/settings.json": settingsWithPackageGuard,
	}
	maps.Copy(files, extra)
	return files
}

// writeFile overwrites the project file rel without touching its recorded hash.
func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeManifest writes the committed manifest listing the hash of each
// content by path; a path the manifest format rejects is written verbatim, so
// the file does not parse.
func writeManifest(t *testing.T, root string, entries map[string]string) {
	t.Helper()
	var b strings.Builder
	for rel, content := range entries {
		sum, _ := strings.CutPrefix(state.ComputeHash([]byte(content)), state.HashPrefix)
		fmt.Fprintf(&b, "%s  %s\n", sum, rel)
	}
	writeFile(t, root, state.ManifestFile(), b.String())
}

// crlfGuardStatus and crlfGuardReason are what a guard with a CRLF
// interpreter line is judged on this platform: Git Bash on Windows starts it,
// other kernels cannot.
var crlfGuardStatus, crlfGuardReason = func() (LayerStatus, string) {
	if runtime.GOOS == "windows" {
		return LayerEnabled, ""
	}
	return LayerDisabled, "CRLF interpreter line"
}()

// guardLayers are the layers that only the package guard provides.
var guardLayers = []string{"pretooluse-hooks", "age-gating", "install-script-blocking"}

// TestGuardEffective pins that the guard-derived layers are credited only
// while package-guard.py is verifiably in force: enabled, unmodified since
// generation, and run by a PreToolUse hook covering Bash in the effective
// settings with hooks switched on (U23-01, K1, K2).
func TestGuardEffective(t *testing.T) {
	t.Parallel()
	const localPath = ".claude/settings.local.json"
	attachGuard := map[string]bool{"attach-guard": true}
	tests := []struct {
		name string
		// files are written and their hashes recorded; disk then overwrites
		// files after recording, as a later edit would.
		files map[string]string
		disk  map[string]string
		// manifest, when set, is written as the committed manifest, listing
		// the hash of each content (a teammate's committed regeneration).
		manifest  map[string]string
		tools     map[string]bool
		noProject bool
		// guardMode, when set, is recorded as the guard's generated mode
		// (fixtures write it 0o644).
		guardMode os.FileMode
		// noTemplate assesses without the generator's package guard.
		noTemplate bool
		want       LayerStatus
		wantReason string
	}{
		{
			name:       "emptied_guard",
			files:      guardedFiles(nil),
			disk:       map[string]string{packageGuardPath: ""},
			want:       LayerDisabled,
			wantReason: "modified from the generated version",
		},
		{
			name:       "appended_line",
			files:      guardedFiles(nil),
			disk:       map[string]string{packageGuardPath: pristineGuard + "sys.exit(0)\n"},
			want:       LayerDisabled,
			wantReason: "qsdev update --configs-only --overwrite-modified' to restore " + packageGuardPath,
		},
		{
			name:       "disableAllHooks_in_settings_local",
			files:      guardedFiles(map[string]string{localPath: `{"disableAllHooks": true}`}),
			want:       LayerDisabled,
			wantReason: "hooks disabled (disableAllHooks in " + localPath + ")",
		},
		{
			name: "disableAllHooks_in_settings_json",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, "{", `{"disableAllHooks": true, `, 1),
			}),
			want:       LayerDisabled,
			wantReason: "hooks disabled (disableAllHooks in .claude/settings.json)",
		},
		{
			name: "matcher_NeverMatchesAnything",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, guardMatcher, `"matcher": "NeverMatchesAnything"`, 1),
			}),
			want:       LayerDisabled,
			wantReason: "not registered as a blocking PreToolUse hook matching Bash",
		},
		{
			name: "command_echo_guardpath",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, `"command": "`, `"command": "echo `, 1),
			}),
			want:       LayerDisabled,
			wantReason: "not registered as a blocking PreToolUse hook matching Bash",
		},
		{
			name: "hook_if_narrowed",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, `{"type": "command",`, `{"type": "command", "if": "Bash(nonexistent-xyz:*)",`, 1),
			}),
			want:       LayerDisabled,
			wantReason: "not registered as a blocking PreToolUse hook matching Bash",
		},
		{
			name: "hook_type_prompt",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, `"type": "command"`, `"type": "prompt", "prompt": "x"`, 1),
			}),
			want:       LayerDisabled,
			wantReason: "not registered as a blocking PreToolUse hook matching Bash",
		},
		{
			name: "command_output_discarded",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, `package-guard.py"`, `package-guard.py >/dev/null 2>&1 || true"`, 1),
			}),
			want:       LayerDisabled,
			wantReason: "not registered as a blocking PreToolUse hook matching Bash",
		},
		{
			name: "command_sandbox_decoy",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, `"command": "`, `"command": "echo sandbox exec -- `, 1),
			}),
			want:       LayerDisabled,
			wantReason: "not registered as a blocking PreToolUse hook matching Bash",
		},
		{
			name: "command_sandboxed_fail_closed",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, `"command": "`,
					`"command": "qsdev sandbox exec --category linter -- `, 1),
			}),
			want: LayerEnabled,
		},
		{
			name:      "guard_mode_recorded_and_kept",
			files:     guardedFiles(nil),
			guardMode: 0o644,
			want:      LayerEnabled,
		},
		{
			name: "hook_async",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, `{"type": "command",`, `{"type": "command", "async": true,`, 1),
			}),
			want:       LayerDisabled,
			wantReason: "not async, timeout at least 30s",
		},
		{
			name: "hook_timeout_short",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, `{"type": "command",`, `{"type": "command", "timeout": 1,`, 1),
			}),
			want:       LayerDisabled,
			wantReason: "not registered as a blocking PreToolUse hook matching Bash",
		},
		{
			name: "hook_timeout_generated",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, `{"type": "command",`, `{"type": "command", "timeout": 30,`, 1),
			}),
			want: LayerEnabled,
		},
		{
			name: "matcher_go_flag_group",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, guardMatcher, `"matcher": "(?i)bash"`, 1),
			}),
			want:       LayerDisabled,
			wantReason: "not registered as a blocking PreToolUse hook matching Bash",
		},
		{
			name: "matcher_quantified_anchor",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, guardMatcher, `"matcher": "^?Bash|PowerShell|Monitor"`, 1),
			}),
			want: LayerDisabled,
		},
		{
			name: "matcher_starred_end_anchor",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, guardMatcher, `"matcher": "Bash|PowerShell|Monitor|$*"`, 1),
			}),
			want: LayerDisabled,
		},
		{
			name: "matcher_number",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, guardMatcher, `"matcher": 5`, 1),
			}),
			want: LayerDisabled,
		},
		{
			name: "matcher_null",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, guardMatcher, `"matcher": null`, 1),
			}),
			want: LayerDisabled,
		},
		{
			name: "hook_if_number",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, `{"type": "command",`, `{"type": "command", "if": 1,`, 1),
			}),
			want: LayerDisabled,
		},
		{
			name: "hook_if_null",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, `{"type": "command",`, `{"type": "command", "if": null,`, 1),
			}),
			want: LayerDisabled,
		},
		{
			name: "hook_if_array",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, `{"type": "command",`, `{"type": "command", "if": ["Bash(npm *)"],`, 1),
			}),
			want: LayerDisabled,
		},
		{
			name: "hook_status_message_number",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, `{"type": "command",`, `{"type": "command", "statusMessage": 5,`, 1),
			}),
			want: LayerDisabled,
		},
		{
			name: "matcher_comma_list",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, guardMatcher, `"matcher": "Edit, Bash,PowerShell , Monitor"`, 1),
			}),
			want: LayerEnabled,
		},
		{
			// The generator registers the guard for every shell tool: one
			// left out runs its commands unguarded.
			name: "matcher_bash_only",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, guardMatcher, `"matcher": "Bash"`, 1),
			}),
			want:       LayerDisabled,
			wantReason: "matching PowerShell",
		},
		{
			name: "matcher_bash_and_powershell",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, guardMatcher, `"matcher": "Bash|PowerShell"`, 1),
			}),
			want:       LayerDisabled,
			wantReason: "so Monitor commands run unguarded",
		},
		{
			name: "command_relative_path",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, `\"${CLAUDE_PROJECT_DIR}\"/`, "", 1),
			}),
			want:       LayerDisabled,
			wantReason: "not registered as a blocking PreToolUse hook matching Bash",
		},
		{
			// A PR replaces the guard and re-hashes its manifest entry: only
			// the generator's own content vouches for the guard.
			name:       "manifest_rehashed_over_replaced_guard",
			files:      guardedFiles(nil),
			disk:       map[string]string{packageGuardPath: "#!/usr/bin/env python3\nprint('{}')\n"},
			manifest:   map[string]string{packageGuardPath: "#!/usr/bin/env python3\nprint('{}')\n"},
			want:       LayerDisabled,
			wantReason: "modified from the generated version",
		},
		{
			name:       "state_records_replaced_guard",
			files:      guardedFiles(map[string]string{packageGuardPath: "import sys\nsys.exit(0)\n"}),
			want:       LayerDisabled,
			wantReason: "modified from the generated version",
		},
		{
			name:  "state_older_than_template",
			files: guardedFiles(map[string]string{packageGuardPath: pristineGuard + "# older template\n"}),
			disk:  map[string]string{packageGuardPath: pristineGuard},
			want:  LayerEnabled,
		},
		{
			// The content matches, line endings aside, but outside Windows the
			// kernel cannot start a script with a CRLF interpreter line.
			name:       "crlf_checkout",
			files:      guardedFiles(nil),
			disk:       map[string]string{packageGuardPath: strings.ReplaceAll(pristineGuard, "\n", "\r\n")},
			want:       crlfGuardStatus,
			wantReason: crlfGuardReason,
		},
		{
			name:       "no_generator_template",
			files:      guardedFiles(nil),
			noTemplate: true,
			want:       LayerDisabled,
			wantReason: "not verified",
		},
		{
			name:       "shell_prefix_in_settings_local",
			files:      guardedFiles(map[string]string{localPath: `{"env": {"CLAUDE_CODE_SHELL_PREFIX": "true"}}`}),
			want:       LayerDisabled,
			wantReason: `CLAUDE_CODE_SHELL_PREFIX="true" (set in ` + localPath + ")",
		},
		{
			name:       "pythonpath_in_settings_local",
			files:      guardedFiles(map[string]string{localPath: `{"env": {"PYTHONPATH": ".claude/rules"}}`}),
			want:       LayerDisabled,
			wantReason: `PYTHONPATH=".claude/rules" (set in ` + localPath + ")",
		},
		{
			name:       "bash_env_in_settings_local",
			files:      guardedFiles(map[string]string{localPath: `{"env": {"BASH_ENV": "x"}}`}),
			want:       LayerDisabled,
			wantReason: `BASH_ENV="x" (set in ` + localPath + ")",
		},
		{
			name:       "ld_preload_in_settings_local",
			files:      guardedFiles(map[string]string{localPath: `{"env": {"LD_PRELOAD": "/tmp/x.so"}}`}),
			want:       LayerDisabled,
			wantReason: `LD_PRELOAD="/tmp/x.so" (set in ` + localPath + ")",
		},
		{
			name:       "windows_path_spelling",
			files:      guardedFiles(map[string]string{localPath: `{"env": {"Path": "/tmp/evil"}}`}),
			want:       LayerDisabled,
			wantReason: `Path="/tmp/evil" (set in ` + localPath + ")",
		},
		{
			name:       "home_in_settings_json",
			files:      guardedFiles(map[string]string{".claude/settings.json": strings.Replace(settingsWithPackageGuard, "{", `{"env": {"HOME": "fakehome"}, `, 1)}),
			want:       LayerDisabled,
			wantReason: `HOME="fakehome" (set in .claude/settings.json)`,
		},
		{
			name:       "git_bash_path_in_settings_local",
			files:      guardedFiles(map[string]string{localPath: `{"env": {"CLAUDE_CODE_GIT_BASH_PATH": "tools/bash.exe"}}`}),
			want:       LayerDisabled,
			wantReason: `CLAUDE_CODE_GIT_BASH_PATH="tools/bash.exe"`,
		},
		{
			name:  "inert_env_in_settings_local",
			files: guardedFiles(map[string]string{localPath: `{"env": {"TOOL_GATES_DENIED": "WebFetch"}}`}),
			want:  LayerEnabled,
		},
		{
			name:  "shell_prefix_empty",
			files: guardedFiles(map[string]string{localPath: `{"env": {"CLAUDE_CODE_SHELL_PREFIX": ""}}`}),
			want:  LayerEnabled,
		},
		{
			name: "hook_args_exec_form",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, `{"type": "command",`, `{"type": "command", "args": [],`, 1),
			}),
			want:       LayerDisabled,
			wantReason: "not registered as a blocking PreToolUse hook matching Bash",
		},
		{
			name: "hook_shell_powershell",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, `{"type": "command",`, `{"type": "command", "shell": "powershell",`, 1),
			}),
			want:       LayerDisabled,
			wantReason: "not registered as a blocking PreToolUse hook matching Bash",
		},
		{
			name: "matcher_leading_close_bracket",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, guardMatcher, `"matcher": "[]]?Bash"`, 1),
			}),
			want:       LayerDisabled,
			wantReason: "not registered as a blocking PreToolUse hook matching Bash",
		},
		{
			name:       "manifest_matches_but_guard_edited",
			files:      guardedFiles(nil),
			disk:       map[string]string{packageGuardPath: ""},
			manifest:   map[string]string{packageGuardPath: pristineGuard},
			want:       LayerDisabled,
			wantReason: "modified from the generated version",
		},
		{
			name:     "manifest_unparseable_falls_back_to_state",
			files:    guardedFiles(nil),
			manifest: map[string]string{"../escape": pristineGuard},
			want:     LayerEnabled,
		},
		{
			name:       "settings_unparseable",
			files:      guardedFiles(map[string]string{localPath: `{"hooks": `}),
			want:       LayerDisabled,
			wantReason: localPath,
		},
		{
			name:       "attach_guard_not_enabled",
			files:      guardedFiles(nil),
			tools:      map[string]bool{},
			want:       LayerDisabled,
			wantReason: "attach-guard not enabled",
		},
		{
			name:       "guard_absent",
			files:      map[string]string{".claude/settings.json": settingsWithPackageGuard},
			want:       LayerDisabled,
			wantReason: "package-guard.py not present",
		},
		{
			name: "guard_under_decoy_cased_key",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, `"hooks"`, `"Hooks"`, 1),
			}),
			want: LayerDisabled,
		},
		{
			name: "script_path_substring_only",
			files: guardedFiles(map[string]string{
				".claude/settings.json": strings.Replace(settingsWithPackageGuard, "package-guard.py", "package-guard.py.bak", 1),
			}),
			want: LayerDisabled,
		},
		{
			name:      "project_path_empty",
			files:     guardedFiles(nil),
			noProject: true,
			want:      LayerDisabled,
		},
		{
			name: "registered_in_settings_local_only",
			files: guardedFiles(map[string]string{
				".claude/settings.json": `{"permissions": {"deny": []}}`,
				localPath:               settingsWithPackageGuard,
			}),
			want: LayerEnabled,
		},
		{
			name:  "decoy_key_in_settings_local",
			files: guardedFiles(map[string]string{localPath: `{"DisableAllHooks": true}`}),
			want:  LayerEnabled,
		},
		{
			name:  "pristine",
			files: guardedFiles(nil),
			want:  LayerEnabled,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir, genState := writeProjectFiles(t, tt.files)
			if tt.guardMode != 0 {
				fs := genState.Files[packageGuardPath]
				fs.Mode = tt.guardMode
				genState.Files[packageGuardPath] = fs
			}
			for rel, content := range tt.disk {
				writeFile(t, dir, rel, content)
			}
			if tt.manifest != nil {
				writeManifest(t, dir, tt.manifest)
			}
			if tt.noProject {
				dir = ""
			}
			tools := tt.tools
			if tools == nil {
				tools = attachGuard
			}
			opts := testAssessOpts
			if tt.noTemplate {
				opts = AssessOptions{}
			}
			cov := AssessDefenseLayers(dir, opts, tools, types.DetectedProject{}, genState, 3)
			for _, name := range guardLayers {
				got := layerByName(t, cov, name)
				if got.Status != tt.want {
					t.Errorf("%s: status = %q (%s), want %q", name, got.Status, got.Reason, tt.want)
				}
				if !strings.Contains(got.Reason, tt.wantReason) {
					t.Errorf("%s: reason = %q, want it to contain %q", name, got.Reason, tt.wantReason)
				}
			}
			// Credit is given from the project's settings files alone, and says so.
			if got := layerByName(t, cov, "pretooluse-hooks"); got.Status == LayerEnabled &&
				!strings.Contains(got.Reason, "user/managed settings not inspected") {
				t.Errorf("pretooluse-hooks: reason = %q, want it to note user/managed settings are not inspected", got.Reason)
			}
			// No Grype is configured, so only the guard can scan installs.
			vuln := layerByName(t, cov, "vulnerability-scanning")
			if gotEnabled := vuln.Status == LayerEnabled; gotEnabled != (tt.want == LayerEnabled) {
				t.Errorf("vulnerability-scanning: status = %q (%s), want enabled=%v", vuln.Status, vuln.Reason, tt.want == LayerEnabled)
			}
		})
	}
}

// writeT1GuardedProject writes a supply-chain-only (T1) project generated
// with attach-guard: a pristine guard registered in settings.json and the
// lock file audit hook, all recorded in the state file with real hashes.
func writeT1GuardedProject(t *testing.T) string {
	t.Helper()
	dir, st := writeProjectFiles(t, guardedFiles(map[string]string{
		"CLAUDE.md":               "# Project\n",
		".pre-commit-config.yaml": preCommitWithLockAudit,
	}))
	st.QsdevVersion = "1.0.0"
	st.EnabledTools = map[string]bool{"attach-guard": true}
	writeState(t, dir, filepath.Join(".devinit", ".qsdev-init-state.yaml"), st)
	writeFile(t, dir, ".qsdev.yaml", "version: 1\ntier: supply-chain-only\n")
	return dir
}

// gradeAtMost reports whether grade is worst or below it.
func gradeAtMost(grade, worst string) bool {
	order := make([]string, 0, len(gradeTable)+1)
	for _, e := range gradeTable {
		order = append(order, e.grade)
	}
	order = append(order, "F")
	return slices.Index(order, grade) >= slices.Index(order, worst)
}

// TestGuardEffective_UnloadableSibling pins that a PreToolUse or
// PermissionRequest entry Claude Code cannot load, sitting beside an intact
// guard registration, turns the guard off: Claude Code then applies nothing
// from that settings file. The same entry in settings.local.json drops only
// that file, so the committed guard stays in force.
func TestGuardEffective_UnloadableSibling(t *testing.T) {
	t.Parallel()
	sibling := func(event, entry string) string {
		return strings.Replace(settingsWithPackageGuard, `{"hooks": {"PreToolUse": [`,
			`{"hooks": {"`+event+`": [`+entry+`], "PreToolUse": [`, 1)
	}
	withPreToolUse := func(entry string) string {
		return strings.Replace(settingsWithPackageGuard, `]}]}}`, `]}, `+entry+`]}}`, 1)
	}
	tests := []struct {
		name     string
		settings string
	}{
		{"hook if number", withPreToolUse(`{"matcher": "Write", "hooks": [{"type": "command", "command": "true", "if": 1}]}`)},
		{"hook command not a string", withPreToolUse(`{"matcher": "Write", "hooks": [{"type": "command", "command": true}]}`)},
		{"hook timeout string", withPreToolUse(`{"matcher": "Write", "hooks": [{"type": "command", "command": "true", "timeout": "x"}]}`)},
		{"hook not an object", withPreToolUse(`{"matcher": "Write", "hooks": [5]}`)},
		{"hook type unknown", withPreToolUse(`{"matcher": "Write", "hooks": [{"type": "bogus"}]}`)},
		{"hook command missing", withPreToolUse(`{"matcher": "Write", "hooks": [{"type": "command"}]}`)},
		{"hooks not an array", withPreToolUse(`{"matcher": "Write", "hooks": {"type": "command", "command": "true"}}`)},
		{"matcher not a string", withPreToolUse(`{"matcher": ["Write"], "hooks": []}`)},
		{"entry not an object", withPreToolUse(`"Write"`)},
		{"permission request entry", sibling("PermissionRequest", `{"hooks": [{"type": "command", "command": "true", "once": "yes"}]}`)},
		{"permission request not an array", strings.Replace(settingsWithPackageGuard, `{"hooks": {`, `{"hooks": {"PermissionRequest": {}, `, 1)},
		{"guard hooks nested under another event", sibling("PostToolUse", `{"matcher": "*", "hooks": [], "PreToolUse": [{"hooks": []}]}`)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			for _, file := range []string{".claude/settings.json", ".claude/settings.local.json"} {
				files := guardedFiles(map[string]string{file: tt.settings})
				if file != ".claude/settings.json" {
					// The local file carries only the bad entry's file; the
					// committed one keeps the guard.
					files = guardedFiles(map[string]string{file: strings.Replace(tt.settings, "package-guard.py", "other.py", 1)})
				}
				dir, genState := writeProjectFiles(t, files)
				cov := AssessDefenseLayers(dir, testAssessOpts, map[string]bool{"attach-guard": true}, types.DetectedProject{}, genState, 3)
				got := layerByName(t, cov, "pretooluse-hooks")
				if file == ".claude/settings.json" {
					if got.Status != LayerDisabled || !strings.Contains(got.Reason, "Claude Code does not load .claude/settings.json") {
						t.Errorf("sibling in %s: pretooluse-hooks = %q (%s), want disabled because the file is not loaded", file, got.Status, got.Reason)
					}
					continue
				}
				if got.Status != LayerEnabled {
					t.Errorf("sibling in %s: pretooluse-hooks = %q (%s), want enabled: only the local file is dropped", file, got.Status, got.Reason)
				}
			}
		})
	}
}

// TestAssess_GuttedGuard is the K1/K2 regression test: an emptied guard, or
// hooks switched off in settings.local.json or the user settings.json, leaves
// every layer the guard provides disabled, and the gutted project no longer
// grades well.
func TestAssess_GuttedGuard(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		disk      map[string]string
		user      string // written as the user settings.json after the pristine run
		wantGrade string // worst acceptable grade; "" skips the grade check
	}{
		{name: "K1 emptied guard", disk: map[string]string{packageGuardPath: ""}, wantGrade: "C"},
		{name: "K2 hooks disabled locally", disk: map[string]string{".claude/settings.local.json": `{"disableAllHooks":true}`}},
		{name: "K2 hooks disabled for the user", user: `{"disableAllHooks":true}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := writeT1GuardedProject(t)
			opts := AssessOptions{ClaudeUserDir: t.TempDir(), PackageGuard: []byte(pristineGuard)}
			pristine, err := Assess(dir, opts)
			if err != nil {
				t.Fatalf("Assess: %v", err)
			}
			for _, name := range append(slices.Clone(guardLayers), "vulnerability-scanning") {
				if got := layerByName(t, pristine.Defense, name); got.Status != LayerEnabled {
					t.Fatalf("pristine fixture: %s = %q (%s), want enabled", name, got.Status, got.Reason)
				}
			}

			for rel, content := range tt.disk {
				writeFile(t, dir, rel, content)
			}
			if tt.user != "" {
				writeFile(t, opts.ClaudeUserDir, "settings.json", tt.user)
			}
			report, err := Assess(dir, opts)
			if err != nil {
				t.Fatalf("Assess: %v", err)
			}
			for _, name := range append(slices.Clone(guardLayers), "vulnerability-scanning") {
				if got := layerByName(t, report.Defense, name); got.Status != LayerDisabled {
					t.Errorf("%s = %q (%s), want disabled", name, got.Status, got.Reason)
				}
			}
			if got := layerByName(t, report.Defense, "lock-file-enforcement"); got.Status == LayerEnabled {
				t.Errorf("lock-file-enforcement = enabled (%s) without the guard in force", got.Reason)
			}
			t.Logf("grade %s (%.1f), pristine %s (%.1f)", ScoreToGrade(report.Score.Total), report.Score.Total,
				ScoreToGrade(pristine.Score.Total), pristine.Score.Total)
			if grade := ScoreToGrade(report.Score.Total); tt.wantGrade != "" && !gradeAtMost(grade, tt.wantGrade) {
				t.Errorf("grade = %s (%.1f), want %s or worse", grade, report.Score.Total, tt.wantGrade)
			}
		})
	}
}

// TestGuardEffective_ModeChanged pins that posture judges "unmodified" as
// check does: a guard whose permission bits differ from the recorded ones is
// not credited (modes are not compared on Windows).
func TestGuardEffective_ModeChanged(t *testing.T) {
	t.Parallel()
	dir, genState := writeProjectFiles(t, guardedFiles(nil))
	fs := genState.Files[packageGuardPath]
	fs.Mode = 0o755 // fixtures write 0o644: the guard lost its exec bit
	genState.Files[packageGuardPath] = fs
	cov := AssessDefenseLayers(dir, testAssessOpts, map[string]bool{"attach-guard": true}, types.DetectedProject{}, genState, 3)
	want := LayerDisabled
	if runtime.GOOS == "windows" {
		want = LayerEnabled
	}
	for _, name := range guardLayers {
		if got := layerByName(t, cov, name); got.Status != want {
			t.Errorf("%s: status = %q (%s), want %q", name, got.Status, got.Reason, want)
		}
	}
}
