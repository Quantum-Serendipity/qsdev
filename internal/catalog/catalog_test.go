package catalog

import (
	"fmt"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
)

func loadTestCatalog(t *testing.T) *Catalog {
	t.Helper()
	cat, err := Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	return cat
}

// --- YAML Load Tests ---

func TestLoad_EmbeddedDefaults(t *testing.T) {
	t.Parallel()
	cat, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	if cat == nil {
		t.Fatal("Load() returned nil catalog")
	}
}

func TestLoad_Validation(t *testing.T) {
	t.Parallel()
	cat, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}
	errs := cat.Validate()
	if len(errs) > 0 {
		for _, e := range errs {
			t.Errorf("validation error: %s", e)
		}
	}
}

// --- Tier Tests ---

func TestTierOrder(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	order := cat.TierOrder()

	want := []string{"supply-chain-only", "standard", "full"}
	if len(order) != len(want) {
		t.Fatalf("TierOrder() = %v, want %v", order, want)
	}
	for i, name := range want {
		if order[i] != name {
			t.Errorf("TierOrder()[%d] = %q, want %q", i, order[i], name)
		}
	}
}

func TestTierDefs_AllPresent(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	for _, name := range []string{"supply-chain-only", "standard", "full"} {
		if _, ok := cat.TierDef(name); !ok {
			t.Errorf("tier %q not found", name)
		}
	}
}

func TestTierDefs_Descriptions(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)

	tests := []struct {
		name string
		want string
	}{
		{"supply-chain-only", "Package supply chain security + devenv sandbox; no Claude Code restrictions"},
		{"standard", "Supply chain deny rules + Claude Code governance + CLAUDE.md + gitleaks"},
		{"full", "Full tooling: MCP servers, agent tools, consulting workflows, AlwaysOn tools"},
	}
	for _, tt := range tests {
		def, _ := cat.TierDef(tt.name)
		if def.Description != tt.want {
			t.Errorf("tier %q description = %q, want %q", tt.name, def.Description, tt.want)
		}
	}
}

func TestTierDefs_PermissionPresets(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)

	tests := []struct {
		name string
		want string
	}{
		{"supply-chain-only", "supply-chain-only"},
		{"standard", "standard"},
		{"full", "standard"},
	}
	for _, tt := range tests {
		def, _ := cat.TierDef(tt.name)
		if def.DefaultPermissionPreset != tt.want {
			t.Errorf("tier %q default_permission_preset = %q, want %q",
				tt.name, def.DefaultPermissionPreset, tt.want)
		}
	}
}

func TestTierInheritance_StandardInheritsSupplyChainOnly(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	def, ok := cat.TierDef("standard")
	if !ok {
		t.Fatal("standard tier not found")
	}
	if def.Inherits != "supply-chain-only" {
		t.Errorf("standard.inherits = %q, want %q", def.Inherits, "supply-chain-only")
	}
}

func TestTierInheritance_FullInheritsStandard(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	def, ok := cat.TierDef("full")
	if !ok {
		t.Fatal("full tier not found")
	}
	if def.Inherits != "standard" {
		t.Errorf("full.inherits = %q, want %q", def.Inherits, "standard")
	}
}

// --- Tier Resolution Tests ---

func TestResolveTier_SupplyChainOnly(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	resolved, err := cat.ResolveTier("supply-chain-only")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Security.Level != "baseline" {
		t.Errorf("security.level = %q, want %q", resolved.Security.Level, "baseline")
	}
	if len(resolved.Tools.Enabled) != 0 {
		t.Errorf("tools.enabled = %v, want empty", resolved.Tools.Enabled)
	}
	if resolved.ClaudeCode.PermissionLevel != "supply-chain-only" {
		t.Errorf("claude_code.permission_level = %q, want %q",
			resolved.ClaudeCode.PermissionLevel, "supply-chain-only")
	}
}

func TestResolveTier_Standard(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	resolved, err := cat.ResolveTier("standard")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.ClaudeCode.PermissionLevel != "standard" {
		t.Errorf("claude_code.permission_level = %q, want %q",
			resolved.ClaudeCode.PermissionLevel, "standard")
	}
	if !containsStr(resolved.Tools.Enabled, "gitleaks") {
		t.Errorf("tools.enabled = %v, should contain gitleaks", resolved.Tools.Enabled)
	}
}

