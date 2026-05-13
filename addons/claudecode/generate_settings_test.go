package claudecode_test

import (
	"encoding/json"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/claudecode"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// helper to generate settings and fail on error.
func mustGenerateSettings(t *testing.T, answers types.WizardAnswers, registry *ecosystem.Registry, opts ...claudecode.Option) *types.GeneratedFile {
	t.Helper()
	cfg := claudecode.NewConfig(opts...)
	gf, err := claudecode.GenerateSettings(answers, registry, cfg)
	if err != nil {
		t.Fatalf("GenerateSettings returned error: %v", err)
	}
	return gf
}

// helper to unmarshal the generated JSON into SettingsJSON.
func mustUnmarshalSettings(t *testing.T, gf *types.GeneratedFile) claudecode.SettingsJSON {
	t.Helper()
	var s claudecode.SettingsJSON
	if err := json.Unmarshal(gf.Content, &s); err != nil {
		t.Fatalf("JSON unmarshal failed: %v\nContent:\n%s", err, string(gf.Content))
	}
	return s
}

// containsRule checks whether a string slice contains the given rule.
func containsRule(rules []string, rule string) bool {
	for _, r := range rules {
		if r == rule {
			return true
		}
	}
	return false
}

func TestGenerateSettings_MinimalPreset(t *testing.T) {
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{
		PermissionLevel: "minimal",
	}
	gf := mustGenerateSettings(t, answers, reg)
	s := mustUnmarshalSettings(t, gf)

	// Allow should contain Read(*) but not Edit(*) or Write(*).
	if !containsRule(s.Permissions.Allow, "Read(*)") {
		t.Error("minimal allow should contain Read(*)")
	}
	if containsRule(s.Permissions.Allow, "Edit(*)") {
		t.Error("minimal allow should NOT contain Edit(*)")
	}
	if containsRule(s.Permissions.Allow, "Write(*)") {
		t.Error("minimal allow should NOT contain Write(*)")
	}

	// Allow should contain build/test commands.
	if !containsRule(s.Permissions.Allow, "Bash(go build *)") {
		t.Error("minimal allow should contain Bash(go build *)")
	}
	if !containsRule(s.Permissions.Allow, "Bash(cargo test *)") {
		t.Error("minimal allow should contain Bash(cargo test *)")
	}

	// Deny should have base rules.
	if len(s.Permissions.Deny) == 0 {
		t.Error("minimal deny should not be empty")
	}
	if !containsRule(s.Permissions.Deny, "Bash(npm install *)") {
		t.Error("minimal deny should contain Bash(npm install *)")
	}

	// DefaultMode should NOT be set for minimal.
	if s.Permissions.DefaultMode != "" {
		t.Errorf("minimal should not set defaultMode, got %q", s.Permissions.DefaultMode)
	}

	// Verify JSON is valid by attempting re-marshal.
	if _, err := json.Marshal(s); err != nil {
		t.Errorf("re-marshal failed: %v", err)
	}
}

func TestGenerateSettings_StandardPreset(t *testing.T) {
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{
		PermissionLevel: "standard",
	}
	gf := mustGenerateSettings(t, answers, reg)
	s := mustUnmarshalSettings(t, gf)

	// Standard allow should include Edit, Write, git.
	if !containsRule(s.Permissions.Allow, "Edit(*)") {
		t.Error("standard allow should contain Edit(*)")
	}
	if !containsRule(s.Permissions.Allow, "Write(*)") {
		t.Error("standard allow should contain Write(*)")
	}
	if !containsRule(s.Permissions.Allow, "Bash(git *)") {
		t.Error("standard allow should contain Bash(git *)")
	}

	// Should contain build/dev commands.
	if !containsRule(s.Permissions.Allow, "Bash(nix develop *)") {
		t.Error("standard allow should contain Bash(nix develop *)")
	}
	if !containsRule(s.Permissions.Allow, "Bash(npm run *)") {
		t.Error("standard allow should contain Bash(npm run *)")
	}
	if !containsRule(s.Permissions.Allow, "Bash(cargo audit *)") {
		t.Error("standard allow should contain Bash(cargo audit *)")
	}

	// Deny should have all base deny rules.
	if !containsRule(s.Permissions.Deny, "Bash(npm install *)") {
		t.Error("standard deny should contain Bash(npm install *)")
	}
	if !containsRule(s.Permissions.Deny, "Bash(pip install *)") {
		t.Error("standard deny should contain Bash(pip install *)")
	}
	if !containsRule(s.Permissions.Deny, "Bash(curl * | bash)") {
		t.Error("standard deny should contain Bash(curl * | bash)")
	}

	// DefaultMode and disableBypass should be set.
	if s.Permissions.DefaultMode != "default" {
		t.Errorf("standard defaultMode should be 'default', got %q", s.Permissions.DefaultMode)
	}
	if s.Permissions.DisableBypassPermissionsMode != "disable" {
		t.Errorf("standard disableBypassPermissionsMode should be 'disable', got %q", s.Permissions.DisableBypassPermissionsMode)
	}

	// Ask rules should be present.
	if !containsRule(s.Permissions.Ask, "Bash(nix flake update)") {
		t.Error("standard ask should contain Bash(nix flake update)")
	}
	if !containsRule(s.Permissions.Ask, "Bash(pip install -r requirements.txt)") {
		t.Error("standard ask should contain Bash(pip install -r requirements.txt)")
	}
}

func TestGenerateSettings_PermissivePreset(t *testing.T) {
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{
		PermissionLevel: "permissive",
	}
	gf := mustGenerateSettings(t, answers, reg)
	s := mustUnmarshalSettings(t, gf)

	// Permissive should include docker and make.
	if !containsRule(s.Permissions.Allow, "Bash(docker *)") {
		t.Error("permissive allow should contain Bash(docker *)")
	}
	if !containsRule(s.Permissions.Allow, "Bash(make *)") {
		t.Error("permissive allow should contain Bash(make *)")
	}

	// Should also include standard base rules.
	if !containsRule(s.Permissions.Allow, "Edit(*)") {
		t.Error("permissive allow should contain Edit(*)")
	}
	if !containsRule(s.Permissions.Allow, "Bash(git *)") {
		t.Error("permissive allow should contain Bash(git *)")
	}

	// Deny should still be full.
	if !containsRule(s.Permissions.Deny, "Bash(npm install *)") {
		t.Error("permissive deny should contain Bash(npm install *)")
	}
	if !containsRule(s.Permissions.Deny, `Bash(rm -rf *)`) {
		t.Error("permissive deny should contain Bash(rm -rf *)")
	}

	// DefaultMode and disableBypass should be set.
	if s.Permissions.DefaultMode != "default" {
		t.Errorf("permissive defaultMode should be 'default', got %q", s.Permissions.DefaultMode)
	}
	if s.Permissions.DisableBypassPermissionsMode != "disable" {
		t.Errorf("permissive disableBypassPermissionsMode should be 'disable', got %q", s.Permissions.DisableBypassPermissionsMode)
	}
}

func TestGenerateSettings_CustomPreset(t *testing.T) {
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{
		PermissionLevel: "custom",
	}
	gf := mustGenerateSettings(t, answers, reg,
		claudecode.WithExtraAllowPatterns("Bash(my-tool *)", "Read(*.log)"),
		claudecode.WithExtraDenyPatterns("Bash(forbidden *)"),
	)
	s := mustUnmarshalSettings(t, gf)

	// Custom allow should only contain ExtraAllowPatterns.
	if !containsRule(s.Permissions.Allow, "Bash(my-tool *)") {
		t.Error("custom allow should contain Bash(my-tool *)")
	}
	if !containsRule(s.Permissions.Allow, "Read(*.log)") {
		t.Error("custom allow should contain Read(*.log)")
	}
	// Should NOT contain standard rules.
	if containsRule(s.Permissions.Allow, "Edit(*)") {
		t.Error("custom allow should NOT contain Edit(*)")
	}
	if containsRule(s.Permissions.Allow, "Bash(git *)") {
		t.Error("custom allow should NOT contain Bash(git *)")
	}

	// Deny should still have base rules.
	if !containsRule(s.Permissions.Deny, "Bash(npm install *)") {
		t.Error("custom deny should still contain base Bash(npm install *)")
	}
	// Extra deny should be present.
	if !containsRule(s.Permissions.Deny, "Bash(forbidden *)") {
		t.Error("custom deny should contain extra Bash(forbidden *)")
	}

	// DefaultMode should NOT be set for custom.
	if s.Permissions.DefaultMode != "" {
		t.Errorf("custom should not set defaultMode, got %q", s.Permissions.DefaultMode)
	}
}

func TestGenerateSettings_EcosystemDenyRules(t *testing.T) {
	reg := ecosystem.NewRegistry()
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "javascript",
		DisplayNameVal: "JavaScript",
		TierVal:        1,
		DenyRulesVal: []string{
			"Bash(npm install --ignore-scripts *)",
			"Bash(npx --yes *)",
		},
	})

	answers := types.WizardAnswers{
		PermissionLevel: "standard",
		Languages: []types.LanguageChoice{
			{Name: "javascript", Version: "22", PackageManager: "npm"},
		},
	}
	gf := mustGenerateSettings(t, answers, reg)
	s := mustUnmarshalSettings(t, gf)

	// Ecosystem-specific deny rules should be present.
	if !containsRule(s.Permissions.Deny, "Bash(npm install --ignore-scripts *)") {
		t.Error("deny should contain ecosystem rule Bash(npm install --ignore-scripts *)")
	}
	if !containsRule(s.Permissions.Deny, "Bash(npx --yes *)") {
		t.Error("deny should contain ecosystem rule Bash(npx --yes *)")
	}

	// Base deny rules should also be present.
	if !containsRule(s.Permissions.Deny, "Bash(pip install *)") {
		t.Error("deny should still contain base rule Bash(pip install *)")
	}
}

