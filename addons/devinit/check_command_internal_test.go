package devinit

import (
	"encoding/json"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/check"
	"github.com/Quantum-Serendipity/qsdev/internal/claudesettings"
	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/internal/shebang"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func denySetRules(t *testing.T, set string) []string {
	t.Helper()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	rules := cat.PermissionDenyRules(set)
	if len(rules) == 0 {
		t.Fatalf("catalog deny set %q is empty", set)
	}
	return rules
}

// TestRequiredDenyRules_CoverPresetDenySets is the regression test for
// `qsdev check` enforcing only two deny sets: removing a sudo_prefix or
// sandbox_config_edits rule from settings.json went unnoticed.
func TestRequiredDenyRules_CoverPresetDenySets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		answers     types.WizardAnswers
		cfg         *types.QsdevConfig
		mustInclude []string
		mustExclude []string
	}{
		{
			name:        "standard preset requires every standard deny set",
			answers:     types.WizardAnswers{PermissionLevel: "standard"},
			mustInclude: []string{"sudo_prefix", "sandbox_config_edits", "shell_wrapping", "destructive_ops", "pipe_to_shell"},
		},
		{
			name:        "no answers or config falls back to standard",
			mustInclude: []string{"sudo_prefix", "sandbox_config_edits"},
		},
		{
			name:        "supply-chain-only preset does not require sets it never generates",
			answers:     types.WizardAnswers{PermissionLevel: "supply-chain-only"},
			mustInclude: []string{"sudo_prefix", "pipe_to_shell"},
			mustExclude: []string{"destructive_ops", "sandbox_config_edits"},
		},
		{
			name:        "local answers cannot loosen the committed preset",
			answers:     types.WizardAnswers{PermissionLevel: "supply-chain-only", Tier: "standard"},
			cfg:         &types.QsdevConfig{ClaudeCode: types.ClaudeCodeConfig{PermissionLevel: "standard"}},
			mustInclude: []string{"sudo_prefix", "sandbox_config_edits", "destructive_ops"},
		},
		{
			name:        "local answers may tighten the committed preset",
			answers:     types.WizardAnswers{PermissionLevel: "minimal"},
			cfg:         &types.QsdevConfig{ClaudeCode: types.ClaudeCodeConfig{PermissionLevel: "standard"}},
			mustInclude: []string{"sudo_prefix", "sandbox_config_edits", "destructive_ops"},
		},
		{
			name:        "tier from project config selects its default preset",
			cfg:         &types.QsdevConfig{Tier: "supply-chain-only"},
			mustInclude: []string{"pipe_to_shell"},
			mustExclude: []string{"sandbox_config_edits"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := requiredDenyRules(tt.answers, tt.cfg)
			if err != nil {
				t.Fatal(err)
			}
			for _, set := range tt.mustInclude {
				for _, rule := range denySetRules(t, set) {
					if !slices.Contains(got, rule) {
						t.Errorf("required rules missing %q from deny set %s", rule, set)
					}
				}
			}
			for _, set := range tt.mustExclude {
				for _, rule := range denySetRules(t, set) {
					if slices.Contains(got, rule) {
						t.Errorf("required rules include %q from deny set %s, which this preset does not generate", rule, set)
					}
				}
			}
		})
	}
}

// TestRequiredDenyRules_MatchGeneratedSettings verifies that check never
// requires a rule the generator does not write, so a freshly generated
// project passes deny_rules_present for every permission preset.
func TestRequiredDenyRules_MatchGeneratedSettings(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	for _, preset := range cat.PermissionPresets() {
		t.Run(preset, func(t *testing.T) {
			t.Parallel()
			answers := types.WizardAnswers{ClaudeCode: true, PermissionLevel: preset}
			file, err := claudecode.GenerateSettings(answers, ecosystem.DefaultRegistry(), claudecode.Config{})
			if err != nil {
				t.Fatal(err)
			}
			var settings struct {
				Permissions struct {
					Deny []string `json:"deny"`
				} `json:"permissions"`
			}
			if err := json.Unmarshal(file.Content, &settings); err != nil {
				t.Fatal(err)
			}
			required, err := requiredDenyRules(answers, nil)
			if err != nil {
				t.Fatal(err)
			}
			for _, rule := range required {
				if !slices.Contains(settings.Permissions.Deny, rule) {
					t.Errorf("check requires %q but generated settings.json does not contain it", rule)
				}
			}
		})
	}
}