func TestResolveTier_Full(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	resolved, err := cat.ResolveTier("full")
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Security.Level != "enhanced" {
		t.Errorf("security.level = %q, want %q", resolved.Security.Level, "enhanced")
	}
	for _, tool := range []string{"semgrep", "gitleaks", "secretspec"} {
		if !containsStr(resolved.Tools.Enabled, tool) {
			t.Errorf("tools.enabled = %v, should contain %q", resolved.Tools.Enabled, tool)
		}
	}
	if !containsStr(resolved.ClaudeCode.MCPServers, "context7") {
		t.Errorf("mcp_servers = %v, should contain context7", resolved.ClaudeCode.MCPServers)
	}
}

func TestResolveTier_Unknown(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	_, err := cat.ResolveTier("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown tier")
	}
}

// --- Compliance Tests ---

func TestComplianceLevels_AllPresent(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	for _, name := range []string{"baseline", "enhanced", "strict"} {
		if _, ok := cat.ComplianceLevel(name); !ok {
			t.Errorf("compliance level %q not found", name)
		}
	}
}

func TestComplianceLevels_AgeGating(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)

	tests := []struct {
		name  string
		hours int
	}{
		{"baseline", 72},
		{"enhanced", 168},
		{"strict", 336},
	}
	for _, tt := range tests {
		def, _ := cat.ComplianceLevel(tt.name)
		if def.AgeGatingThresholdHours != tt.hours {
			t.Errorf("%s age_gating_threshold_hours = %d, want %d",
				tt.name, def.AgeGatingThresholdHours, tt.hours)
		}
	}
}

func TestComplianceLevels_RequiredHooks(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)

	tests := []struct {
		name  string
		count int
	}{
		{"baseline", 2},
		{"enhanced", 3},
		{"strict", 4},
	}
	for _, tt := range tests {
		def, _ := cat.ComplianceLevel(tt.name)
		if len(def.RequiredPreCommitHooks) != tt.count {
			t.Errorf("%s required hooks = %d, want %d",
				tt.name, len(def.RequiredPreCommitHooks), tt.count)
		}
	}
}

// --- Profile Tests ---

func TestProfiles_AllPresent(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	for _, name := range []string{"supply-chain-only", "standard", "full"} {
		if _, ok := cat.Profile(name); !ok {
			t.Errorf("profile %q not found", name)
		}
	}
}

func TestProfileAliases(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	aliases := cat.ProfileAliases()

	tests := []struct {
		alias  string
		target string
	}{
		{"startup-fast", "standard"},
		{"consulting-default", "full"},
	}
	for _, tt := range tests {
		if aliases[tt.alias] != tt.target {
			t.Errorf("alias %q = %q, want %q", tt.alias, aliases[tt.alias], tt.target)
		}
	}
}

func TestProfile_FullTools(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	p, ok := cat.Profile("full")
	if !ok {
		t.Fatal("full profile not found")
	}
	if p.Tools == nil {
		t.Fatal("full profile has nil tools")
	}
	for _, tool := range []string{"semgrep", "gitleaks", "secretspec"} {
		if !containsStr(p.Tools.Enabled, tool) {
			t.Errorf("full profile tools = %v, should contain %q", p.Tools.Enabled, tool)
		}
	}
}

// --- Project Profile Tests ---

func TestProjectProfiles_AllPresent(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	expected := []string{
		"go-web", "ts-fullstack", "ts-backend", "python-data", "python-web",
		"rust-cli", "rust-web", "java-web", "elixir-web", "dotnet-web",
	}
	for _, name := range expected {
		if _, ok := cat.ProjectProfile(name); !ok {
			t.Errorf("project profile %q not found", name)
		}
	}
}

func TestProjectProfile_GoWeb(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	p, ok := cat.ProjectProfile("go-web")
	if !ok {
		t.Fatal("go-web not found")
	}
	if p.Tier != "full" {
		t.Errorf("tier = %q, want full", p.Tier)
	}
	if len(p.Languages) != 1 || p.Languages[0].Name != "go" {
		t.Errorf("languages = %v, want [go]", p.Languages)
	}
	if p.Languages[0].Version != "1.24" {
		t.Errorf("go version = %q, want 1.24", p.Languages[0].Version)
	}
}

// --- Tool Tests ---

