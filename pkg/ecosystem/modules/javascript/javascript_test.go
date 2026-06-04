package javascript_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/javascript"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*javascript.Module)(nil)

func TestName(t *testing.T) {
	m := &javascript.Module{}
	if got := m.Name(); got != "javascript" {
		t.Errorf("Name() = %q, want %q", got, "javascript")
	}
}

func TestDisplayName(t *testing.T) {
	m := &javascript.Module{}
	if got := m.DisplayName(); got != "JavaScript/TypeScript" {
		t.Errorf("DisplayName() = %q, want %q", got, "JavaScript/TypeScript")
	}
}

func TestTier(t *testing.T) {
	m := &javascript.Module{}
	if got := m.Tier(); got != 1 {
		t.Errorf("Tier() = %d, want %d", got, 1)
	}
}

// --- Detection tests ---

func TestDetect_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	m := &javascript.Module{}
	result := m.Detect(dir)

	if result.Detected {
		t.Error("expected Detected=false for empty directory")
	}
	if result.Confidence != ecosystem.ConfidenceAbsent {
		t.Errorf("Confidence = %v, want ConfidenceAbsent", result.Confidence)
	}
}

func TestDetect_PackageJSONOnly(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name": "test-project"}`)

	m := &javascript.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when package.json is present")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want ConfidenceCertain", result.Confidence)
	}
	if result.SuggestedConfig.PackageManager != "npm" {
		t.Errorf("PackageManager = %q, want %q (default)", result.SuggestedConfig.PackageManager, "npm")
	}
	assertEvidenceContains(t, result.Evidence, "package.json")
}

func TestDetect_PackageLockJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name": "test"}`)
	writeFile(t, dir, "package-lock.json", `{}`)

	m := &javascript.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.SuggestedConfig.PackageManager != "npm" {
		t.Errorf("PackageManager = %q, want %q", result.SuggestedConfig.PackageManager, "npm")
	}
}

func TestDetect_PnpmLockYaml(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name": "test"}`)
	writeFile(t, dir, "pnpm-lock.yaml", "lockfileVersion: 9\n")

	m := &javascript.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.SuggestedConfig.PackageManager != "pnpm" {
		t.Errorf("PackageManager = %q, want %q", result.SuggestedConfig.PackageManager, "pnpm")
	}
	assertEvidenceContains(t, result.Evidence, "pnpm")
}

func TestDetect_YarnLock(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name": "test"}`)
	writeFile(t, dir, "yarn.lock", "# yarn lockfile v1\n")

	m := &javascript.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.SuggestedConfig.PackageManager != "yarn" {
		t.Errorf("PackageManager = %q, want %q", result.SuggestedConfig.PackageManager, "yarn")
	}
	assertEvidenceContains(t, result.Evidence, "yarn")
}

func TestDetect_BunLock(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name": "test"}`)
	writeFile(t, dir, "bun.lock", "")

	m := &javascript.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.SuggestedConfig.PackageManager != "bun" {
		t.Errorf("PackageManager = %q, want %q", result.SuggestedConfig.PackageManager, "bun")
	}
	assertEvidenceContains(t, result.Evidence, "bun")
}

func TestDetect_BunLockb(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name": "test"}`)
	writeFile(t, dir, "bun.lockb", "\x00binary")

	m := &javascript.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.SuggestedConfig.PackageManager != "bun" {
		t.Errorf("PackageManager = %q, want %q", result.SuggestedConfig.PackageManager, "bun")
	}
}

func TestDetect_NvmrcVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name": "test", "engines": {"node": ">=18"}}`)
	writeFile(t, dir, ".nvmrc", "v20.11.0\n")

	m := &javascript.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	// .nvmrc should take priority over engines.node; "v" prefix is stripped.
	if result.SuggestedConfig.Version != "20.11.0" {
		t.Errorf("Version = %q, want %q (from .nvmrc, v prefix stripped)", result.SuggestedConfig.Version, "20.11.0")
	}
	assertEvidenceContains(t, result.Evidence, ".nvmrc")
}

