package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnifiedDefaults_ToCatalog_Tiers(t *testing.T) {
	t.Parallel()

	ud := &UnifiedDefaults{
		Tiers: map[string]TierDef{
			"custom": {
				Order:                   1,
				Description:             "Custom tier",
				DefaultPermissionPreset: "standard",
			},
			"another": {
				Order:                   2,
				Description:             "Another tier",
				DefaultPermissionPreset: "standard",
			},
		},
	}

	cat := ud.ToCatalog()

	if len(cat.tiers.Tiers) != 2 {
		t.Fatalf("tiers count = %d, want 2", len(cat.tiers.Tiers))
	}
	if cat.tiers.Tiers["custom"].Description != "Custom tier" {
		t.Errorf("custom tier description = %q, want %q",
			cat.tiers.Tiers["custom"].Description, "Custom tier")
	}
	if cat.tiers.Tiers["another"].Order != 2 {
		t.Errorf("another tier order = %d, want 2", cat.tiers.Tiers["another"].Order)
	}
}

func TestUnifiedDefaults_ToCatalog_Security(t *testing.T) {
	t.Parallel()

	ud := &UnifiedDefaults{
		SecurityHooks: []string{"ripsecrets", "gitleaks"},
		BasePackages:  []string{"git", "jq"},
		UnsetVars:     []string{"AWS_SECRET_ACCESS_KEY", "GITHUB_TOKEN"},
		KeepVars:      []string{"TERM", "HOME"},
	}

	cat := ud.ToCatalog()

	if len(cat.security.Hooks.Default) != 2 {
		t.Fatalf("security hooks count = %d, want 2", len(cat.security.Hooks.Default))
	}
	if cat.security.Hooks.Default[0] != "ripsecrets" {
		t.Errorf("hooks[0] = %q, want ripsecrets", cat.security.Hooks.Default[0])
	}
	if cat.security.Hooks.Default[1] != "gitleaks" {
		t.Errorf("hooks[1] = %q, want gitleaks", cat.security.Hooks.Default[1])
	}

	if len(cat.security.BasePackages) != 2 {
		t.Fatalf("base packages count = %d, want 2", len(cat.security.BasePackages))
	}
	if cat.security.BasePackages[0] != "git" {
		t.Errorf("base_packages[0] = %q, want git", cat.security.BasePackages[0])
	}

	if len(cat.security.CleanEnvironment.UnsetVars) != 2 {
		t.Fatalf("unset vars count = %d, want 2", len(cat.security.CleanEnvironment.UnsetVars))
	}
	if cat.security.CleanEnvironment.UnsetVars[0] != "AWS_SECRET_ACCESS_KEY" {
		t.Errorf("unset_vars[0] = %q, want AWS_SECRET_ACCESS_KEY",
			cat.security.CleanEnvironment.UnsetVars[0])
	}

	if len(cat.security.CleanEnvironment.KeepVars) != 2 {
		t.Fatalf("keep vars count = %d, want 2", len(cat.security.CleanEnvironment.KeepVars))
	}
	if cat.security.CleanEnvironment.KeepVars[0] != "TERM" {
		t.Errorf("keep_vars[0] = %q, want TERM", cat.security.CleanEnvironment.KeepVars[0])
	}
}

func TestUnifiedDefaults_ToCatalog_Derivations(t *testing.T) {
	t.Parallel()

	agentTools := DefaultAgentTools{
		PostmortemEnabled:    true,
		VersionSentinel:      true,
		VersionSentinelHours: 24,
		SembleEnabled:        true,
		SembleMode:           "both",
	}

	ud := &UnifiedDefaults{
		TierToCompliance: map[string]string{
			"supply-chain-only": "baseline",
			"standard":          "enhanced",
		},
		TierToEnabledTools: map[string][]string{
			"standard": {"gitleaks"},
			"full":     {"semgrep", "gitleaks", "secretspec"},
		},
		DefaultMCPServers: []string{"context7", "github"},
		DefaultAgentTools: &agentTools,
	}

	cat := ud.ToCatalog()

	if len(cat.derivations.TierToCompliance) != 2 {
		t.Fatalf("tier_to_compliance count = %d, want 2", len(cat.derivations.TierToCompliance))
	}
	if cat.derivations.TierToCompliance["standard"] != "enhanced" {
		t.Errorf("tier_to_compliance[standard] = %q, want enhanced",
			cat.derivations.TierToCompliance["standard"])
	}

	if len(cat.derivations.TierToEnabledTools["full"]) != 3 {
		t.Fatalf("tier_to_enabled_tools[full] count = %d, want 3",
			len(cat.derivations.TierToEnabledTools["full"]))
	}

	if len(cat.derivations.DefaultMCPServers) != 2 {
		t.Fatalf("default_mcp_servers count = %d, want 2", len(cat.derivations.DefaultMCPServers))
	}
	if cat.derivations.DefaultMCPServers[0] != "context7" {
		t.Errorf("default_mcp_servers[0] = %q, want context7",
			cat.derivations.DefaultMCPServers[0])
	}

	if !cat.derivations.DefaultAgentTools.PostmortemEnabled {
		t.Error("postmortem_enabled should be true")
	}
	if cat.derivations.DefaultAgentTools.VersionSentinelHours != 24 {
		t.Errorf("version_sentinel_hours = %d, want 24",
			cat.derivations.DefaultAgentTools.VersionSentinelHours)
	}
	if cat.derivations.DefaultAgentTools.SembleMode != "both" {
		t.Errorf("semble_mode = %q, want both",
			cat.derivations.DefaultAgentTools.SembleMode)
	}
}