func TestTools_CoreToolsPresent(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	core := []string{
		"attach-guard", "agent-postmortem", "version-sentinel", "semble",
		"trail-of-bits-skills", "secretspec", "context7", "github-mcp",
		"socket-dev-mcp", "postgres-mcp", "changelog", "semgrep",
		"gitleaks", "container-security", "license-compliance", "commitlint",
	}
	for _, name := range core {
		if _, ok := cat.Tool(name); !ok {
			t.Errorf("tool %q not found", name)
		}
	}
}

func TestTools_Count(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	tools := cat.Tools()
	if len(tools) < 37 {
		t.Errorf("expected at least 37 tools, got %d", len(tools))
	}
}

func TestTool_SemgrepMetadata(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	tool, ok := cat.Tool("semgrep")
	if !ok {
		t.Fatal("semgrep not found")
	}
	if tool.DisplayName != "Semgrep SAST" {
		t.Errorf("display_name = %q, want %q", tool.DisplayName, "Semgrep SAST")
	}
	if tool.Category != "security" {
		t.Errorf("category = %q, want security", tool.Category)
	}
	if tool.DefaultPolicy != "always-on" {
		t.Errorf("default_policy = %q, want always-on", tool.DefaultPolicy)
	}
}

// --- Security Tests ---

func TestSecurityHooks(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	hooks := cat.SecurityHooks()
	expected := []string{"ripsecrets", "check-added-large-files", "no-commit-to-branch",
		"check-merge-conflicts", "shellcheck", "statix"}
	if len(hooks) != len(expected) {
		t.Fatalf("SecurityHooks() count = %d, want %d", len(hooks), len(expected))
	}
	for i, h := range expected {
		if hooks[i] != h {
			t.Errorf("SecurityHooks()[%d] = %q, want %q", i, hooks[i], h)
		}
	}
}

func TestBasePackages(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	pkgs := cat.BasePackages()
	expected := []string{"git", "jq", "curl", "coreutils"}
	if len(pkgs) != len(expected) {
		t.Fatalf("BasePackages() count = %d, want %d", len(pkgs), len(expected))
	}
	for i, p := range expected {
		if pkgs[i] != p {
			t.Errorf("BasePackages()[%d] = %q, want %q", i, pkgs[i], p)
		}
	}
}

func TestUnsetVars_ContainsKnownCredentials(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	vars := cat.UnsetVars()
	for _, v := range []string{"AWS_SECRET_ACCESS_KEY", "GITHUB_TOKEN", "VAULT_TOKEN", "DATABASE_PASSWORD"} {
		if !containsStr(vars, v) {
			t.Errorf("UnsetVars() should contain %q", v)
		}
	}
}

func TestCustomHooks(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	hooks := cat.CustomHooks()
	if len(hooks) != 2 {
		t.Fatalf("CustomHooks() count = %d, want 2", len(hooks))
	}
	ids := make(map[string]bool)
	for _, h := range hooks {
		ids[h.ID] = true
	}
	if !ids["lock-file-audit"] {
		t.Error("missing custom hook lock-file-audit")
	}
	if !ids["nix-secrets-check"] {
		t.Error("missing custom hook nix-secrets-check")
	}
}

// --- Hook Tier Tests ---

func TestHookTierOrder(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	order := cat.HookTierOrder()
	want := []string{"baseline", "enhanced", "specialized"}
	if len(order) != len(want) {
		t.Fatalf("HookTierOrder() = %v, want %v", order, want)
	}
	for i, name := range want {
		if order[i] != name {
			t.Errorf("HookTierOrder()[%d] = %q, want %q", i, order[i], name)
		}
	}
}

func TestHookTiers_BaselineHooks(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	tiers := cat.HookTiers()
	baseline := tiers["baseline"]
	for _, h := range []string{"ripsecrets", "gitleaks", "check-added-large-files"} {
		if !containsStr(baseline, h) {
			t.Errorf("baseline hooks should contain %q, got %v", h, baseline)
		}
	}
}

// --- Derivation Tests ---

func TestTierToCompliance(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	m := cat.TierToCompliance()

	tests := []struct {
		tier       string
		compliance string
	}{
		{"supply-chain-only", "baseline"},
		{"standard", "enhanced"},
		{"full", "strict"},
	}
	for _, tt := range tests {
		if m[tt.tier] != tt.compliance {
			t.Errorf("TierToCompliance[%q] = %q, want %q", tt.tier, m[tt.tier], tt.compliance)
		}
	}
}