func TestGenerateSettings_DenyRuleDeduplication(t *testing.T) {
	reg := ecosystem.NewRegistry()
	// Two modules both returning an overlapping rule.
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "javascript",
		DisplayNameVal: "JavaScript",
		TierVal:        1,
		DenyRulesVal: []string{
			"Bash(npm install --ignore-scripts *)",
			"Bash(npm install *)", // overlaps with base
		},
	})
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "typescript",
		DisplayNameVal: "TypeScript",
		TierVal:        1,
		DenyRulesVal: []string{
			"Bash(npm install --ignore-scripts *)", // duplicate with javascript module
			"Bash(npx --yes *)",
		},
	})

	answers := types.WizardAnswers{
		PermissionLevel: "standard",
		Languages: []types.LanguageChoice{
			{Name: "javascript", Version: "22", PackageManager: "npm"},
			{Name: "typescript", Version: "5", PackageManager: "npm"},
		},
	}
	gf := mustGenerateSettings(t, answers, reg)
	s := mustUnmarshalSettings(t, gf)

	// Count occurrences of the overlapping rule.
	count := 0
	for _, r := range s.Permissions.Deny {
		if r == "Bash(npm install --ignore-scripts *)" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Bash(npm install --ignore-scripts *) should appear exactly once, got %d", count)
	}

	// Also check that base rule Bash(npm install *) is not duplicated.
	count = 0
	for _, r := range s.Permissions.Deny {
		if r == "Bash(npm install *)" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Bash(npm install *) should appear exactly once, got %d", count)
	}
}

