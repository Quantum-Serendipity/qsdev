package claudecode_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/internal/answers"
	qsdevconfig "github.com/Quantum-Serendipity/qsdev/internal/config"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// selfprotectCommand is the command the self-protection hook runs.
func selfprotectCommand() string {
	return branding.Get().AppName + " selfprotect"
}

// hasSelfprotect reports whether the PreToolUse matchers run the
// self-protection hook, directly or wrapped by the sandbox.
func hasSelfprotect(matchers []claudecode.HookMatcher) bool {
	for _, m := range matchers {
		for _, h := range m.Hooks {
			if strings.Contains(h.Command, selfprotectCommand()) {
				return true
			}
		}
	}
	return false
}

// hookChoicesFor returns the HookChoices enabling exactly the names whose bit
// is set in mask, indexed into names.
func hookChoicesFor(t *testing.T, names []string, mask int) types.HookChoices {
	t.Helper()
	var h types.HookChoices
	for i, name := range names {
		if mask&(1<<i) == 0 {
			continue
		}
		if err := h.EnableHook(name); err != nil {
			t.Fatalf("EnableHook(%q): %v", name, err)
		}
	}
	return h
}

// settingsPreToolUse generates the Claude Code files for answers and returns
// the PreToolUse matchers of the emitted settings.json.
func settingsPreToolUse(t *testing.T, answers types.WizardAnswers) []claudecode.HookMatcher {
	t.Helper()
	files, err := claudecode.NewClaudeCodeGenerator(newTestRegistry(t, goMock()), claudecode.Config{}).Generate(answers)
	if err != nil {
		t.Fatalf("Generate: %v", err)
	}
	for _, f := range files {
		if f.Path != settingsRel {
			continue
		}
		var s claudecode.SettingsJSON
		if err := json.Unmarshal(f.Content, &s); err != nil {
			t.Fatalf("unmarshal settings: %v", err)
		}
		return s.Hooks["PreToolUse"]
	}
	t.Fatal("Generate emitted no settings.json")
	return nil
}

// packageGuardScript is the script the package-guard hook runs.
const packageGuardScript = ".claude/hooks/package-guard.py"

// hasPackageGuard reports whether the PreToolUse matchers run the
// package-guard hook, directly or wrapped by the sandbox.
func hasPackageGuard(matchers []claudecode.HookMatcher) bool {
	for _, m := range matchers {
		for _, h := range m.Hooks {
			if strings.Contains(h.Command, packageGuardScript) {
				return true
			}
		}
	}
	return false
}

// eachRegistryCase calls check with the PreToolUse matchers the default
// registry builds for every subset of hook choices (the answers file is
// agent-writable, so any subset is reachable), with and without a recorded
// safety-block opt-out and with Claude Code on and off.
func eachRegistryCase(t *testing.T, check func(t *testing.T, a types.WizardAnswers, pre []claudecode.HookMatcher)) {
	t.Helper()
	names := types.HookChoiceNames()
	r := claudecode.ExportDefaultHookRegistry()
	for mask := range 1 << len(names) {
		hooks := hookChoicesFor(t, names, mask)
		for _, optOut := range []bool{false, true} {
			hooks.SafetyBlockOptOut = optOut
			for _, cc := range []bool{true, false} {
				a := types.WizardAnswers{ClaudeCode: cc, Hooks: hooks}
				check(t, a, r.BuildHooksMap(a)["PreToolUse"])
			}
		}
	}
}

// generateSubsets returns the adversarial hook-choice subsets the Generate
// level cases use: none, all, and each single hook on and each single hook
// off, each with and without a recorded safety-block opt-out.
func generateSubsets(t *testing.T) map[string]types.HookChoices {
	t.Helper()
	names := types.HookChoiceNames()
	all := (1 << len(names)) - 1
	base := map[string]types.HookChoices{
		"all false": {},
		"all true":  hookChoicesFor(t, names, all),
	}
	for i, name := range names {
		base["only "+name] = hookChoicesFor(t, names, 1<<i)
		base["all but "+name] = hookChoicesFor(t, names, all&^(1<<i))
	}
	subsets := make(map[string]types.HookChoices, 2*len(base))
	for name, hooks := range base {
		subsets[name] = hooks
		hooks.SafetyBlockOptOut = true
		subsets[name+" opted out"] = hooks
	}
	return subsets
}