func TestCatalog_ToUnified_RoundTrip(t *testing.T) {
	t.Parallel()

	cat := loadTestCatalog(t)

	unified := cat.ToUnified()
	roundTripped := unified.ToCatalog()

	// Verify tier count matches.
	origTiers := cat.TierDefs()
	rtTiers := roundTripped.TierDefs()
	if len(rtTiers) != len(origTiers) {
		t.Errorf("round-trip tier count = %d, want %d", len(rtTiers), len(origTiers))
	}

	// Verify tool count matches.
	origTools := cat.Tools()
	rtTools := roundTripped.Tools()
	if len(rtTools) != len(origTools) {
		t.Errorf("round-trip tool count = %d, want %d", len(rtTools), len(origTools))
	}

	// Verify security hook count matches.
	origHooks := cat.SecurityHooks()
	rtHooks := roundTripped.SecurityHooks()
	if len(rtHooks) != len(origHooks) {
		t.Errorf("round-trip security hook count = %d, want %d", len(rtHooks), len(origHooks))
	}

	// Verify compliance level count matches.
	origCompliance := cat.ComplianceLevels()
	rtCompliance := roundTripped.ComplianceLevels()
	if len(rtCompliance) != len(origCompliance) {
		t.Errorf("round-trip compliance count = %d, want %d",
			len(rtCompliance), len(origCompliance))
	}

	// Verify derivation mappings preserved.
	origTTC := cat.TierToCompliance()
	rtTTC := roundTripped.TierToCompliance()
	for tier, level := range origTTC {
		if rtTTC[tier] != level {
			t.Errorf("round-trip tier_to_compliance[%q] = %q, want %q",
				tier, rtTTC[tier], level)
		}
	}

	// Verify base packages preserved.
	origPkgs := cat.BasePackages()
	rtPkgs := roundTripped.BasePackages()
	if len(rtPkgs) != len(origPkgs) {
		t.Errorf("round-trip base packages count = %d, want %d",
			len(rtPkgs), len(origPkgs))
	}

	// Verify languages preserved.
	origLangs := cat.Languages()
	rtLangs := roundTripped.Languages()
	if len(rtLangs) != len(origLangs) {
		t.Errorf("round-trip languages count = %d, want %d",
			len(rtLangs), len(origLangs))
	}
}

func TestLoadUnifiedFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "defaults.yaml")

	content := `
tiers:
  test-tier:
    order: 99
    description: "Test tier from unified file"
    default_permission_preset: standard
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	cat, err := loadUnifiedFile(path)
	if err != nil {
		t.Fatalf("loadUnifiedFile() error: %v", err)
	}

	tier, ok := cat.tiers.Tiers["test-tier"]
	if !ok {
		t.Fatal("test-tier should be present after loading unified file")
	}
	if tier.Order != 99 {
		t.Errorf("test-tier order = %d, want 99", tier.Order)
	}
	if tier.Description != "Test tier from unified file" {
		t.Errorf("test-tier description = %q, want %q",
			tier.Description, "Test tier from unified file")
	}
}

func TestLoadUnifiedFile_InvalidYAML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	path := filepath.Join(dir, "defaults.yaml")

	content := `
tiers:
  broken:
    - this: [is not valid
    yaml because the bracket is unclosed
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	_, err := loadUnifiedFile(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML, got nil")
	}
	if !strings.Contains(err.Error(), "parsing unified defaults") {
		t.Errorf("error should mention parsing, got: %v", err)
	}
}

func TestLoadEmbeddedOnly(t *testing.T) {
	t.Parallel()

	cat, err := LoadEmbeddedOnly()
	if err != nil {
		t.Fatalf("LoadEmbeddedOnly() error: %v", err)
	}
	if cat == nil {
		t.Fatal("LoadEmbeddedOnly() returned nil catalog")
	}

	tiers := cat.TierDefs()
	if len(tiers) != 3 {
		t.Errorf("tier count = %d, want 3", len(tiers))
	}

	for _, name := range []string{"supply-chain-only", "standard", "full"} {
		if _, ok := tiers[name]; !ok {
			t.Errorf("expected tier %q to be present", name)
		}
	}
}

func TestGenerateDefaultsTemplate(t *testing.T) {
	t.Parallel()

	output, err := GenerateDefaultsTemplate()
	if err != nil {
		t.Fatalf("GenerateDefaultsTemplate() error: %v", err)
	}

	text := string(output)

	// Verify output starts with #.
	if !strings.HasPrefix(text, "#") {
		t.Error("template should start with #")
	}

	// Verify header is present.
	if !strings.Contains(text, "qsdev user defaults") {
		t.Error("template should contain 'qsdev user defaults'")
	}

	// Verify commented-out YAML content is present.
	if !strings.Contains(text, "tiers:") {
		t.Error("template should contain 'tiers:' (commented)")
	}

	// Verify all lines are comments.
	for i, line := range strings.Split(text, "\n") {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "#") {
			t.Errorf("line %d is not a comment: %q", i+1, line)
		}
	}

	// Verify other key sections appear.
	for _, section := range []string{"tools:", "security_hooks:", "compliance:", "profiles:"} {
		if !strings.Contains(text, section) {
			t.Errorf("template should contain %q", section)
		}
	}
}