func TestGenerateSettings_SandboxEnabled(t *testing.T) {
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{
		PermissionLevel: "standard",
	}
	gf := mustGenerateSettings(t, answers, reg,
		claudecode.WithSandbox(true),
		claudecode.WithAllowedDomains("github.com", "registry.npmjs.org"),
	)
	s := mustUnmarshalSettings(t, gf)

	if s.Sandbox == nil {
		t.Fatal("sandbox should be present when enabled")
	}
	if !containsRule(s.Sandbox.WriteDeny, "/etc") {
		t.Error("sandbox writeDeny should contain /etc")
	}
	if !containsRule(s.Sandbox.WriteDeny, "/usr") {
		t.Error("sandbox writeDeny should contain /usr")
	}
	if !containsRule(s.Sandbox.NetAllow, "github.com") {
		t.Error("sandbox netAllow should contain github.com")
	}
	if !containsRule(s.Sandbox.NetAllow, "registry.npmjs.org") {
		t.Error("sandbox netAllow should contain registry.npmjs.org")
	}
}

func TestGenerateSettings_NoSandbox(t *testing.T) {
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{
		PermissionLevel: "standard",
	}
	gf := mustGenerateSettings(t, answers, reg)
	s := mustUnmarshalSettings(t, gf)

	if s.Sandbox != nil {
		t.Errorf("sandbox should be nil when not enabled, got %+v", s.Sandbox)
	}

	// Verify the "sandbox" key is omitted from JSON output.
	content := string(gf.Content)
	if containsStr(content, `"sandbox"`) {
		t.Error("sandbox key should not appear in JSON when disabled")
	}
}