func TestTierToEnabledTools(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	m := cat.TierToEnabledTools()

	if len(m["supply-chain-only"]) != 0 {
		t.Errorf("supply-chain-only tools = %v, want empty", m["supply-chain-only"])
	}
	if len(m["standard"]) != 1 || m["standard"][0] != "gitleaks" {
		t.Errorf("standard tools = %v, want [gitleaks]", m["standard"])
	}
	fullTools := m["full"]
	for _, tool := range []string{"semgrep", "gitleaks", "secretspec"} {
		if !containsStr(fullTools, tool) {
			t.Errorf("full tools = %v, should contain %q", fullTools, tool)
		}
	}
}

func TestDefaultMCPServers(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	servers := cat.DefaultMCPServers()
	want := []string{"context7", "github", "socket", "semble"}
	if len(servers) != len(want) {
		t.Fatalf("DefaultMCPServers() = %v, want %v", servers, want)
	}
	for i, s := range want {
		if servers[i] != s {
			t.Errorf("DefaultMCPServers()[%d] = %q, want %q", i, servers[i], s)
		}
	}
}

func TestDefaultAgentToolConfig(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	cfg := cat.DefaultAgentToolConfig()
	if !cfg.PostmortemEnabled {
		t.Error("postmortem_enabled should be true")
	}
	if !cfg.VersionSentinel {
		t.Error("version_sentinel should be true")
	}
	if cfg.VersionSentinelHours != 24 {
		t.Errorf("version_sentinel_hours = %d, want 24", cfg.VersionSentinelHours)
	}
	if !cfg.SembleEnabled {
		t.Error("semble_enabled should be true")
	}
	if cfg.SembleMode != "both" {
		t.Errorf("semble_mode = %q, want both", cfg.SembleMode)
	}
}

func TestDerivations_CrossReferenceIntegrity(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)

	complianceLevels := cat.ComplianceLevels()
	for tier, level := range cat.TierToCompliance() {
		if _, ok := complianceLevels[level]; !ok {
			t.Errorf("tier_to_compliance: tier %q maps to unknown compliance level %q", tier, level)
		}
	}

	tools := cat.Tools()
	for tier, toolNames := range cat.TierToEnabledTools() {
		for _, toolName := range toolNames {
			if _, ok := tools[toolName]; !ok {
				t.Errorf("tier_to_enabled_tools: tier %q references unknown tool %q", tier, toolName)
			}
		}
	}
}

// --- Validation List Tests ---

func TestLanguages(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	langs := cat.Languages()
	if len(langs) < 27 {
		t.Errorf("Languages() count = %d, want >= 27", len(langs))
	}
	for _, l := range []string{"go", "javascript", "python", "rust"} {
		if !containsStr(langs, l) {
			t.Errorf("Languages() should contain %q", l)
		}
	}
}

func TestCoreLanguages(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	langs := cat.CoreLanguages()
	if len(langs) != 8 {
		t.Errorf("CoreLanguages() count = %d, want 8", len(langs))
	}
}

func TestServices(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	svcs := cat.Services()
	if len(svcs) != 6 {
		t.Errorf("Services() count = %d, want 6", len(svcs))
	}
}

func TestPermissionPresets(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	presets := cat.PermissionPresets()
	if len(presets) != 5 {
		t.Errorf("PermissionPresets() count = %d, want 5", len(presets))
	}
}

// --- Merge Tests ---

func TestMergeCatalogs_OverlayAddsTier(t *testing.T) {
	t.Parallel()
	base := loadTestCatalog(t)

	overlay := &Catalog{}
	overlay.tiers.Tiers = map[string]TierDef{
		"custom": {
			Order:       4,
			Description: "Custom tier",
		},
	}

	merged := MergeCatalogs(base, overlay)

	if _, ok := merged.TierDef("custom"); !ok {
		t.Error("merged catalog should have custom tier")
	}
	if _, ok := merged.TierDef("full"); !ok {
		t.Error("merged catalog should still have full tier")
	}
}

func TestMergeCatalogs_OverlayAddsTool(t *testing.T) {
	t.Parallel()
	base := loadTestCatalog(t)

	overlay := &Catalog{}
	overlay.tools.Tools = map[string]ToolDef{
		"custom-scanner": {
			DisplayName:   "Custom Scanner",
			Category:      "security",
			DefaultPolicy: "opt-in",
		},
	}

	merged := MergeCatalogs(base, overlay)

	if _, ok := merged.Tool("custom-scanner"); !ok {
		t.Error("merged catalog should have custom-scanner")
	}
	if _, ok := merged.Tool("semgrep"); !ok {
		t.Error("merged catalog should still have semgrep")
	}
}