// TestCheckCmd_FailsWhenGuardHooksStripped drives `qsdev check` against a
// generated project whose settings.json lost its guard hooks and was switched
// to bypassPermissions: the report must carry high-severity posture failures
// and the command must exit non-zero at --audit-level low.
// postureFailures parses the JSON report of `qsdev check --format json` and
// returns the names of the failing Claude settings posture results.
func postureFailures(t *testing.T, out string) []string {
	t.Helper()
	var report check.CheckReport
	// The JSON report may be followed by cobra's error output.
	if err := json.NewDecoder(strings.NewReader(out[strings.Index(out, "{"):])).Decode(&report); err != nil {
		t.Fatalf("parsing report: %v\n%s", err, out)
	}
	var names []string
	if !slices.ContainsFunc(report.Checks, func(c check.CheckResult) bool { return strings.HasPrefix(c.Name, "claude_") }) {
		t.Fatalf("report has no Claude settings posture result:\n%s", out)
	}
	for _, c := range report.Checks {
		if strings.HasPrefix(c.Name, "claude_") && c.Status == check.StatusFail {
			names = append(names, c.Name)
		}
	}
	return names
}

// stubHookProgramsOnPath prepends to PATH a directory holding an empty stub
// for each program the generated hooks in dir look up on PATH: the app binary
// and each hook script's env interpreter. `qsdev check` only looks them up,
// never runs them, so the posture result does not depend on what the machine
// has installed.
func stubHookProgramsOnPath(t *testing.T, dir string) {
	t.Helper()
	names := []string{branding.Get().AppName}
	err := filepath.WalkDir(filepath.Join(dir, ".claude", "hooks"), func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		if line, err := shebang.Read(p); err == nil && line.ViaEnv() && line.Program() != "" {
			names = append(names, line.Program())
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	for _, name := range names {
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		if err := os.WriteFile(filepath.Join(bin, name), nil, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
}

func TestCheckCmd_FailsWhenGuardHooksStripped(t *testing.T) {
	dir := initLifecycleProject(t)
	stubHookProgramsOnPath(t, dir)

	out, _ := runLifecycleCmd(t, dir, checkCmd(), "--format", "json", "--audit-level", "low")
	if failed := postureFailures(t, out); len(failed) != 0 {
		t.Fatalf("freshly generated project fails posture checks: %v", failed)
	}

	editSettings(t, dir, func(settings map[string]any) {
		delete(settings, "hooks")
		perms := settings["permissions"].(map[string]any)
		perms["defaultMode"] = "bypassPermissions"
		delete(perms, "disableBypassPermissionsMode")
	})

	out, err := runLifecycleCmd(t, dir, checkCmd(), "--format", "json", "--audit-level", "low")
	if err == nil {
		t.Fatalf("check passed with guard hooks stripped:\n%s", out)
	}
	failed := postureFailures(t, out)
	for _, want := range []string{"claude_bypass_permissions_mode", "claude_disable_bypass_missing", "claude_hook_missing"} {
		if !slices.Contains(failed, want) {
			t.Errorf("report lacks failing %s; posture failures: %v", want, failed)
		}
	}

	// A CI checkout has no (gitignored) answers file: the expected hooks
	// come from the committed .qsdev.yaml instead.
	if err := os.Remove(answers.PrimaryPath(dir)); err != nil {
		t.Fatal(err)
	}
	out, err = runLifecycleCmd(t, dir, checkCmd(), "--format", "json", "--audit-level", "low")
	if err == nil || !slices.Contains(postureFailures(t, out), "claude_hook_missing") {
		t.Errorf("without saved answers the stripped hooks went unreported (err=%v):\n%s", err, out)
	}
}

// TestCheckCmd_FailsWhenHookDoesNotResolve guards XS-WS1 A7: a registered
// hook that cannot start (its script's interpreter or its program is not on
// PATH) exits 127, which Claude Code treats as a non-blocking error, so
// `qsdev check` must fail on it.
func TestCheckCmd_FailsWhenHookDoesNotResolve(t *testing.T) {
	tests := []struct {
		name string
		edit func(t *testing.T, dir string)
	}{
		{"script interpreter", func(t *testing.T, dir string) {
			t.Helper()
			rel := registeredHookScript(t, dir)
			_, rest, _ := strings.Cut(readProjectFile(t, dir, rel), "\n")
			if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(rel)), []byte("#!/usr/bin/env qsdev-xsws1-no-such-interpreter\n"+rest), 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"bare program", func(t *testing.T, dir string) {
			t.Helper()
			editSettings(t, dir, func(settings map[string]any) {
				hook := settings["hooks"].(map[string]any)["PreToolUse"].([]any)[0].(map[string]any)["hooks"].([]any)[0].(map[string]any)
				hook["command"] = "qsdev-xsws1-no-such-bin selfprotect"
			})
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := initLifecycleProject(t)
			stubHookProgramsOnPath(t, dir)
			tt.edit(t, dir)

			out, err := runLifecycleCmd(t, dir, checkCmd(), "--format", "json")
			if err == nil {
				t.Fatalf("check passed with a hook that cannot run:\n%s", out)
			}
			if failed := postureFailures(t, out); !slices.Contains(failed, "claude_hook_unresolvable") {
				t.Errorf("report lacks failing claude_hook_unresolvable; posture failures: %v", failed)
			}
		})
	}
}

// registeredHookScript returns the first project hook script the generated
// settings.json registers for PreToolUse.
func registeredHookScript(t *testing.T, dir string) string {
	t.Helper()
	settings, err := claudesettings.Parse([]byte(readProjectFile(t, dir, ".claude/settings.json")))
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range settings.Hooks[claudesettings.EventPreToolUse] {
		for _, h := range m.Hooks {
			if script := claudesettings.ScriptRe.FindString(h.Command); script != "" {
				return script
			}
		}
	}
	t.Fatal("generated settings.json registers no PreToolUse hook script")
	return ""
}

// editSettings rewrites .claude/settings.json in dir through edit.
func editSettings(t *testing.T, dir string, edit func(map[string]any)) {
	t.Helper()
	var settings map[string]any
	if err := json.Unmarshal([]byte(readProjectFile(t, dir, ".claude/settings.json")), &settings); err != nil {
		t.Fatal(err)
	}
	edit(settings)
	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".claude", "settings.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// fileStateFailures runs `qsdev check --format json` in dir and returns the
// names of the failing generated-file-state results.
func fileStateFailures(t *testing.T, dir string) ([]string, error) {
	t.Helper()
	out, err := runLifecycleCmd(t, dir, checkCmd(), "--format", "json", "--audit-level", "medium")
	var report check.CheckReport
	// The JSON report may be followed by cobra's error output.
	if jerr := json.NewDecoder(strings.NewReader(out[strings.Index(out, "{"):])).Decode(&report); jerr != nil {
		t.Fatalf("parsing report: %v\n%s", jerr, out)
	}
	var names []string
	for _, c := range report.Checks {
		if c.Category == check.CategoryFileState && c.Status == check.StatusFail {
			names = append(names, c.Name)
		}
	}
	return names, err
}

// TestCheckCmd_CICheckoutDetectsGeneratedFileDrift is the F349 regression: a
// CI checkout lacks the gitignored generation state, so `qsdev check` must
// verify machine-owned generated files against the committed manifest that
// init writes, and fail when one is edited, deleted, or the manifest is gone.
func TestCheckCmd_CICheckoutDetectsGeneratedFileDrift(t *testing.T) {
	dir := initLifecycleProject(t)

	manifest, err := state.LoadManifest(filepath.Join(dir, state.ManifestFile()))
	if err != nil {
		t.Fatalf("init did not write the manifest: %v", err)
	}
	if want := state.BuildManifest(loadProjectState(t, dir)); !maps.Equal(manifest, want) {
		t.Fatalf("manifest disagrees with the init state:\n got %v\nwant %v", manifest, want)
	}
	const hook = ".claude/hooks/package-guard.py"
	if _, ok := manifest[hook]; !ok {
		t.Fatalf("manifest does not list %s: %v", hook, manifest)
	}

	// A clean CI checkout: the gitignored state directory does not exist.
	if err := os.RemoveAll(filepath.Join(dir, branding.Get().StateDir)); err != nil {
		t.Fatal(err)
	}
	if failed, _ := fileStateFailures(t, dir); len(failed) != 0 {
		t.Fatalf("clean checkout fails generated-file checks: %v", failed)
	}

	hookPath := filepath.Join(dir, filepath.FromSlash(hook))
	original, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(hookPath, []byte("import sys\nsys.exit(0)\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	failed, err := fileStateFailures(t, dir)
	if err == nil || !slices.Contains(failed, "file_unmodified_"+hook) {
		t.Errorf("edited hook went unreported on a CI checkout (err=%v, failures=%v)", err, failed)
	}

	if err := os.Remove(hookPath); err != nil {
		t.Fatal(err)
	}
	failed, err = fileStateFailures(t, dir)
	if err == nil || !slices.Contains(failed, "file_exists_"+hook) {
		t.Errorf("deleted hook went unreported on a CI checkout (err=%v, failures=%v)", err, failed)
	}

	if err := os.WriteFile(hookPath, original, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(dir, state.ManifestFile())); err != nil {
		t.Fatal(err)
	}
	failed, err = fileStateFailures(t, dir)
	if err == nil || !slices.Contains(failed, "generated_manifest") {
		t.Errorf("missing manifest went unreported (err=%v, failures=%v)", err, failed)
	}
}

// TestManifestFollowsLifecycleCommands checks that every command that rewrites
// the generation state keeps the committed manifest in step with it, and that
// repair writes a manifest for a project initialized before it existed.
func TestManifestFollowsLifecycleCommands(t *testing.T) {
	dir := initLifecycleProject(t)
	manifestPath := filepath.Join(dir, state.ManifestFile())
	assertInStep := func(step string) {
		t.Helper()
		m, err := state.LoadManifest(manifestPath)
		if err != nil {
			t.Fatalf("after %s: %v", step, err)
		}
		if want := state.BuildManifest(loadProjectState(t, dir)); !maps.Equal(m, want) {
			t.Errorf("after %s the manifest disagrees with the state:\n got %v\nwant %v", step, m, want)
		}
	}

	assertInStep("init")
	mustDisable(t, dir, "gitleaks", "--force")
	assertInStep("disable")
	mustEnable(t, dir, "gitleaks", "--force")
	assertInStep("enable")

	if err := os.Remove(manifestPath); err != nil {
		t.Fatal(err)
	}
	// Repair also reports environment findings it cannot fix in a test
	// project (no git repository, no go.sum), which make it exit non-zero;
	// the manifest is written regardless.
	if out, err := runLifecycleCmd(t, dir, repairCmd()); err != nil {
		t.Logf("repair: %v\n%s", err, out)
	}
	assertInStep("repair")
}

// TestCheckCmd_ToolGatesPolicy guards W046 end to end: tool-gates enabled
// without .qsdev.yaml hooks.tool_gates is reported as "enabled (no policy)";
// once the policy is committed, update hands it to the hook through
// settings.json env, and dropping that env fails the check.
func TestCheckCmd_ToolGatesPolicy(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(rel)), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("go.mod", "module example.com/tg\n\ngo 1.24\n")
	if out, err := executeInitCmd(t, dir, "--yes", "--lang", "go", "--tier", "full", "--claude-hooks", "safety-block,tool-gates"); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}

	report := func() check.CheckReport {
		t.Helper()
		out, _ := runLifecycleCmd(t, dir, checkCmd(), "--format", "json", "--audit-level", "low")
		var r check.CheckReport
		if err := json.NewDecoder(strings.NewReader(out[strings.Index(out, "{"):])).Decode(&r); err != nil {
			t.Fatalf("parsing report: %v\n%s", err, out)
		}
		return r
	}
	has := func(r check.CheckReport, name string, status check.CheckStatus) bool {
		return slices.ContainsFunc(r.Checks, func(c check.CheckResult) bool { return c.Name == name && c.Status == status })
	}

	if r := report(); !has(r, "claude_hook_no_policy", check.StatusWarn) {
		t.Fatalf("tool-gates without policy not reported: %+v", r.Checks)
	}

	write(".qsdev.yaml", readProjectFile(t, dir, ".qsdev.yaml")+"hooks:\n  tool_gates:\n    denied: [WebFetch, \"mcp__github__*\"]\n")
	// A committed policy that settings.json does not carry yet is not in
	// force: the check fails until 'qsdev init --update' writes it.
	if r := report(); !has(r, "claude_hook_env_changed", check.StatusFail) || has(r, "claude_hook_no_policy", check.StatusWarn) {
		t.Fatalf("committed but ungenerated tool-gates policy not reported: %+v", r.Checks)
	}
	if out, err := executeInitCmd(t, dir, "--update"); err != nil {
		t.Fatalf("update: %v\n%s", err, out)
	}
	var settings map[string]any
	if err := json.Unmarshal([]byte(readProjectFile(t, dir, ".claude/settings.json")), &settings); err != nil {
		t.Fatal(err)
	}
	env, _ := settings["env"].(map[string]any)
	if got := env[claudecode.ToolGatesDeniedEnv]; got != "WebFetch,mcp__github__*" {
		t.Fatalf("settings.json env %s = %v, want the committed deny list", claudecode.ToolGatesDeniedEnv, got)
	}
	if r := report(); has(r, "claude_hook_no_policy", check.StatusWarn) || has(r, "claude_hook_env_changed", check.StatusFail) {
		t.Fatalf("configured tool-gates still reported: %+v", r.Checks)
	}

	delete(settings, "env")
	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	write(".claude/settings.json", string(data))
	if r := report(); !has(r, "claude_hook_env_changed", check.StatusFail) {
		t.Errorf("dropped tool-gates policy env not reported: %+v", r.Checks)
	}
}

// TestRunCheck_ExpectedSettingsEnforceAlwaysOn is the U28-V01 regression:
// saved answers that dropped the safety block must not also drop the
// package-guard registration from the settings check expects. The loaded
// answers are reconciled against the committed tools block, as every
// generation path does, before the expected settings are generated, so the
// stripped always-on hook is reported. An answers-only opt-out (the tool off
// and the opt-out recorded in the answers file, but no committed
// tools.disabled entry) is not an opt-out either.
func TestRunCheck_ExpectedSettingsEnforceAlwaysOn(t *testing.T) {
	tests := []struct {
		name   string
		tamper func(a *types.WizardAnswers)
	}{
		{name: "safety block off", tamper: func(a *types.WizardAnswers) { a.Hooks.SafetyBlock = false }},
		{name: "answers-only opt-out", tamper: func(a *types.WizardAnswers) {
			a.EnabledTools[toolreg.ToolAttachGuard] = false
			a.Hooks.SafetyBlockOptOut = true
			a.Hooks.SafetyBlock = false
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := initLifecycleProject(t)

			saved, err := answers.LoadPrimary(dir)
			if err != nil {
				t.Fatal(err)
			}
			tt.tamper(&saved)
			if err := answers.SavePrimary(dir, saved); err != nil {
				t.Fatal(err)
			}

			const guard = "package-guard.py"
			settingsPath := filepath.Join(dir, ".claude", "settings.json")
			var settings map[string]any
			if err := json.Unmarshal([]byte(readProjectFile(t, dir, ".claude/settings.json")), &settings); err != nil {
				t.Fatal(err)
			}
			if removed := stripHookCommands(settings, guard); removed == 0 {
				t.Fatalf("generated settings.json registers no %s hook", guard)
			}
			data, err := json.Marshal(settings)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
				t.Fatal(err)
			}

			report := runCheckJSON(t, dir)
			if !slices.ContainsFunc(report.Checks, func(c check.CheckResult) bool {
				return c.Name == "claude_hook_missing" && c.Status == check.StatusFail &&
					strings.Contains(c.Metadata["command"], guard)
			}) {
				t.Errorf("check did not report the stripped %s registration: %+v", guard, report.Checks)
			}
		})
	}
}

// stripHookCommands removes every hook whose command contains substr from a
// decoded settings.json and returns how many it removed.
func stripHookCommands(settings map[string]any, substr string) int {
	hooks, _ := settings["hooks"].(map[string]any)
	removed := 0
	for _, matchers := range hooks {
		list, _ := matchers.([]any)
		for _, m := range list {
			entry, _ := m.(map[string]any)
			inner, _ := entry["hooks"].([]any)
			kept := inner[:0]
			for _, h := range inner {
				cmd, _ := h.(map[string]any)["command"].(string)
				if strings.Contains(cmd, substr) {
					removed++
					continue
				}
				kept = append(kept, h)
			}
			entry["hooks"] = kept
		}
	}
	return removed
}

// TestRunCheck_LocalAnswersCannotNarrowRequiredTools verifies check takes
// the always-on scope (Claude Code on, the tier) from the committed
// .qsdev.yaml, not from the local answers file: answers that switch Claude
// Code off or lower the tier must not hide a hand-removed always-on tool.
func TestRunCheck_LocalAnswersCannotNarrowRequiredTools(t *testing.T) {
	tests := []struct {
		name   string
		tool   string
		hook   string // a registration to strip, which check must report
		tamper func(a *types.WizardAnswers)
	}{
		{name: "claude code off locally", tool: toolreg.ToolAttachGuard, hook: "package-guard.py", tamper: func(a *types.WizardAnswers) { a.ClaudeCode = false }},
		{name: "tier lowered locally", tool: toolreg.ToolTrailOfBitsSkills, tamper: func(a *types.WizardAnswers) {
			a.Tier = "supply-chain-only"
			a.PermissionLevel = "supply-chain-only"
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := initLifecycleProject(t)

			saved, err := answers.LoadPrimary(dir)
			if err != nil {
				t.Fatal(err)
			}
			tt.tamper(&saved)
			if err := answers.SavePrimary(dir, saved); err != nil {
				t.Fatal(err)
			}
			cfgPath := filepath.Join(dir, ".qsdev.yaml")
			cfg := readProjectFile(t, dir, ".qsdev.yaml")
			stripped := strings.ReplaceAll(cfg, "        - "+tt.tool+"\n", "")
			stripped = strings.ReplaceAll(stripped, "    - "+tt.tool+"\n", "")
			if stripped == cfg {
				t.Fatalf(".qsdev.yaml has no %s entry:\n%s", tt.tool, cfg)
			}
			if err := os.WriteFile(cfgPath, []byte(stripped), 0o644); err != nil {
				t.Fatal(err)
			}
			if tt.hook != "" {
				editSettings(t, dir, func(settings map[string]any) {
					if stripHookCommands(settings, tt.hook) == 0 {
						t.Fatalf("generated settings.json registers no %s hook", tt.hook)
					}
				})
			}

			report := runCheckJSON(t, dir)
			if tt.hook != "" && !slices.ContainsFunc(report.Checks, func(c check.CheckResult) bool {
				return c.Name == "claude_hook_missing" && c.Status == check.StatusFail &&
					strings.Contains(c.Metadata["command"], tt.hook)
			}) {
				t.Errorf("check did not report the stripped %s registration: %+v", tt.hook, report.Checks)
			}
			want := "tool_missing_" + tt.tool
			if !slices.ContainsFunc(report.Checks, func(c check.CheckResult) bool {
				return c.Name == want && c.Status == check.StatusFail
			}) {
				t.Errorf("check did not fail %s: %+v", want, report.Checks)
			}
		})
	}
}

// TestRunCheck_LocalPermissionLevelCannotNarrowDenyRules verifies that a
// looser permission level in the local answers file does not shrink the deny
// rules check requires below the committed preset: a local level may only
// tighten it, as a local config layer may.
func TestRunCheck_LocalPermissionLevelCannotNarrowDenyRules(t *testing.T) {
	dir := initLifecycleProject(t)

	saved, err := answers.LoadPrimary(dir)
	if err != nil {
		t.Fatal(err)
	}
	saved.PermissionLevel = "supply-chain-only"
	if err := answers.SavePrimary(dir, saved); err != nil {
		t.Fatal(err)
	}
	// destructive_ops is in the committed standard preset but not in
	// supply-chain-only, so only the committed floor requires it.
	stripped := denySetRules(t, "destructive_ops")
	editSettings(t, dir, func(settings map[string]any) {
		perms, _ := settings["permissions"].(map[string]any)
		deny, _ := perms["deny"].([]any)
		kept := slices.DeleteFunc(slices.Clone(deny), func(r any) bool {
			s, _ := r.(string)
			return slices.Contains(stripped, s)
		})
		if len(kept) == len(deny) {
			t.Fatalf("generated settings.json carries no destructive_ops deny rule: %v", deny)
		}
		perms["deny"] = kept
	})

	report := runCheckJSON(t, dir)
	for _, rule := range stripped {
		if !slices.ContainsFunc(report.Checks, func(c check.CheckResult) bool {
			return c.Name == "deny_rule_missing" && c.Status == check.StatusFail && c.Metadata["rule"] == rule
		}) {
			t.Errorf("check did not report the stripped deny rule %q", rule)
		}
	}
}

// TestGuardReplacedAndRehashed drives `qsdev check` and the posture status
// against a generated project whose package-guard.py was replaced and whose
// manifest entry and local state were re-hashed over the replacement, as a
// PR could: the guard is judged against the generator's own content, so
// check fails at --audit-level critical and status credits no guard layer.
// The project as generated passes both.
func TestGuardReplacedAndRehashed(t *testing.T) {
	dir := initLifecycleProject(t)
	stubHookProgramsOnPath(t, dir)
	guardLayers := []string{"pretooluse-hooks", "install-script-blocking"}
	assertGuardLayers := func(want posture.LayerStatus) {
		t.Helper()
		report, err := posture.Assess(dir, postureOptions(posture.AssessOptions{}))
		if err != nil {
			t.Fatal(err)
		}
		for _, name := range guardLayers {
			if l := posture.FindLayerByName(report.Defense.Layers, name); l == nil || l.Status != want {
				t.Errorf("%s = %+v, want %s", name, l, want)
			}
		}
	}
	if out, err := runLifecycleCmd(t, dir, checkCmd(), "--format", "json", "--audit-level", "critical"); err != nil {
		t.Fatalf("freshly generated project fails check at critical: %v\n%s", err, out)
	}
	assertGuardLayers(posture.LayerEnabled)

	replaced := []byte("#!/usr/bin/env python3\nimport sys\nsys.exit(0)\n")
	if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(claudecode.PackageGuardPath)), replaced, 0o755); err != nil {
		t.Fatal(err)
	}
	stateFile := filepath.Join(dir, filepath.FromSlash(state.InitStateFile()))
	st, err := state.LoadStateFromFile(stateFile)
	if err != nil {
		t.Fatal(err)
	}
	fs := st.Files[claudecode.PackageGuardPath]
	fs.Hash = state.ComputeHash(replaced)
	st.Files[claudecode.PackageGuardPath] = fs
	if err := state.SaveStateToFile(stateFile, st); err != nil {
		t.Fatal(err)
	}
	if err := state.WriteManifest(dir, state.BuildManifest(st)); err != nil {
		t.Fatal(err)
	}

	out, err := runLifecycleCmd(t, dir, checkCmd(), "--format", "json", "--audit-level", "critical")
	if err == nil {
		t.Fatalf("check passed at critical with a replaced, re-hashed guard:\n%s", out)
	}
	if !strings.Contains(out, `"file_unmodified_`+claudecode.PackageGuardPath+`"`) {
		t.Errorf("report lacks file_unmodified_%s:\n%s", claudecode.PackageGuardPath, out)
	}
	assertGuardLayers(posture.LayerDisabled)
}

// editProjectConfig rewrites the project's committed config with edit
// applied to its YAML document.
func editProjectConfig(t *testing.T, dir string, edit func(map[string]any)) {
	t.Helper()
	path := filepath.Join(dir, branding.Get().ConfigFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]any
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	edit(cfg)
	if data, err = yaml.Marshal(cfg); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// claudeCodeConfig returns the claude_code section of a config document.
func claudeCodeConfig(t *testing.T, cfg map[string]any) map[string]any {
	t.Helper()
	cc, ok := cfg["claude_code"].(map[string]any)
	if !ok {
		t.Fatalf("config has no claude_code section: %v", cfg)
	}
	return cc
}

// TestCheckCmd_GuardBypassWhenExpectedGenerationFails drives `qsdev check` on
// a CI-like checkout (no local state or answers) whose package guard was
// replaced and re-hashed in the committed manifest, or unregistered, together
// with a committed config edit that stops the generator from producing the
// expected files or turns Claude Code off. Each must still fail at
// --audit-level critical: the guard is judged against the embedded template
// whatever the config says, and a generator that cannot run is itself a
// critical failure rather than a vacuous pass.
func TestCheckCmd_GuardBypassWhenExpectedGenerationFails(t *testing.T) {
	unknownSkill := func(cfg map[string]any) {
		claudeCodeConfig(t, cfg)["skills"] = []any{"no-such-skill"}
	}
	tests := []struct {
		name       string
		gut        bool
		unregister bool
		config     func(map[string]any)
		wantFail   []string
	}{
		{
			name:     "rehash plus unknown skill",
			gut:      true,
			config:   unknownSkill,
			wantFail: []string{"file_unmodified_" + claudecode.PackageGuardPath, "expected_generation_failed"},
		},
		{
			name: "rehash plus claude_code disabled",
			gut:  true,
			config: func(cfg map[string]any) {
				claudeCodeConfig(t, cfg)["enabled"] = false
			},
			wantFail: []string{"file_unmodified_" + claudecode.PackageGuardPath},
		},
		{
			name:       "hook removal plus unknown skill",
			unregister: true,
			config:     unknownSkill,
			wantFail:   []string{"expected_generation_failed"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := freshCloneProject(t)
			stubHookProgramsOnPath(t, dir)
			if tt.gut {
				replaced := []byte("import sys; sys.exit(0)\n")
				if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(claudecode.PackageGuardPath)), replaced, 0o755); err != nil {
					t.Fatal(err)
				}
				manifestPath := filepath.Join(dir, state.ManifestFile())
				manifest, err := state.LoadManifest(manifestPath)
				if err != nil {
					t.Fatal(err)
				}
				if _, ok := manifest[claudecode.PackageGuardPath]; !ok {
					t.Fatalf("manifest does not list %s", claudecode.PackageGuardPath)
				}
				manifest[claudecode.PackageGuardPath] = state.ComputeHash(replaced)
				if err := state.WriteManifest(dir, manifest); err != nil {
					t.Fatal(err)
				}
			}
			if tt.unregister {
				editSettings(t, dir, func(s map[string]any) {
					hooks := s["hooks"].(map[string]any)
					var kept []any
					for _, m := range hooks[claudesettings.EventPreToolUse].([]any) {
						data, _ := json.Marshal(m)
						if !strings.Contains(string(data), claudecode.PackageGuardPath) {
							kept = append(kept, m)
						}
					}
					hooks[claudesettings.EventPreToolUse] = kept
				})
			}
			editProjectConfig(t, dir, tt.config)

			out, err := runLifecycleCmd(t, dir, checkCmd(), "--format", "json", "--audit-level", "critical")
			if err == nil {
				t.Fatalf("check passed at critical:\n%s", out)
			}
			var report check.CheckReport
			if jerr := json.NewDecoder(strings.NewReader(out[strings.Index(out, "{"):])).Decode(&report); jerr != nil {
				t.Fatalf("parsing report: %v\n%s", jerr, out)
			}
			for _, name := range tt.wantFail {
				if !slices.ContainsFunc(report.Checks, func(c check.CheckResult) bool {
					return c.Name == name && c.Status == check.StatusFail && c.Severity == check.SeverityCritical
				}) {
					t.Errorf("report lacks a critical %s failure:\n%s", name, out)
				}
			}
			if slices.ContainsFunc(report.Checks, func(c check.CheckResult) bool {
				return c.Name == "claude_settings_posture" && c.Status == check.StatusPass
			}) && slices.Contains(tt.wantFail, "expected_generation_failed") {
				t.Errorf("claude_settings_posture passes without the expected settings:\n%s", out)
			}
		})
	}
}