func TestGenerateSettings_HooksSection(t *testing.T) {
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{
		PermissionLevel: "standard",
		Hooks: types.HookChoices{
			SafetyBlock: true,
		},
	}
	gf := mustGenerateSettings(t, answers, reg)
	s := mustUnmarshalSettings(t, gf)

	if s.Hooks == nil {
		t.Fatal("hooks should be present when SafetyBlock is true")
	}

	preToolUse, ok := s.Hooks["PreToolUse"]
	if !ok {
		t.Fatal("hooks should contain PreToolUse")
	}
	if len(preToolUse) != 1 {
		t.Fatalf("PreToolUse should have 1 matcher, got %d", len(preToolUse))
	}
	if preToolUse[0].Matcher != "Bash" {
		t.Errorf("PreToolUse matcher should be 'Bash', got %q", preToolUse[0].Matcher)
	}
	if len(preToolUse[0].Hooks) != 1 {
		t.Fatalf("PreToolUse Bash matcher should have 1 hook, got %d", len(preToolUse[0].Hooks))
	}
	hook := preToolUse[0].Hooks[0]
	if hook.Type != "command" {
		t.Errorf("hook type should be 'command', got %q", hook.Type)
	}
	if hook.Command != `"${CLAUDE_PROJECT_DIR}"/.claude/hooks/package-guard.py` {
		t.Errorf("hook command wrong: %q", hook.Command)
	}
	if hook.Timeout != 30 {
		t.Errorf("hook timeout should be 30, got %d", hook.Timeout)
	}
}

func TestGenerateSettings_NoHooksWhenSafetyBlockFalse(t *testing.T) {
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{
		PermissionLevel: "standard",
		Hooks: types.HookChoices{
			SafetyBlock: false,
		},
	}
	gf := mustGenerateSettings(t, answers, reg)
	s := mustUnmarshalSettings(t, gf)

	if s.Hooks != nil {
		t.Errorf("hooks should be nil when SafetyBlock is false, got %+v", s.Hooks)
	}

	// Verify the "hooks" key is omitted from JSON output.
	content := string(gf.Content)
	if containsStr(content, `"hooks"`) {
		t.Error("hooks key should not appear in JSON when SafetyBlock is false")
	}
}

func TestGenerateSettings_ValidJSONRoundTrip(t *testing.T) {
	reg := ecosystem.NewRegistry()
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "go",
		DisplayNameVal: "Go",
		TierVal:        1,
		DenyRulesVal:   []string{"Bash(go install -v *)"},
	})

	answers := types.WizardAnswers{
		PermissionLevel: "standard",
		Languages: []types.LanguageChoice{
			{Name: "go", Version: "1.24", PackageManager: "gomod"},
		},
		Hooks: types.HookChoices{SafetyBlock: true},
	}
	gf := mustGenerateSettings(t, answers, reg, claudecode.WithSandbox(true))

	// Unmarshal into generic map to verify JSON validity.
	var generic map[string]interface{}
	if err := json.Unmarshal(gf.Content, &generic); err != nil {
		t.Fatalf("JSON unmarshal to generic map failed: %v", err)
	}

	// Unmarshal back into SettingsJSON.
	var s claudecode.SettingsJSON
	if err := json.Unmarshal(gf.Content, &s); err != nil {
		t.Fatalf("JSON unmarshal to SettingsJSON failed: %v", err)
	}

	// Re-marshal and verify it's still valid.
	remarshaled, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		t.Fatalf("re-marshal failed: %v", err)
	}

	var s2 claudecode.SettingsJSON
	if err := json.Unmarshal(remarshaled, &s2); err != nil {
		t.Fatalf("second unmarshal failed: %v", err)
	}

	// Spot-check key fields survive round-trip.
	if s2.Permissions.DefaultMode != s.Permissions.DefaultMode {
		t.Errorf("defaultMode changed: %q -> %q", s.Permissions.DefaultMode, s2.Permissions.DefaultMode)
	}
	if len(s2.Permissions.Deny) != len(s.Permissions.Deny) {
		t.Errorf("deny count changed: %d -> %d", len(s.Permissions.Deny), len(s2.Permissions.Deny))
	}
}