// --- Backward Compatibility Contract Tests ---

func TestContract_TierNames(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	for _, name := range []string{"supply-chain-only", "standard", "full"} {
		if _, ok := cat.TierDef(name); !ok {
			t.Errorf("BACKWARD COMPAT: tier %q must exist", name)
		}
	}
}

func TestContract_ComplianceLevelNames(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	for _, name := range []string{"baseline", "enhanced", "strict"} {
		if _, ok := cat.ComplianceLevel(name); !ok {
			t.Errorf("BACKWARD COMPAT: compliance level %q must exist", name)
		}
	}
}

func TestContract_ProfileNames(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	for _, name := range []string{"supply-chain-only", "standard", "full"} {
		if _, ok := cat.Profile(name); !ok {
			t.Errorf("BACKWARD COMPAT: profile %q must exist", name)
		}
	}
}

func TestContract_ProfileAliases(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	aliases := cat.ProfileAliases()
	if aliases["startup-fast"] != "standard" {
		t.Error("BACKWARD COMPAT: alias startup-fast must resolve to standard")
	}
	if aliases["consulting-default"] != "full" {
		t.Error("BACKWARD COMPAT: alias consulting-default must resolve to full")
	}
}

func TestUnsetVars_SupersetOfKnownCredentialVars(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	unsetVars := cat.UnsetVars()

	unsetSet := make(map[string]bool, len(unsetVars))
	for _, v := range unsetVars {
		unsetSet[v] = true
	}

	for _, v := range secrets.KnownCredentialVars {
		if !unsetSet[v] {
			t.Errorf("secrets.KnownCredentialVars contains %q but security.yaml unset_vars does not", v)
		}
	}
}

func containsStr(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}

func newMinimalCatalog() *Catalog {
	cat := &Catalog{}
	cat.tiers.Tiers = map[string]TierDef{}
	cat.compliance.Levels = map[string]ComplianceLevelDef{}
	cat.profiles.Profiles = map[string]ProfileDef{}
	cat.profiles.Aliases = map[string]string{}
	cat.projectProfiles.Profiles = map[string]ProjectProfileDef{}
	cat.tools.Tools = map[string]ToolDef{}
	cat.hookTiers.Tiers = map[string][]string{}
	cat.derivations.TierToCompliance = map[string]string{}
	cat.derivations.TierToEnabledTools = map[string][]string{}
	return cat
}

func hasValidationError(errs []CatalogError, substr string) bool {
	for _, e := range errs {
		if strings.Contains(e.Error(), substr) {
			return true
		}
	}
	return false
}

// --- Negative Validation Tests ---

func TestValidate_CircularTierInheritance(t *testing.T) {
	t.Parallel()
	cat := newMinimalCatalog()
	cat.tiers.Tiers["alpha"] = TierDef{Order: 1, Description: "A", Inherits: "beta"}
	cat.tiers.Tiers["beta"] = TierDef{Order: 2, Description: "B", Inherits: "gamma"}
	cat.tiers.Tiers["gamma"] = TierDef{Order: 3, Description: "C", Inherits: "alpha"}

	errs := cat.Validate()
	if !hasValidationError(errs, "circular inheritance") {
		t.Errorf("expected circular inheritance error, got: %v", errs)
	}
}

func TestValidate_DeepInheritanceChain(t *testing.T) {
	t.Parallel()
	cat := newMinimalCatalog()
	for i := 0; i <= 11; i++ {
		name := fmt.Sprintf("tier-%d", i)
		def := TierDef{Order: i + 1, Description: name}
		if i > 0 {
			def.Inherits = fmt.Sprintf("tier-%d", i-1)
		}
		cat.tiers.Tiers[name] = def
	}

	errs := cat.Validate()
	if !hasValidationError(errs, "exceeds maximum depth") {
		t.Errorf("expected max depth error, got: %v", errs)
	}
}

func TestValidate_TierInheritsUnknown(t *testing.T) {
	t.Parallel()
	cat := newMinimalCatalog()
	cat.tiers.Tiers["orphan"] = TierDef{Order: 1, Description: "Orphan", Inherits: "ghost"}

	errs := cat.Validate()
	if !hasValidationError(errs, `inherits unknown tier "ghost"`) {
		t.Errorf("expected unknown inherits error, got: %v", errs)
	}
}