// generateAnswers returns standard-tier Claude Code answers with hooks.
func generateAnswers(hooks types.HookChoices) types.WizardAnswers {
	return types.WizardAnswers{
		ClaudeCode: true, Tier: "standard", Hooks: hooks,
		Languages: []types.LanguageChoice{{Name: "go"}},
	}
}

// TestGenerate_AlwaysRegistersSelfprotect is the U18-01 generator invariant:
// no combination of hook choices (the answers file is agent-writable) can
// produce Claude Code settings without the self-protection hook.
func TestGenerate_AlwaysRegistersSelfprotect(t *testing.T) {
	t.Parallel()

	t.Run("registry matrix", func(t *testing.T) {
		t.Parallel()
		eachRegistryCase(t, func(t *testing.T, a types.WizardAnswers, pre []claudecode.HookMatcher) {
			t.Helper()
			if !hasSelfprotect(pre) {
				t.Fatalf("ClaudeCode=%v hooks=%+v: PreToolUse lacks %q", a.ClaudeCode, a.Hooks, selfprotectCommand())
			}
		})
	})

	for name, hooks := range generateSubsets(t) {
		t.Run("generate "+name, func(t *testing.T) {
			t.Parallel()
			if !hasSelfprotect(settingsPreToolUse(t, generateAnswers(hooks))) {
				t.Errorf("hooks=%+v: settings.json PreToolUse lacks %q", hooks, selfprotectCommand())
			}
		})
	}
}

// TestGenerate_PackageGuardUnlessOptOut is the B1 package-guard invariant:
// the hook is registered exactly when no safety-block opt-out is recorded,
// whatever the other hook choices (including the SafetyBlock mirror) say, and
// Generate writes its script exactly when it registers it.
func TestGenerate_PackageGuardUnlessOptOut(t *testing.T) {
	t.Parallel()

	t.Run("registry matrix", func(t *testing.T) {
		t.Parallel()
		eachRegistryCase(t, func(t *testing.T, a types.WizardAnswers, pre []claudecode.HookMatcher) {
			t.Helper()
			if got, want := hasPackageGuard(pre), !a.Hooks.SafetyBlockOptOut; got != want {
				t.Fatalf("ClaudeCode=%v hooks=%+v: package-guard registered = %v, want %v", a.ClaudeCode, a.Hooks, got, want)
			}
		})
	})

	for name, hooks := range generateSubsets(t) {
		for _, cc := range []bool{true, false} {
			t.Run(fmt.Sprintf("generate %s claude code %v", name, cc), func(t *testing.T) {
				t.Parallel()
				a := generateAnswers(hooks)
				a.ClaudeCode = cc
				files, err := claudecode.NewClaudeCodeGenerator(newTestRegistry(t, goMock()), claudecode.Config{}).Generate(a)
				if err != nil {
					t.Fatalf("Generate: %v", err)
				}
				want := !hooks.SafetyBlockOptOut
				registered, script := false, false
				for _, f := range files {
					switch f.Path {
					case settingsRel:
						var s claudecode.SettingsJSON
						if err := json.Unmarshal(f.Content, &s); err != nil {
							t.Fatalf("unmarshal settings: %v", err)
						}
						registered = hasPackageGuard(s.Hooks["PreToolUse"])
					case packageGuardScript:
						script = true
					}
				}
				if registered != want || script != want {
					t.Errorf("hooks=%+v: package-guard registered = %v, script written = %v, want both %v",
						hooks, registered, script, want)
				}
			})
		}
	}
}

// settingsHasSelfprotect reports whether the project's settings.json
// registers the self-protection hook.
func settingsHasSelfprotect(t *testing.T, dir string) bool {
	t.Helper()
	var s claudecode.SettingsJSON
	if err := json.Unmarshal([]byte(readFile(t, filepath.Join(dir, settingsRel))), &s); err != nil {
		t.Fatalf("parsing settings.json: %v", err)
	}
	return hasSelfprotect(s.Hooks["PreToolUse"])
}

// claudeInitProject runs `claude init --yes` in a fresh project with its own
// HOME and returns the project directory.
func claudeInitProject(t *testing.T) string {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	chdir(t, dir)
	mustRunClaude(t, "init", "--yes")
	return dir
}