func TestDetect_EnginesNodeVersion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name": "test", "engines": {"node": ">=18.0.0"}}`)

	m := &javascript.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.SuggestedConfig.Version != ">=18.0.0" {
		t.Errorf("Version = %q, want %q", result.SuggestedConfig.Version, ">=18.0.0")
	}
	assertEvidenceContains(t, result.Evidence, "engines.node")
}

func TestDetect_TypeScript(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"name": "test"}`)
	writeFile(t, dir, "tsconfig.json", `{"compilerOptions": {}}`)

	m := &javascript.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.SuggestedConfig.Extras["typescript"] != "true" {
		t.Error("expected Extras[typescript]=true when tsconfig.json exists")
	}
	assertEvidenceContains(t, result.Evidence, "tsconfig.json")
}

// --- DevenvNixFragment tests ---

func TestDevenvNixFragment_NPM(t *testing.T) {
	m := &javascript.Module{}
	config := ecosystem.ModuleConfig{
		PackageManager: "npm",
		Version:        "20.11.0",
	}
	fragment, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	requiredStrings := []string{
		"languages.javascript",
		"enable = true",
		"pkgs.nodejs_20",
		"npm.enable = true",
	}
	for _, s := range requiredStrings {
		if !strings.Contains(fragment, s) {
			t.Errorf("DevenvNixFragment() missing %q\ngot:\n%s", s, fragment)
		}
	}
	// npm should NOT have pnpm/yarn/bun enables
	for _, s := range []string{"pnpm.enable", "yarn.enable", "languages.bun"} {
		if strings.Contains(fragment, s) {
			t.Errorf("DevenvNixFragment() should not contain %q for npm\ngot:\n%s", s, fragment)
		}
	}
}

func TestDevenvNixFragment_PNPM(t *testing.T) {
	m := &javascript.Module{}
	config := ecosystem.ModuleConfig{
		PackageManager: "pnpm",
		Version:        "22",
	}
	fragment, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	if !strings.Contains(fragment, "pkgs.nodejs_22") {
		t.Errorf("expected pkgs.nodejs_22, got:\n%s", fragment)
	}
	if !strings.Contains(fragment, "pnpm.enable = true") {
		t.Errorf("expected pnpm.enable, got:\n%s", fragment)
	}
	// Ensure the standalone npm enable line is absent (pnpm.enable contains "npm.enable" as substring)
	for line := range strings.SplitSeq(fragment, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "npm.enable = true;" {
			t.Errorf("pnpm config should not enable npm separately, got line: %q", line)
		}
	}
}

func TestDevenvNixFragment_Yarn(t *testing.T) {
	m := &javascript.Module{}
	config := ecosystem.ModuleConfig{
		PackageManager: "yarn",
		Version:        "v18.17.0",
	}
	fragment, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	if !strings.Contains(fragment, "pkgs.nodejs_18") {
		t.Errorf("expected pkgs.nodejs_18, got:\n%s", fragment)
	}
	if !strings.Contains(fragment, "yarn.enable = true") {
		t.Errorf("expected yarn.enable, got:\n%s", fragment)
	}
}

func TestDevenvNixFragment_Bun(t *testing.T) {
	m := &javascript.Module{}
	config := ecosystem.ModuleConfig{
		PackageManager: "bun",
	}
	fragment, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	if !strings.Contains(fragment, "languages.bun.enable = true") {
		t.Errorf("expected languages.bun.enable, got:\n%s", fragment)
	}
	// Default version package
	if !strings.Contains(fragment, "pkgs.nodejs_22") {
		t.Errorf("expected default pkgs.nodejs_22 for bun, got:\n%s", fragment)
	}
}

func TestDevenvNixFragment_TypeScript(t *testing.T) {
	m := &javascript.Module{}
	config := ecosystem.ModuleConfig{
		PackageManager: "npm",
		Extras:         map[string]string{"typescript": "true"},
	}
	fragment, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("DevenvNixFragment() error: %v", err)
	}

	if !strings.Contains(fragment, "languages.typescript.enable = true") {
		t.Errorf("expected typescript.enable, got:\n%s", fragment)
	}
}

func TestDevenvNixFragment_VersionMapping(t *testing.T) {
	m := &javascript.Module{}

	tests := []struct {
		version  string
		expected string
	}{
		{"18.17.0", "pkgs.nodejs_18"},
		{">=20", "pkgs.nodejs_20"},
		{"v22.1.0", "pkgs.nodejs_22"},
		{"^24.0.0", "pkgs.nodejs_24"},
		{"", "pkgs.nodejs_22"},       // default
		{"16.0.0", "pkgs.nodejs_22"}, // unmapped -> default
	}

	for _, tt := range tests {
		t.Run("version_"+tt.version, func(t *testing.T) {
			config := ecosystem.ModuleConfig{
				PackageManager: "npm",
				Version:        tt.version,
			}
			fragment, err := m.DevenvNixFragment(config)
			if err != nil {
				t.Fatalf("DevenvNixFragment() error: %v", err)
			}
			if !strings.Contains(fragment, tt.expected) {
				t.Errorf("version %q: expected %q in fragment\ngot:\n%s", tt.version, tt.expected, fragment)
			}
		})
	}
}

// --- SecurityConfigs tests ---

func TestSecurityConfigs_NPM(t *testing.T) {
	m := &javascript.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{PackageManager: "npm"})

	if len(configs) != 1 {
		t.Fatalf("SecurityConfigs() returned %d configs, want 1", len(configs))
	}

	cfg := configs[0]
	if cfg.Path != ".npmrc" {
		t.Errorf("Path = %q, want %q", cfg.Path, ".npmrc")
	}
	if cfg.Strategy != types.Overwrite {
		t.Errorf("Strategy = %v, want Overwrite", cfg.Strategy)
	}

	content := string(cfg.Content)
	requiredSettings := []string{
		"save-exact=true",
		"ignore-scripts=true",
		"min-release-age=3",
		"audit=true",
		"audit-level=moderate",
	}
	for _, s := range requiredSettings {
		if !strings.Contains(content, s) {
			t.Errorf(".npmrc missing %q\ncontent:\n%s", s, content)
		}
	}
	// Verify comments are present
	if !strings.Contains(content, "# Security-hardened npm configuration") {
		t.Errorf(".npmrc missing header comment\ncontent:\n%s", content)
	}
	if !strings.Contains(content, "npm >= 11.10.0") {
		t.Errorf(".npmrc missing version requirement comment\ncontent:\n%s", content)
	}
}

func TestSecurityConfigs_PNPM(t *testing.T) {
	m := &javascript.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{PackageManager: "pnpm"})

	if len(configs) != 1 {
		t.Fatalf("SecurityConfigs() returned %d configs, want 1", len(configs))
	}

	cfg := configs[0]
	if cfg.Path != "pnpm-workspace.yaml" {
		t.Errorf("Path = %q, want %q", cfg.Path, "pnpm-workspace.yaml")
	}
	if cfg.Strategy != types.Overwrite {
		t.Errorf("Strategy = %v, want Overwrite", cfg.Strategy)
	}

	content := string(cfg.Content)
	requiredStrings := []string{
		"strictDepBuilds",
		"true",
		"minimumReleaseAge",
		"4320", // 3 days in minutes, NOT milliseconds
		"trustPolicy",
		"no-downgrade",
		"blockExoticSubdeps",
	}
	for _, s := range requiredStrings {
		if !strings.Contains(content, s) {
			t.Errorf("pnpm-workspace.yaml missing %q\ncontent:\n%s", s, content)
		}
	}
	// Verify it has a comment header
	if !strings.Contains(content, "Security-hardened") {
		t.Errorf("pnpm-workspace.yaml missing header comment\ncontent:\n%s", content)
	}
	if !strings.Contains(content, "pnpm >= 10.16") {
		t.Errorf("pnpm-workspace.yaml missing version requirement comment\ncontent:\n%s", content)
	}
}

func TestSecurityConfigs_Yarn(t *testing.T) {
	m := &javascript.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{PackageManager: "yarn"})

	if len(configs) != 1 {
		t.Fatalf("SecurityConfigs() returned %d configs, want 1", len(configs))
	}

	cfg := configs[0]
	if cfg.Path != ".yarnrc.yml" {
		t.Errorf("Path = %q, want %q", cfg.Path, ".yarnrc.yml")
	}
	if cfg.Strategy != types.Overwrite {
		t.Errorf("Strategy = %v, want Overwrite", cfg.Strategy)
	}

	content := string(cfg.Content)
	requiredStrings := []string{
		"enableImmutableInstalls",
		"enableHardenedMode",
		"enableScripts",
		"false",
		"npmMinimalAgeGate",
		"7d",
	}
	for _, s := range requiredStrings {
		if !strings.Contains(content, s) {
			t.Errorf(".yarnrc.yml missing %q\ncontent:\n%s", s, content)
		}
	}
	if !strings.Contains(content, "Security-hardened") {
		t.Errorf(".yarnrc.yml missing header comment\ncontent:\n%s", content)
	}
	if !strings.Contains(content, "Yarn >= 4.10.0") {
		t.Errorf(".yarnrc.yml missing version requirement comment\ncontent:\n%s", content)
	}
}

func TestSecurityConfigs_Bun(t *testing.T) {
	m := &javascript.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{PackageManager: "bun"})

	if len(configs) != 1 {
		t.Fatalf("SecurityConfigs() returned %d configs, want 1", len(configs))
	}

	cfg := configs[0]
	if cfg.Path != "bunfig.toml" {
		t.Errorf("Path = %q, want %q", cfg.Path, "bunfig.toml")
	}
	if cfg.Strategy != types.Overwrite {
		t.Errorf("Strategy = %v, want Overwrite", cfg.Strategy)
	}

	content := string(cfg.Content)
	requiredStrings := []string{
		"[install]",
		`minimumReleaseAge = "7d"`,
	}
	for _, s := range requiredStrings {
		if !strings.Contains(content, s) {
			t.Errorf("bunfig.toml missing %q\ncontent:\n%s", s, content)
		}
	}
	if !strings.Contains(content, "# Security-hardened Bun configuration") {
		t.Errorf("bunfig.toml missing header comment\ncontent:\n%s", content)
	}
	if !strings.Contains(content, "Bun >= 1.3") {
		t.Errorf("bunfig.toml missing version requirement comment\ncontent:\n%s", content)
	}
}

func TestSecurityConfigs_DefaultsToNPM(t *testing.T) {
	m := &javascript.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{})

	if len(configs) != 1 {
		t.Fatalf("SecurityConfigs() returned %d configs, want 1", len(configs))
	}
	if configs[0].Path != ".npmrc" {
		t.Errorf("default SecurityConfigs should produce .npmrc, got %q", configs[0].Path)
	}
}

// --- Registry proxy tests ---

func TestSecurityConfigs_NPM_RegistryProxy(t *testing.T) {
	m := &javascript.Module{}
	proxy := "https://npm.corp.example.com"
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{
		PackageManager: "npm",
		RegistryProxy:  proxy,
	})

	if len(configs) != 1 {
		t.Fatalf("SecurityConfigs() returned %d configs, want 1", len(configs))
	}

	content := string(configs[0].Content)
	// Proxy line must be present.
	if !strings.Contains(content, "registry="+proxy) {
		t.Errorf(".npmrc missing registry proxy line\ncontent:\n%s", content)
	}
	// Existing security settings must be preserved.
	for _, s := range []string{"save-exact=true", "ignore-scripts=true", "min-release-age=3", "audit=true"} {
		if !strings.Contains(content, s) {
			t.Errorf(".npmrc missing existing security setting %q when proxy is set\ncontent:\n%s", s, content)
		}
	}
}

func TestSecurityConfigs_NPM_NoRegistryProxy(t *testing.T) {
	m := &javascript.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{PackageManager: "npm"})

	content := string(configs[0].Content)
	if strings.Contains(content, "registry=") {
		t.Errorf(".npmrc should not contain registry= when proxy is empty\ncontent:\n%s", content)
	}
}

func TestSecurityConfigs_PNPM_RegistryProxy(t *testing.T) {
	m := &javascript.Module{}
	proxy := "https://npm.corp.example.com"
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{
		PackageManager: "pnpm",
		RegistryProxy:  proxy,
	})

	if len(configs) != 1 {
		t.Fatalf("SecurityConfigs() returned %d configs, want 1", len(configs))
	}

	content := string(configs[0].Content)
	if !strings.Contains(content, "npmRegistryServer") {
		t.Errorf("pnpm-workspace.yaml missing npmRegistryServer when proxy is set\ncontent:\n%s", content)
	}
	if !strings.Contains(content, proxy) {
		t.Errorf("pnpm-workspace.yaml missing proxy URL\ncontent:\n%s", content)
	}
	// Existing security settings must be preserved.
	for _, s := range []string{"strictDepBuilds", "minimumReleaseAge", "trustPolicy", "blockExoticSubdeps"} {
		if !strings.Contains(content, s) {
			t.Errorf("pnpm-workspace.yaml missing %q when proxy is set\ncontent:\n%s", s, content)
		}
	}
}

func TestSecurityConfigs_PNPM_NoRegistryProxy(t *testing.T) {
	m := &javascript.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{PackageManager: "pnpm"})

	content := string(configs[0].Content)
	if strings.Contains(content, "npmRegistryServer") {
		t.Errorf("pnpm-workspace.yaml should not contain npmRegistryServer when proxy is empty\ncontent:\n%s", content)
	}
}

func TestSecurityConfigs_Yarn_RegistryProxy(t *testing.T) {
	m := &javascript.Module{}
	proxy := "https://npm.corp.example.com"
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{
		PackageManager: "yarn",
		RegistryProxy:  proxy,
	})

	if len(configs) != 1 {
		t.Fatalf("SecurityConfigs() returned %d configs, want 1", len(configs))
	}

	content := string(configs[0].Content)
	if !strings.Contains(content, "npmRegistryServer") {
		t.Errorf(".yarnrc.yml missing npmRegistryServer when proxy is set\ncontent:\n%s", content)
	}
	if !strings.Contains(content, proxy) {
		t.Errorf(".yarnrc.yml missing proxy URL\ncontent:\n%s", content)
	}
	// Existing security settings must be preserved.
	for _, s := range []string{"enableImmutableInstalls", "enableHardenedMode", "enableScripts", "npmMinimalAgeGate"} {
		if !strings.Contains(content, s) {
			t.Errorf(".yarnrc.yml missing %q when proxy is set\ncontent:\n%s", s, content)
		}
	}
}

func TestSecurityConfigs_Yarn_NoRegistryProxy(t *testing.T) {
	m := &javascript.Module{}
	configs := m.SecurityConfigs(ecosystem.ModuleConfig{PackageManager: "yarn"})

	content := string(configs[0].Content)
	if strings.Contains(content, "npmRegistryServer") {
		t.Errorf(".yarnrc.yml should not contain npmRegistryServer when proxy is empty\ncontent:\n%s", content)
	}
}

// --- PreCommitHooks tests ---

func TestPreCommitHooks(t *testing.T) {
	m := &javascript.Module{}
	hooks := m.PreCommitHooks(ecosystem.ModuleConfig{})

	if len(hooks) != 2 {
		t.Fatalf("PreCommitHooks() returned %d hooks, want 2", len(hooks))
	}

	expectedIDs := []string{"prettier", "eslint"}
	for i, hook := range hooks {
		if hook.ID != expectedIDs[i] {
			t.Errorf("hooks[%d].ID = %q, want %q", i, hook.ID, expectedIDs[i])
		}
		if !hook.BuiltIn {
			t.Errorf("hooks[%d].BuiltIn = false, want true", i)
		}
		if hook.Language != "node" {
			t.Errorf("hooks[%d].Language = %q, want %q", i, hook.Language, "node")
		}
		if len(hook.Stages) == 0 || hook.Stages[0] != "pre-commit" {
			t.Errorf("hooks[%d].Stages = %v, want [pre-commit]", i, hook.Stages)
		}
	}
}

// --- DenyRules tests ---

func TestDenyRules(t *testing.T) {
	m := &javascript.Module{}
	rules := m.DenyRules(ecosystem.ModuleConfig{})

	// Only npx + pipe-to-shell patterns remain (package installs moved to ask).
	if len(rules) != 5 {
		t.Fatalf("DenyRules() returned %d rules, want 5 (npx + 4 pipe-to-shell)", len(rules))
	}

	expectedPatterns := []string{
		"npx",
		"curl * | sh",
		"curl * | bash",
		"wget * | sh",
		"wget * | bash",
	}
	for _, pattern := range expectedPatterns {
		found := false
		for _, rule := range rules {
			if strings.Contains(rule, pattern) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("DenyRules() missing pattern containing %q\nrules: %v", pattern, rules)
		}
	}
}

// --- CICommands tests ---

func TestCICommands_NPM(t *testing.T) {
	m := &javascript.Module{}
	cmds := m.CICommands(ecosystem.ModuleConfig{PackageManager: "npm"})

	if len(cmds) != 1 {
		t.Fatalf("CICommands() returned %d commands, want 1", len(cmds))
	}
	if cmds[0].Command != "npm ci --ignore-scripts" {
		t.Errorf("Command = %q, want %q", cmds[0].Command, "npm ci --ignore-scripts")
	}
	if cmds[0].Phase != ecosystem.CIPhaseInstall {
		t.Errorf("Phase = %v, want CIPhaseInstall", cmds[0].Phase)
	}
}

func TestCICommands_PNPM(t *testing.T) {
	m := &javascript.Module{}
	cmds := m.CICommands(ecosystem.ModuleConfig{PackageManager: "pnpm"})

	if len(cmds) != 1 {
		t.Fatalf("CICommands() returned %d commands, want 1", len(cmds))
	}
	if cmds[0].Command != "pnpm install --frozen-lockfile" {
		t.Errorf("Command = %q, want %q", cmds[0].Command, "pnpm install --frozen-lockfile")
	}
}

func TestCICommands_Yarn(t *testing.T) {
	m := &javascript.Module{}
	cmds := m.CICommands(ecosystem.ModuleConfig{PackageManager: "yarn"})

	if len(cmds) != 1 {
		t.Fatalf("CICommands() returned %d commands, want 1", len(cmds))
	}
	if cmds[0].Command != "yarn install --immutable" {
		t.Errorf("Command = %q, want %q", cmds[0].Command, "yarn install --immutable")
	}
}

func TestCICommands_Bun(t *testing.T) {
	m := &javascript.Module{}
	cmds := m.CICommands(ecosystem.ModuleConfig{PackageManager: "bun"})

	if len(cmds) != 1 {
		t.Fatalf("CICommands() returned %d commands, want 1", len(cmds))
	}
	if cmds[0].Command != "bun install --frozen-lockfile" {
		t.Errorf("Command = %q, want %q", cmds[0].Command, "bun install --frozen-lockfile")
	}
}

func TestCICommands_DefaultPM(t *testing.T) {
	m := &javascript.Module{}
	cmds := m.CICommands(ecosystem.ModuleConfig{})

	if len(cmds) != 1 {
		t.Fatalf("CICommands() returned %d commands, want 1", len(cmds))
	}
	if cmds[0].Command != "npm ci --ignore-scripts" {
		t.Errorf("default CICommands should use npm ci --ignore-scripts, got %q", cmds[0].Command)
	}
}

// --- PackageManagers tests ---

func TestPackageManagers(t *testing.T) {
	m := &javascript.Module{}
	pms := m.PackageManagers()

	if len(pms) != 4 {
		t.Fatalf("PackageManagers() returned %d entries, want 4", len(pms))
	}

	expectedPMs := []struct {
		name     string
		lockFile string
		frozen   string
		ageGate  bool
	}{
		{"npm", "package-lock.json", "npm ci", true},
		{"pnpm", "pnpm-lock.yaml", "pnpm install --frozen-lockfile", true},
		{"yarn", "yarn.lock", "yarn install --immutable", true},
		{"bun", "bun.lock", "bun install --frozen-lockfile", true},
	}

	for i, expected := range expectedPMs {
		pm := pms[i]
		if pm.Name != expected.name {
			t.Errorf("pms[%d].Name = %q, want %q", i, pm.Name, expected.name)
		}
		if pm.LockFile != expected.lockFile {
			t.Errorf("pms[%d].LockFile = %q, want %q", i, pm.LockFile, expected.lockFile)
		}
		if pm.FrozenInstallCommand != expected.frozen {
			t.Errorf("pms[%d].FrozenInstallCommand = %q, want %q", i, pm.FrozenInstallCommand, expected.frozen)
		}
		if pm.AgeGatingSupport != expected.ageGate {
			t.Errorf("pms[%d].AgeGatingSupport = %v, want %v", i, pm.AgeGatingSupport, expected.ageGate)
		}
		if pm.InstallCommand == "" {
			t.Errorf("pms[%d].InstallCommand should not be empty", i)
		}
	}
}

// --- WizardFields tests ---

func TestWizardFields(t *testing.T) {
	m := &javascript.Module{}
	fields := m.WizardFields()

	if len(fields) != 2 {
		t.Fatalf("WizardFields() returned %d fields, want 2", len(fields))
	}

	// package_manager field
	pmField := fields[0]
	if pmField.Key != "package_manager" {
		t.Errorf("fields[0].Key = %q, want %q", pmField.Key, "package_manager")
	}
	if pmField.Type != ecosystem.FieldTypeSelect {
		t.Errorf("fields[0].Type = %v, want FieldTypeSelect", pmField.Type)
	}
	if len(pmField.Options) != 4 {
		t.Errorf("fields[0].Options has %d entries, want 4", len(pmField.Options))
	}

	// Verify all 4 PM options are present
	pmValues := make(map[string]bool)
	for _, opt := range pmField.Options {
		pmValues[opt.Value] = true
	}
	for _, pm := range []string{"npm", "pnpm", "yarn", "bun"} {
		if !pmValues[pm] {
			t.Errorf("package_manager options missing %q", pm)
		}
	}

	// typescript field
	tsField := fields[1]
	if tsField.Key != "typescript" {
		t.Errorf("fields[1].Key = %q, want %q", tsField.Key, "typescript")
	}
	if tsField.Type != ecosystem.FieldTypeConfirm {
		t.Errorf("fields[1].Type = %v, want FieldTypeConfirm", tsField.Type)
	}
}

// --- Registration tests ---

func TestRegistration(t *testing.T) {
	reg := ecosystem.DefaultRegistry()
	mod, ok := reg.ByName("javascript")
	if !ok {
		t.Fatal("expected module 'javascript' to be registered in DefaultRegistry")
	}
	if mod.Name() != "javascript" {
		t.Errorf("registered module Name() = %q, want %q", mod.Name(), "javascript")
	}
}

// --- Helper test utilities ---

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertEvidenceContains(t *testing.T, evidence []string, substr string) {
	t.Helper()
	for _, e := range evidence {
		if strings.Contains(e, substr) {
			return
		}
	}
	t.Errorf("evidence %v should contain entry with %q", evidence, substr)
}