func TestValidate_TierMissingDescription(t *testing.T) {
	t.Parallel()
	cat := newMinimalCatalog()
	cat.tiers.Tiers["no-desc"] = TierDef{Order: 99}

	errs := cat.Validate()
	if !hasValidationError(errs, "missing description") {
		t.Errorf("expected missing description error, got: %v", errs)
	}
}

func TestValidate_TierZeroOrder(t *testing.T) {
	t.Parallel()
	cat := newMinimalCatalog()
	cat.tiers.Tiers["zero-order"] = TierDef{Description: "has desc"}

	errs := cat.Validate()
	if !hasValidationError(errs, "order must be > 0") {
		t.Errorf("expected zero order error, got: %v", errs)
	}
}

func TestValidate_ProfileReferencesUnknownTier(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	cat.profiles.Profiles["broken"] = ProfileDef{Tier: "nonexistent"}

	errs := cat.Validate()
	if !hasValidationError(errs, `references unknown tier "nonexistent"`) {
		t.Errorf("expected broken profile tier error, got: %v", errs)
	}
}

func TestValidate_BrokenProfileAlias(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	cat.profiles.Aliases["bad-alias"] = "nonexistent-profile"

	errs := cat.Validate()
	if !hasValidationError(errs, "aliases.bad-alias") {
		t.Errorf("expected broken alias error, got: %v", errs)
	}
}

func TestValidate_DerivationUnknownTier(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	cat.derivations.TierToCompliance["ghost-tier"] = "baseline"

	errs := cat.Validate()
	if !hasValidationError(errs, `references unknown tier "ghost-tier"`) {
		t.Errorf("expected unknown derivation tier error, got: %v", errs)
	}
}

func TestValidate_DerivationUnknownComplianceLevel(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	cat.derivations.TierToCompliance["full"] = "nonexistent"

	errs := cat.Validate()
	if !hasValidationError(errs, `unknown compliance level "nonexistent"`) {
		t.Errorf("expected unknown compliance error, got: %v", errs)
	}
}

func TestValidate_DerivationUnknownTool(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	cat.derivations.TierToEnabledTools["full"] = []string{"fake-tool"}

	errs := cat.Validate()
	if !hasValidationError(errs, `unknown tool "fake-tool"`) {
		t.Errorf("expected unknown tool error, got: %v", errs)
	}
}

func TestValidate_BrokenHookTierOrder(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	cat.hookTiers.TierOrder = append(cat.hookTiers.TierOrder, "phantom-tier")

	errs := cat.Validate()
	if !hasValidationError(errs, `"phantom-tier"`) {
		t.Errorf("expected broken hook tier order error, got: %v", errs)
	}
}

func TestValidate_BrokenProjectProfileTier(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	cat.projectProfiles.Profiles["broken-pp"] = ProjectProfileDef{Tier: "nonexistent"}

	errs := cat.Validate()
	if !hasValidationError(errs, `references unknown tier "nonexistent"`) {
		t.Errorf("expected broken project profile tier error, got: %v", errs)
	}
}

// --- Multi-Level Inheritance Tests ---

func TestResolveTier_ThreeLevelInheritance(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)
	resolved, err := cat.ResolveTier("full")
	if err != nil {
		t.Fatal(err)
	}

	if resolved.Security.AgeGating == nil || !*resolved.Security.AgeGating {
		t.Error("full should inherit age_gating=true from supply-chain-only")
	}
	if resolved.Security.ScriptBlocking == nil || !*resolved.Security.ScriptBlocking {
		t.Error("full should inherit script_blocking=true from supply-chain-only")
	}
	if resolved.Security.LockEnforcement == nil || !*resolved.Security.LockEnforcement {
		t.Error("full should inherit lock_enforcement=true from supply-chain-only")
	}
	if resolved.Security.Level != "enhanced" {
		t.Errorf("security.level = %q, want enhanced (full's override)", resolved.Security.Level)
	}

	if !containsStr(resolved.Tools.Enabled, "gitleaks") {
		t.Error("full should inherit gitleaks from standard")
	}
	if !containsStr(resolved.Tools.Enabled, "semgrep") {
		t.Error("full should have semgrep from its own definition")
	}

	gitleaksCount := 0
	for _, tool := range resolved.Tools.Enabled {
		if tool == "gitleaks" {
			gitleaksCount++
		}
	}
	if gitleaksCount != 1 {
		t.Errorf("gitleaks appears %d times, want 1 (deduplication)", gitleaksCount)
	}

	if !containsStr(resolved.ClaudeCode.MCPServers, "context7") {
		t.Error("full should have context7 MCP server")
	}
}