// TestClaudeInit_RegistersSelfprotect covers U18-V03: `claude init` records
// self-protection in the saved answers, so they agree with the settings.json
// it generates.
func TestClaudeInit_RegistersSelfprotect(t *testing.T) {
	dir := claudeInitProject(t)

	// Read the file as saved: loadAnswers would force the invariant itself.
	a, err := answers.LoadPrimary(dir)
	if err != nil {
		t.Fatalf("loading answers: %v", err)
	}
	if !a.Hooks.SelfProtection {
		t.Error("saved answers record self_protection: false after claude init")
	}
	if !settingsHasSelfprotect(t, dir) {
		t.Errorf("settings.json does not register %q", selfprotectCommand())
	}
}

// TestClaudeUpdate_KeepsSelfprotectAfterAnswersEdit is the U18-01 end-to-end
// case: editing the answers file to switch self-protection off, alone or
// together with Claude Code itself, and running `claude update` neither
// removes the hook nor persists the edit. A claude subcommand configures
// Claude Code, so the answers it saves record Claude Code on, matching the
// settings.json it generates.
func TestClaudeUpdate_KeepsSelfprotectAfterAnswersEdit(t *testing.T) {
	tests := []struct {
		name  string
		edits [][2]string
	}{
		{"self_protection off", [][2]string{{"self_protection: true", "self_protection: false"}}},
		{"claude_code and self_protection off", [][2]string{
			{"self_protection: true", "self_protection: false"},
			{"claude_code: true", "claude_code: false"},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := claudeInitProject(t)

			path := answers.PrimaryPath(dir)
			content := readFile(t, path)
			edited := content
			for _, e := range tt.edits {
				next := strings.Replace(edited, e[0], e[1], 1)
				if next == edited {
					t.Fatalf("answers file has no %q to edit:\n%s", e[0], content)
				}
				edited = next
			}
			if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
				t.Fatal(err)
			}

			mustRunClaude(t, "update", "--force")

			if !settingsHasSelfprotect(t, dir) {
				t.Errorf("settings.json lost %q after the answers edit", selfprotectCommand())
			}
			saved := readFile(t, path)
			for _, want := range []string{"self_protection: true", "claude_code: true"} {
				if !strings.Contains(saved, want) {
					t.Errorf("claude update did not re-save %s:\n%s", want, saved)
				}
			}
		})
	}
}

// TestClaudeUpdate_KeepsCommittedTier verifies `claude update` on answers
// saved without a tier (every project `claude init` created before the tier
// was always recorded) adopts the tier committed in .qsdev.yaml rather than
// inferring one and overwriting the committed tier, in either direction.
func TestClaudeUpdate_KeepsCommittedTier(t *testing.T) {
	tests := []struct {
		name       string
		committed  string
		mcpServers []string // chosen so inference would give a different tier
	}{
		{"standard not loosened to full", "standard", []string{"postgres"}},
		{"full not tightened to standard", "full", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := claudeInitProject(t)
			a, err := answers.LoadPrimary(dir)
			if err != nil {
				t.Fatalf("loading answers: %v", err)
			}
			a.Tier = ""
			a.MCPServers = tt.mcpServers
			if err := answers.SavePrimary(dir, a); err != nil {
				t.Fatal(err)
			}
			cfgPath := filepath.Join(dir, branding.Get().ConfigFile)
			if err := os.WriteFile(cfgPath, []byte("version: 1\ntier: "+tt.committed+"\n"), 0o644); err != nil {
				t.Fatal(err)
			}
			markJoined(t, dir)

			mustRunClaude(t, "update", "--force")

			cfg, err := qsdevconfig.ParseQsdevConfig(cfgPath)
			if err != nil {
				t.Fatalf("parsing %s: %v", cfgPath, err)
			}
			if cfg.Tier != tt.committed {
				t.Errorf("%s tier = %q, want committed %q", branding.Get().ConfigFile, cfg.Tier, tt.committed)
			}
			saved, err := answers.LoadPrimary(dir)
			if err != nil {
				t.Fatalf("loading answers: %v", err)
			}
			if saved.Tier != tt.committed {
				t.Errorf("answers tier = %q, want committed %q", saved.Tier, tt.committed)
			}
		})
	}
}