func TestGenerateSettings_FileMetadata(t *testing.T) {
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{
		PermissionLevel: "standard",
	}
	gf := mustGenerateSettings(t, answers, reg)

	if gf.Path != ".claude/settings.json" {
		t.Errorf("path should be .claude/settings.json, got %q", gf.Path)
	}
	if gf.Mode != 0o644 {
		t.Errorf("mode should be 0644, got %04o", gf.Mode)
	}
	if gf.Strategy != types.ThreeWayMerge {
		t.Errorf("strategy should be ThreeWayMerge, got %v", gf.Strategy)
	}
	if len(gf.Content) == 0 {
		t.Error("content should not be empty")
	}
}

func TestGenerateSettings_CriticalDenyRulesPresent(t *testing.T) {
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{
		PermissionLevel: "standard",
	}
	gf := mustGenerateSettings(t, answers, reg)
	s := mustUnmarshalSettings(t, gf)

	criticalRules := []string{
		// Core package managers
		`Bash(npm install *)`,
		`Bash(npm i *)`,
		`Bash(pip install *)`,
		`Bash(pip3 install *)`,
		`Bash(cargo install *)`,
		`Bash(go get *)`,
		`Bash(go install *)`,
		`Bash(gem install *)`,
		`Bash(composer require *)`,
		`Bash(nix-env -i *)`,
		`Bash(nix profile install *)`,
		`Bash(apt install *)`,
		`Bash(brew install *)`,

		// Pipe-to-shell
		`Bash(curl * | bash)`,
		`Bash(curl * | sh)`,
		`Bash(wget * | bash)`,

		// Bypass mitigation — shell wrapping
		`Bash(bash -c *npm install*)`,
		`Bash(bash -c *pip install*)`,
		`Bash(sh -c *npm install*)`,

		// Bypass mitigation — env/command prefix
		`Bash(env npm install *)`,
		`Bash(command npm install *)`,

		// Bypass mitigation — sudo
		`Bash(sudo npm install *)`,
		`Bash(sudo pip install *)`,
		`Bash(sudo apt install *)`,

		// Bypass mitigation — subprocess escape
		`Bash(python -c *subprocess*)`,
		`Bash(node -e *child_process*)`,
		`Bash(node -e *execSync*)`,
		`Bash(ruby -e *system*)`,
		`Bash(perl -e *system*)`,

		// Bypass mitigation — eval/xargs
		`Bash(eval *npm install*)`,
		`Bash(eval *pip install*)`,
		`Bash(xargs npm install *)`,

		// Destructive ops
		`Bash(git push --force *)`,
		`Bash(git reset --hard *)`,
		`Bash(rm -rf *)`,
		`Read(./.env)`,
		`Read(./.env.*)`,
		`Read(./secrets/**)`,
	}

	for _, rule := range criticalRules {
		if !containsRule(s.Permissions.Deny, rule) {
			t.Errorf("critical deny rule missing: %s", rule)
		}
	}
}

func TestGenerateSettings_DefaultPresetWhenNoneSpecified(t *testing.T) {
	reg := ecosystem.NewRegistry()
	answers := types.WizardAnswers{
		// No PermissionLevel set.
	}
	gf := mustGenerateSettings(t, answers, reg)
	s := mustUnmarshalSettings(t, gf)

	// Should default to standard preset.
	if !containsRule(s.Permissions.Allow, "Edit(*)") {
		t.Error("default should behave as standard — allow should contain Edit(*)")
	}
	if s.Permissions.DefaultMode != "default" {
		t.Errorf("default should behave as standard — defaultMode should be 'default', got %q", s.Permissions.DefaultMode)
	}
}

func TestGenerateSettings_NilRegistryHandledGracefully(t *testing.T) {
	answers := types.WizardAnswers{
		PermissionLevel: "standard",
		Languages: []types.LanguageChoice{
			{Name: "go", Version: "1.24", PackageManager: "gomod"},
		},
	}
	cfg := claudecode.NewConfig()
	gf, err := claudecode.GenerateSettings(answers, nil, cfg)
	if err != nil {
		t.Fatalf("nil registry should not cause error: %v", err)
	}
	s := mustUnmarshalSettings(t, gf)

	// Should still have all base deny rules.
	if !containsRule(s.Permissions.Deny, "Bash(npm install *)") {
		t.Error("nil registry should still include base deny rules")
	}
}

// containsStr checks if s contains substr.
func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