// --- Additional Merge Tests ---

func TestMergeCatalogs_OverlayReplacesTier(t *testing.T) {
	t.Parallel()
	base := loadTestCatalog(t)

	overlay := &Catalog{}
	overlay.tiers.Tiers = map[string]TierDef{
		"standard": {Order: 2, Description: "Overridden standard", DefaultPermissionPreset: "custom"},
	}

	merged := MergeCatalogs(base, overlay)

	def, ok := merged.TierDef("standard")
	if !ok {
		t.Fatal("standard tier should exist after merge")
	}
	if def.Description != "Overridden standard" {
		t.Errorf("description = %q, want overlay value", def.Description)
	}
	if def.DefaultPermissionPreset != "custom" {
		t.Errorf("permission_preset = %q, want custom", def.DefaultPermissionPreset)
	}

	if _, ok := merged.TierDef("full"); !ok {
		t.Error("full tier should still exist after merge")
	}
}

func TestMergeCatalogs_OverlayReplacesStringSlice(t *testing.T) {
	t.Parallel()
	base := loadTestCatalog(t)

	overlay := &Catalog{}
	overlay.security.Hooks.Default = []string{"custom-hook"}

	merged := MergeCatalogs(base, overlay)
	hooks := merged.SecurityHooks()

	if len(hooks) != 1 || hooks[0] != "custom-hook" {
		t.Errorf("SecurityHooks() = %v, want [custom-hook]", hooks)
	}
}

func TestMergeCatalogs_CustomHookOverrideByID(t *testing.T) {
	t.Parallel()

	base := &Catalog{}
	base.tiers.Tiers = map[string]TierDef{}
	base.compliance.Levels = map[string]ComplianceLevelDef{}
	base.profiles.Profiles = map[string]ProfileDef{}
	base.profiles.Aliases = map[string]string{}
	base.projectProfiles.Profiles = map[string]ProjectProfileDef{}
	base.tools.Tools = map[string]ToolDef{}
	base.hookTiers.Tiers = map[string][]string{}
	base.derivations.TierToCompliance = map[string]string{}
	base.derivations.TierToEnabledTools = map[string][]string{}
	base.security.CustomHooks = []CustomHookDef{
		{ID: "hook-a", Name: "Hook A base", Language: "system"},
		{ID: "hook-b", Name: "Hook B base", Language: "system"},
	}

	overlay := &Catalog{}
	overlay.security.CustomHooks = []CustomHookDef{
		{ID: "hook-b", Name: "Hook B overlay", Language: "python"},
		{ID: "hook-c", Name: "Hook C new", Language: "system"},
	}

	merged := MergeCatalogs(base, overlay)
	hooks := merged.CustomHooks()

	if len(hooks) != 3 {
		t.Fatalf("CustomHooks count = %d, want 3", len(hooks))
	}
	if hooks[0].ID != "hook-a" || hooks[0].Name != "Hook A base" {
		t.Errorf("hooks[0] = %+v, want hook-a from base", hooks[0])
	}
	if hooks[1].ID != "hook-b" || hooks[1].Name != "Hook B overlay" || hooks[1].Language != "python" {
		t.Errorf("hooks[1] = %+v, want hook-b from overlay", hooks[1])
	}
	if hooks[2].ID != "hook-c" || hooks[2].Name != "Hook C new" {
		t.Errorf("hooks[2] = %+v, want hook-c from overlay", hooks[2])
	}
}

func TestMergeCatalogs_ValidationPartialOverride(t *testing.T) {
	t.Parallel()
	base := loadTestCatalog(t)
	baseServices := base.Services()

	overlay := &Catalog{}
	overlay.validation.Languages.All = []string{"go", "rust"}

	merged := MergeCatalogs(base, overlay)

	langs := merged.Languages()
	if len(langs) != 2 || langs[0] != "go" || langs[1] != "rust" {
		t.Errorf("Languages() = %v, want [go rust]", langs)
	}

	svcs := merged.Services()
	if len(svcs) != len(baseServices) {
		t.Errorf("Services() count = %d, want %d (base preserved)", len(svcs), len(baseServices))
	}
}

