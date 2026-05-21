package claudecode_test

import (
	"encoding/json"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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

	// Deny should have base rules (dangerous patterns).
	if len(s.Permissions.Deny) == 0 {
		t.Error("minimal deny should not be empty")
	}
	if !containsRule(s.Permissions.Deny, "Bash(npx *)") {
		t.Error("minimal deny should contain Bash(npx *)")
	}

	// Package install commands should be in ask, not deny.
	if containsRule(s.Permissions.Deny, "Bash(npm install *)") {
		t.Error("minimal deny should NOT contain Bash(npm install *) — it belongs in ask")
	}
	if !containsRule(s.Permissions.Ask, "Bash(npm install *)") {
		t.Error("minimal ask should contain Bash(npm install *)")
	}

	// Frozen lockfile installs should be in allow.
	if !containsRule(s.Permissions.Allow, "Bash(npm ci)") {
		t.Error("minimal allow should contain Bash(npm ci)")
	}

	// DefaultMode should be "plan" for minimal (most restrictive).
	if s.Permissions.DefaultMode != "plan" {
		t.Errorf("minimal should set defaultMode to plan, got %q", s.Permissions.DefaultMode)
	}
	if s.Permissions.DisableBypassPermissionsMode != "disable" {
		t.Errorf("minimal should set disableBypassPermissionsMode to disable, got %q", s.Permissions.DisableBypassPermissionsMode)
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
	if !containsRule(s.Permissions.Allow, "Bash(cargo audit *)") {
		t.Error("standard allow should contain Bash(cargo audit *)")
	}

	// Code-execution commands should be in ask, not allow.
	for _, cmd := range []string{"Bash(npm run *)", "Bash(go run *)", "Bash(nix run *)", "Bash(nix build *)", "Bash(cargo run *)"} {
		if containsRule(s.Permissions.Allow, cmd) {
			t.Errorf("standard allow should NOT contain %s — it belongs in ask", cmd)
		}
		if !containsRule(s.Permissions.Ask, cmd) {
			t.Errorf("standard ask should contain %s", cmd)
		}
	}

	// Deny should have dangerous pattern rules (not package installs).
	if !containsRule(s.Permissions.Deny, "Bash(curl * | bash)") {
		t.Error("standard deny should contain Bash(curl * | bash)")
	}
	if !containsRule(s.Permissions.Deny, "Bash(npx *)") {
		t.Error("standard deny should contain Bash(npx *)")
	}

	// Package install commands should be in ask, not deny.
	if containsRule(s.Permissions.Deny, "Bash(npm install *)") {
		t.Error("standard deny should NOT contain Bash(npm install *) — it belongs in ask")
	}
	if containsRule(s.Permissions.Deny, "Bash(pip install *)") {
		t.Error("standard deny should NOT contain Bash(pip install *) — it belongs in ask")
	}
	if !containsRule(s.Permissions.Ask, "Bash(npm install *)") {
		t.Error("standard ask should contain Bash(npm install *)")
	}
	if !containsRule(s.Permissions.Ask, "Bash(pip install *)") {
		t.Error("standard ask should contain Bash(pip install *)")
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
	if !containsRule(s.Permissions.Ask, "Bash(go get *)") {
		t.Error("standard ask should contain Bash(go get *)")
	}
	if !containsRule(s.Permissions.Ask, "Bash(cargo add *)") {
		t.Error("standard ask should contain Bash(cargo add *)")
	}
	if !containsRule(s.Permissions.Ask, "Bash(gem install *)") {
		t.Error("standard ask should contain Bash(gem install *)")
	}
	if !containsRule(s.Permissions.Ask, "Bash(composer require *)") {
		t.Error("standard ask should contain Bash(composer require *)")
	}

	// Frozen lockfile installs should be in allow.
	if !containsRule(s.Permissions.Allow, "Bash(npm ci)") {
		t.Error("standard allow should contain Bash(npm ci)")
	}
	if !containsRule(s.Permissions.Allow, "Bash(pnpm install --frozen-lockfile)") {
		t.Error("standard allow should contain Bash(pnpm install --frozen-lockfile)")
	}
	if !containsRule(s.Permissions.Allow, "Bash(yarn install --immutable)") {
		t.Error("standard allow should contain Bash(yarn install --immutable)")
	}
	if !containsRule(s.Permissions.Allow, "Bash(bun install --frozen-lockfile)") {
		t.Error("standard allow should contain Bash(bun install --frozen-lockfile)")
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

	// Deny should contain dangerous patterns but not package installs.
	if !containsRule(s.Permissions.Deny, `Bash(rm -rf *)`) {
		t.Error("permissive deny should contain Bash(rm -rf *)")
	}
	if !containsRule(s.Permissions.Deny, `Bash(npx *)`) {
		t.Error("permissive deny should contain Bash(npx *)")
	}
	if containsRule(s.Permissions.Deny, "Bash(npm install *)") {
		t.Error("permissive deny should NOT contain Bash(npm install *) — it belongs in ask")
	}

	// Package installs should be in ask.
	if !containsRule(s.Permissions.Ask, "Bash(npm install *)") {
		t.Error("permissive ask should contain Bash(npm install *)")
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

	// Deny should have dangerous patterns but not package installs.
	if !containsRule(s.Permissions.Deny, "Bash(npx *)") {
		t.Error("custom deny should contain Bash(npx *)")
	}
	if containsRule(s.Permissions.Deny, "Bash(npm install *)") {
		t.Error("custom deny should NOT contain Bash(npm install *) — it belongs in ask")
	}
	// Extra deny should be present.
	if !containsRule(s.Permissions.Deny, "Bash(forbidden *)") {
		t.Error("custom deny should contain extra Bash(forbidden *)")
	}

	// Package installs should be in ask for custom too.
	if !containsRule(s.Permissions.Ask, "Bash(npm install *)") {
		t.Error("custom ask should contain Bash(npm install *)")
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

	// Base deny rules should still be present (dangerous patterns).
	if !containsRule(s.Permissions.Deny, "Bash(curl * | bash)") {
		t.Error("deny should still contain base rule Bash(curl * | bash)")
	}

	// Package install rules should be in ask, not deny.
	if !containsRule(s.Permissions.Ask, "Bash(pip install *)") {
		t.Error("ask should contain Bash(pip install *)")
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
			"Bash(npx --yes *)", // overlaps with another module
		},
	})
	_ = reg.Register(&ecosystem.MockModule{
		NameVal:        "typescript",
		DisplayNameVal: "TypeScript",
		TierVal:        1,
		DenyRulesVal: []string{
			"Bash(npm install --ignore-scripts *)", // duplicate with javascript module
			"Bash(npx --yes *)",                    // duplicate with javascript module
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

	// Count occurrences of the overlapping ecosystem deny rule.
	count := 0
	for _, r := range s.Permissions.Deny {
		if r == "Bash(npm install --ignore-scripts *)" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Bash(npm install --ignore-scripts *) should appear exactly once in deny, got %d", count)
	}

	// Verify npx --yes is also deduped.
	count = 0
	for _, r := range s.Permissions.Deny {
		if r == "Bash(npx --yes *)" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("Bash(npx --yes *) should appear exactly once in deny, got %d", count)
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
		t.Errorf("mode should be 0o644, got %04o", gf.Mode)
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

	criticalDenyRules := []string{
		// npx (arbitrary code execution)
		`Bash(npx *)`,

		// Nix imperative installs
		`Bash(nix-env -i *)`,
		`Bash(nix profile install *)`,

		// System package managers
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

	for _, rule := range criticalDenyRules {
		if !containsRule(s.Permissions.Deny, rule) {
			t.Errorf("critical deny rule missing: %s", rule)
		}
	}

	// Package install commands should NOT be in deny — they belong in ask.
	packageRulesNotInDeny := []string{
		`Bash(npm install *)`,
		`Bash(npm i *)`,
		`Bash(pip install *)`,
		`Bash(pip3 install *)`,
		`Bash(cargo install *)`,
		`Bash(go get *)`,
		`Bash(go install *)`,
		`Bash(gem install *)`,
		`Bash(composer require *)`,
	}

	for _, rule := range packageRulesNotInDeny {
		if containsRule(s.Permissions.Deny, rule) {
			t.Errorf("package install rule should NOT be in deny (belongs in ask): %s", rule)
		}
	}

	// Critical ask rules — package installs gated by PreToolUse hook.
	criticalAskRules := []string{
		`Bash(npm install *)`,
		`Bash(npm i *)`,
		`Bash(pip install *)`,
		`Bash(pip3 install *)`,
		`Bash(cargo install *)`,
		`Bash(cargo add *)`,
		`Bash(go get *)`,
		`Bash(go install *)`,
		`Bash(gem install *)`,
		`Bash(composer require *)`,
		`Bash(uv pip install *)`,
		`Bash(uv add *)`,
		`Bash(yarn add *)`,
		`Bash(pnpm add *)`,
		`Bash(bun add *)`,
	}

	for _, rule := range criticalAskRules {
		if !containsRule(s.Permissions.Ask, rule) {
			t.Errorf("critical ask rule missing: %s", rule)
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

	// Should still have base deny rules (dangerous patterns).
	if !containsRule(s.Permissions.Deny, "Bash(curl * | bash)") {
		t.Error("nil registry should still include base deny rules")
	}
	// Package installs should be in ask.
	if !containsRule(s.Permissions.Ask, "Bash(npm install *)") {
		t.Error("nil registry should still include package install ask rules")
	}
}

func TestGenerateSettings_TierDrivesPresetWhenPermissionLevelEmpty(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry(t)
	answers := types.WizardAnswers{
		ProjectRoot: "/tmp/test",
		ProjectName: "test",
		ClaudeCode:  true,
		Tier:        "supply-chain-only",
		Languages:   []types.LanguageChoice{{Name: "go"}},
		Hooks:       types.HookChoices{SafetyBlock: true},
	}
	gf := mustGenerateSettings(t, answers, reg)
	s := mustUnmarshalSettings(t, gf)

	// When Tier is set and PermissionLevel is empty (FillDefaults doesn't
	// fill it when Tier is present), the tier's default preset is used.
	if s.Permissions.DefaultMode != "" {
		t.Errorf("supply-chain-only should have no defaultMode, got %q", s.Permissions.DefaultMode)
	}

	// Supply chain deny rules should be present.
	if !containsRule(s.Permissions.Deny, `Bash(npx *)`) {
		t.Error("supply-chain-only tier should include supply chain deny rules")
	}
}

func TestGenerateSettings_ExplicitPermissionLevelWithoutTier(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry(t)
	answers := types.WizardAnswers{
		ProjectRoot:     "/tmp/test",
		ProjectName:     "test",
		ClaudeCode:      true,
		Tier:            "",
		PermissionLevel: "minimal",
		Languages:       []types.LanguageChoice{{Name: "go"}},
		Hooks:           types.HookChoices{SafetyBlock: true},
	}
	gf := mustGenerateSettings(t, answers, reg)
	s := mustUnmarshalSettings(t, gf)

	// When Tier is empty, PermissionLevel should be used directly.
	if s.Permissions.DefaultMode != "plan" {
		t.Errorf("minimal preset should have defaultMode=plan, got %q", s.Permissions.DefaultMode)
	}
}

func TestGenerateSettings_SupplyChainOnlyPreset(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry(t)
	answers := types.WizardAnswers{
		ProjectRoot:     "/tmp/test",
		ProjectName:     "test",
		ClaudeCode:      true,
		PermissionLevel: "supply-chain-only",
		Languages:       []types.LanguageChoice{{Name: "go"}},
		Hooks:           types.HookChoices{SafetyBlock: true},
	}
	gf := mustGenerateSettings(t, answers, reg)
	s := mustUnmarshalSettings(t, gf)

	if len(s.Permissions.Allow) != 0 {
		t.Errorf("supply-chain-only Allow should be empty, got %d rules", len(s.Permissions.Allow))
	}

	supplyChainRules := []string{
		`Bash(npx *)`,
		`Bash(nix-env -i *)`,
		`Bash(curl * | bash *)`,
		`Bash(bash -c *npm install*)`,
	}
	for _, rule := range supplyChainRules {
		if !containsRule(s.Permissions.Deny, rule) {
			t.Errorf("supply-chain-only Deny missing supply chain rule: %s", rule)
		}
	}

	destructiveRules := []string{
		`Bash(git push --force *)`,
		`Bash(git reset --hard *)`,
		`Bash(rm -rf *)`,
	}
	for _, rule := range destructiveRules {
		if containsRule(s.Permissions.Deny, rule) {
			t.Errorf("supply-chain-only Deny should NOT include destructive op: %s", rule)
		}
	}

	if !containsRule(s.Permissions.Ask, `Bash(npm install *)`) {
		t.Error("supply-chain-only Ask should include npm install rules")
	}

	if s.Permissions.DefaultMode != "" {
		t.Errorf("supply-chain-only should have no defaultMode, got %q", s.Permissions.DefaultMode)
	}
	if s.Permissions.DisableBypassPermissionsMode != "" {
		t.Errorf("supply-chain-only should have no disableBypass, got %q", s.Permissions.DisableBypassPermissionsMode)
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
