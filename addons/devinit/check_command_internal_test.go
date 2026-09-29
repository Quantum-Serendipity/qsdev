package devinit

import (
	"encoding/json"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/check"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
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
func TestCheckCmd_FailsWhenGuardHooksStripped(t *testing.T) {
	dir := initLifecycleProject(t)
	settingsPath := filepath.Join(dir, ".claude", "settings.json")

	postureFailures := func(out string) []string {
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

	out, _ := runLifecycleCmd(t, dir, checkCmd(), "--format", "json", "--audit-level", "low")
	if failed := postureFailures(out); len(failed) != 0 {
		t.Fatalf("freshly generated project fails posture checks: %v", failed)
	}

	var settings map[string]any
	if err := json.Unmarshal([]byte(readProjectFile(t, dir, ".claude/settings.json")), &settings); err != nil {
		t.Fatal(err)
	}
	delete(settings, "hooks")
	perms := settings["permissions"].(map[string]any)
	perms["defaultMode"] = "bypassPermissions"
	delete(perms, "disableBypassPermissionsMode")
	data, err := json.Marshal(settings)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settingsPath, data, 0o644); err != nil {
		t.Fatal(err)
	}

	out, err = runLifecycleCmd(t, dir, checkCmd(), "--format", "json", "--audit-level", "low")
	if err == nil {
		t.Fatalf("check passed with guard hooks stripped:\n%s", out)
	}
	failed := postureFailures(out)
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
	if err == nil || !slices.Contains(postureFailures(out), "claude_hook_missing") {
		t.Errorf("without saved answers the stripped hooks went unreported (err=%v):\n%s", err, out)
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