// --- Validation Accessor Tests ---

func TestValidationAccessors(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)

	t.Run("KeepVars", func(t *testing.T) {
		t.Parallel()
		vars := cat.KeepVars()
		if len(vars) < 5 {
			t.Errorf("KeepVars() count = %d, want >= 5", len(vars))
		}
		for _, v := range []string{"TERM", "HOME", "USER"} {
			if !containsStr(vars, v) {
				t.Errorf("KeepVars() should contain %q", v)
			}
		}
	})

	t.Run("PackageManagers_node", func(t *testing.T) {
		t.Parallel()
		pms := cat.PackageManagers("node")
		if len(pms) != 4 {
			t.Fatalf("PackageManagers(node) count = %d, want 4", len(pms))
		}
	})

	t.Run("PackageManagers_python", func(t *testing.T) {
		t.Parallel()
		pms := cat.PackageManagers("python")
		if len(pms) != 3 {
			t.Fatalf("PackageManagers(python) count = %d, want 3", len(pms))
		}
	})

	t.Run("PackageManagers_unknown", func(t *testing.T) {
		t.Parallel()
		pms := cat.PackageManagers("brainfuck")
		if pms != nil {
			t.Errorf("PackageManagers(brainfuck) = %v, want nil", pms)
		}
	})

	t.Run("HookPresets", func(t *testing.T) {
		t.Parallel()
		presets := cat.HookPresets()
		if len(presets) != 4 {
			t.Errorf("HookPresets() count = %d, want 4", len(presets))
		}
		for _, p := range []string{"auto-format", "safety-block", "pre-commit", "audit-log"} {
			if !containsStr(presets, p) {
				t.Errorf("HookPresets() should contain %q", p)
			}
		}
	})

	t.Run("SecurityLevels", func(t *testing.T) {
		t.Parallel()
		levels := cat.SecurityLevels()
		if len(levels) != 3 {
			t.Errorf("SecurityLevels() count = %d, want 3", len(levels))
		}
	})

	t.Run("DataClassifications", func(t *testing.T) {
		t.Parallel()
		dc := cat.DataClassifications()
		if len(dc) != 3 {
			t.Errorf("DataClassifications() count = %d, want 3", len(dc))
		}
	})

	t.Run("ToolCategories", func(t *testing.T) {
		t.Parallel()
		cats := cat.ToolCategories()
		if len(cats) != 4 {
			t.Fatalf("ToolCategories() count = %d, want 4", len(cats))
		}
		ids := make(map[string]bool)
		for _, c := range cats {
			ids[c.ID] = true
		}
		for _, id := range []string{"security", "ai-agent", "devex", "infrastructure"} {
			if !ids[id] {
				t.Errorf("ToolCategories() missing id %q", id)
			}
		}
	})
}

// --- End-to-End Integration Test ---

func TestEndToEnd_TierToComplianceChain(t *testing.T) {
	t.Parallel()
	cat := loadTestCatalog(t)

	compliance := cat.TierToCompliance()["full"]
	if compliance != "strict" {
		t.Fatalf("TierToCompliance[full] = %q, want strict", compliance)
	}

	level, ok := cat.ComplianceLevel(compliance)
	if !ok {
		t.Fatalf("ComplianceLevel(%q) not found", compliance)
	}
	if level.AgeGatingThresholdHours != 336 {
		t.Errorf("strict age gating = %d, want 336", level.AgeGatingThresholdHours)
	}
	if len(level.RequiredPreCommitHooks) != 4 {
		t.Errorf("strict hooks count = %d, want 4", len(level.RequiredPreCommitHooks))
	}
	if level.MCPServerPolicy != "explicit-only" {
		t.Errorf("strict MCP policy = %q, want explicit-only", level.MCPServerPolicy)
	}

	enabledTools := cat.TierToEnabledTools()["full"]
	tools := cat.Tools()
	for _, toolName := range enabledTools {
		if _, ok := tools[toolName]; !ok {
			t.Errorf("enabled tool %q not found in tools catalog", toolName)
		}
	}
	for _, expected := range []string{"semgrep", "gitleaks", "secretspec"} {
		if !containsStr(enabledTools, expected) {
			t.Errorf("full tier enabled tools %v should contain %q", enabledTools, expected)
		}
	}
}
