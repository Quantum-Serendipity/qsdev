package catalog

import (
	"testing"
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
		{"standard", "standard"},
		{"full", "enhanced"},
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

func containsStr(ss []string, s string) bool {
	for _, v := range ss {
		if v == s {
			return true
		}
	}
	return false
}
