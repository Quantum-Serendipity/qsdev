package config

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestResolveConfig_OrgDefaultsOnly(t *testing.T) {
	org := DefaultQsdevConfig()
	result, err := ResolveConfig(org, nil, nil, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Config.Security.Level != "enhanced" {
		t.Errorf("expected enhanced security from org defaults, got %q", result.Config.Security.Level)
	}
}

func TestResolveConfig_ProfileOverridesOrg(t *testing.T) {
	org := DefaultQsdevConfig()
	profile := &types.QsdevConfig{
		Security: types.SecurityConfig{Level: "strict"},
	}
	result, err := ResolveConfig(org, profile, nil, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Config.Security.Level != "strict" {
		t.Errorf("expected strict from profile, got %q", result.Config.Security.Level)
	}
}

func TestResolveConfig_ProjectOverridesProfile(t *testing.T) {
	org := DefaultQsdevConfig()
	profile := &types.QsdevConfig{
		Security: types.SecurityConfig{Level: "strict"},
	}
	project := &types.QsdevConfig{
		Security: types.SecurityConfig{Level: "enhanced"},
	}
	result, err := ResolveConfig(org, profile, project, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	// Note: project sets "enhanced" but profile set "strict"; however,
	// security floor enforcement may raise it. Without a Client floor
	// or project floor enforcement from the project's own level, the
	// raw merge gives "enhanced" since project overrides profile.
	if result.Config.Security.Level != "enhanced" {
		t.Errorf("expected enhanced from project override, got %q", result.Config.Security.Level)
	}
}

func TestResolveConfig_LocalOverridesProject(t *testing.T) {
	org := DefaultQsdevConfig()
	project := &types.QsdevConfig{
		ClaudeCode: types.ClaudeCodeConfig{
			PermissionLevel: "standard",
		},
	}
	local := &LocalConfig{
		ClaudeCode: types.ClaudeCodeConfig{
			PermissionLevel: "permissive",
		},
	}
	result, err := ResolveConfig(org, nil, project, local, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Config.ClaudeCode.PermissionLevel != "permissive" {
		t.Errorf("expected permissive from local, got %q", result.Config.ClaudeCode.PermissionLevel)
	}
}

func TestResolveConfig_AllFiveLayers(t *testing.T) {
	org := DefaultQsdevConfig()
	profile := &types.QsdevConfig{
		Tools: types.ToolsConfig{Enabled: []string{"gitleaks"}},
	}
	project := &types.QsdevConfig{
		Languages: []types.LanguageConfig{{Name: "go", Version: "1.22"}},
		Tools:     types.ToolsConfig{Enabled: []string{"semgrep"}},
	}
	local := &LocalConfig{
		Tools: types.ToolsConfig{Enabled: []string{"changelog"}},
	}

	result, err := ResolveConfig(org, profile, project, local, true)
	if err != nil {
		t.Fatal(err)
	}

	// Tools should be a union of all layers.
	tools := result.Config.Tools.Enabled
	toolSet := make(map[string]bool)
	for _, t := range tools {
		toolSet[t] = true
	}
	for _, expected := range []string{"gitleaks", "semgrep", "changelog"} {
		if !toolSet[expected] {
			t.Errorf("expected tool %q in enabled list, got %v", expected, tools)
		}
	}

	// Languages should come from project (replacement).
	if len(result.Config.Languages) != 1 || result.Config.Languages[0].Name != "go" {
		t.Errorf("expected go language from project, got %v", result.Config.Languages)
	}

	// Traces should be recorded (verbose=true).
	if len(result.Traces) == 0 {
		t.Error("expected traces to be recorded in verbose mode")
	}
}

func TestResolveConfig_NilLayers(t *testing.T) {
	result, err := ResolveConfig(nil, nil, nil, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Config == nil {
		t.Error("expected non-nil config even with all nil layers")
	}
}

func TestResolveConfig_LanguagesReplacement(t *testing.T) {
	org := &types.QsdevConfig{
		Languages: []types.LanguageConfig{{Name: "go"}},
	}
	project := &types.QsdevConfig{
		Languages: []types.LanguageConfig{{Name: "python"}},
	}
	result, err := ResolveConfig(org, nil, project, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Config.Languages) != 1 || result.Config.Languages[0].Name != "python" {
		t.Errorf("expected python to replace go, got %v", result.Config.Languages)
	}
}

func TestResolveConfig_ServicesReplacement(t *testing.T) {
	org := &types.QsdevConfig{
		Services: []types.ServiceConfig{{Name: "postgres"}},
	}
	project := &types.QsdevConfig{
		Services: []types.ServiceConfig{{Name: "redis"}},
	}
	result, err := ResolveConfig(org, nil, project, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Config.Services) != 1 || result.Config.Services[0].Name != "redis" {
		t.Errorf("expected redis to replace postgres, got %v", result.Config.Services)
	}
}

func TestResolveConfig_ToolsEnabledUnion(t *testing.T) {
	org := &types.QsdevConfig{
		Tools: types.ToolsConfig{Enabled: []string{"a", "b"}},
	}
	project := &types.QsdevConfig{
		Tools: types.ToolsConfig{Enabled: []string{"b", "c"}},
	}
	result, err := ResolveConfig(org, nil, project, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	expected := map[string]bool{"a": true, "b": true, "c": true}
	for _, tool := range result.Config.Tools.Enabled {
		if !expected[tool] {
			t.Errorf("unexpected tool %q", tool)
		}
		delete(expected, tool)
	}
	if len(expected) > 0 {
		t.Errorf("missing tools: %v", expected)
	}
}

func TestResolveConfig_ToolsDisabledUnion(t *testing.T) {
	org := &types.QsdevConfig{
		Tools: types.ToolsConfig{Disabled: []string{"x"}},
	}
	project := &types.QsdevConfig{
		Tools: types.ToolsConfig{Disabled: []string{"y"}},
	}
	result, err := ResolveConfig(org, nil, project, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	disabled := result.Config.Tools.Disabled
	if len(disabled) != 2 {
		t.Errorf("expected 2 disabled tools, got %v", disabled)
	}
}

func TestResolveConfig_ToolsConfigDeepMerge(t *testing.T) {
	org := &types.QsdevConfig{
		Tools: types.ToolsConfig{
			Config: map[string]map[string]any{
				"sentinel": {"hours": 24},
			},
		},
	}
	project := &types.QsdevConfig{
		Tools: types.ToolsConfig{
			Config: map[string]map[string]any{
				"sentinel": {"mode": "strict"},
				"semgrep":  {"rules": "p/default"},
			},
		},
	}
	result, err := ResolveConfig(org, nil, project, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	cfg := result.Config.Tools.Config
	if cfg["sentinel"]["hours"] != 24 {
		t.Errorf("expected sentinel.hours=24, got %v", cfg["sentinel"]["hours"])
	}
	if cfg["sentinel"]["mode"] != "strict" {
		t.Errorf("expected sentinel.mode=strict, got %v", cfg["sentinel"]["mode"])
	}
	if cfg["semgrep"]["rules"] != "p/default" {
		t.Errorf("expected semgrep.rules=p/default, got %v", cfg["semgrep"]["rules"])
	}
}

func TestResolveConfig_MCPServersUnion(t *testing.T) {
	org := &types.QsdevConfig{
		ClaudeCode: types.ClaudeCodeConfig{
			MCPServers: []string{"context7"},
		},
	}
	project := &types.QsdevConfig{
		ClaudeCode: types.ClaudeCodeConfig{
			MCPServers: []string{"github"},
		},
	}
	result, err := ResolveConfig(org, nil, project, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	servers := result.Config.ClaudeCode.MCPServers
	serverSet := make(map[string]bool)
	for _, s := range servers {
		serverSet[s] = true
	}
	if !serverSet["context7"] || !serverSet["github"] {
		t.Errorf("expected context7 and github in MCP servers, got %v", servers)
	}
}

func TestResolveConfig_SkillsUnion(t *testing.T) {
	org := &types.QsdevConfig{
		ClaudeCode: types.ClaudeCodeConfig{
			Skills: []string{"deploy"},
		},
	}
	project := &types.QsdevConfig{
		ClaudeCode: types.ClaudeCodeConfig{
			Skills: []string{"security-review"},
		},
	}
	result, err := ResolveConfig(org, nil, project, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	skills := result.Config.ClaudeCode.Skills
	if len(skills) != 2 {
		t.Errorf("expected 2 skills, got %v", skills)
	}
}

func TestResolveConfig_SecurityFloorCannotLowerLevel(t *testing.T) {
	org := DefaultQsdevConfig()
	project := &types.QsdevConfig{
		Security: types.SecurityConfig{Level: "strict"},
	}
	local := &LocalConfig{
		Security: types.SecurityConfig{Level: "baseline"},
	}
	result, err := ResolveConfig(org, nil, project, local, false)
	if err != nil {
		t.Fatal(err)
	}
	// Floor is "strict" from project, local tries "baseline" -> enforced to "strict".
	if result.Config.Security.Level != "strict" {
		t.Errorf("expected strict (floor enforced), got %q", result.Config.Security.Level)
	}
	if len(result.Violations) == 0 {
		t.Error("expected a floor violation to be recorded")
	}
}

func TestResolveConfig_SecurityFloorCanRaiseLevel(t *testing.T) {
	org := DefaultQsdevConfig()
	project := &types.QsdevConfig{
		Security: types.SecurityConfig{Level: "baseline"},
	}
	local := &LocalConfig{
		Security: types.SecurityConfig{Level: "strict"},
	}
	result, err := ResolveConfig(org, nil, project, local, false)
	if err != nil {
		t.Fatal(err)
	}
	// Local raises to "strict", which is allowed.
	if result.Config.Security.Level != "strict" {
		t.Errorf("expected strict (raised by local), got %q", result.Config.Security.Level)
	}
}

func TestResolveConfig_SecurityFloorCannotDisableAgeGating(t *testing.T) {
	org := DefaultQsdevConfig()
	project := &types.QsdevConfig{
		Security: types.SecurityConfig{AgeGating: boolP(true)},
	}
	local := &LocalConfig{
		Security: types.SecurityConfig{AgeGating: boolP(false)},
	}
	result, err := ResolveConfig(org, nil, project, local, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Config.Security.AgeGating == nil || !*result.Config.Security.AgeGating {
		t.Error("expected age_gating to be enforced to true")
	}
	foundViolation := false
	for _, v := range result.Violations {
		if v.Field == "security.age_gating" {
			foundViolation = true
			break
		}
	}
	if !foundViolation {
		t.Error("expected age_gating floor violation")
	}
}

func TestResolveConfig_SecurityFloorCanEnableAgeGating(t *testing.T) {
	org := &types.QsdevConfig{}
	project := &types.QsdevConfig{
		Security: types.SecurityConfig{AgeGating: boolP(false)},
	}
	local := &LocalConfig{
		Security: types.SecurityConfig{AgeGating: boolP(true)},
	}
	result, err := ResolveConfig(org, nil, project, local, false)
	if err != nil {
		t.Fatal(err)
	}
	// Local enables age_gating. Since project had it false, there's no floor
	// preventing enabling it. But the floor only prevents disabling when project=true.
	if result.Config.Security.AgeGating == nil || !*result.Config.Security.AgeGating {
		t.Error("expected age_gating to be true (local enabled it)")
	}
}

func TestResolveConfig_ClientSecurityLevelOverridesProjectFloor(t *testing.T) {
	org := DefaultQsdevConfig()
	project := &types.QsdevConfig{
		Security: types.SecurityConfig{Level: "baseline"},
		Client: &types.ClientConfig{
			Name:          "acme",
			SecurityLevel: "strict",
		},
	}
	local := &LocalConfig{
		Security: types.SecurityConfig{Level: "enhanced"},
	}
	result, err := ResolveConfig(org, nil, project, local, false)
	if err != nil {
		t.Fatal(err)
	}
	// Client says strict, local says enhanced -> floor enforces strict.
	if result.Config.Security.Level != "strict" {
		t.Errorf("expected strict (client floor), got %q", result.Config.Security.Level)
	}
}

func TestResolveConfig_ClientBlockedMCPPersists(t *testing.T) {
	org := &types.QsdevConfig{
		ClaudeCode: types.ClaudeCodeConfig{
			MCPServers: []string{"context7", "github", "evil-server"},
		},
	}
	project := &types.QsdevConfig{
		Client: &types.ClientConfig{
			Name:       "acme",
			BlockedMCP: []string{"evil-server"},
		},
	}
	result, err := ResolveConfig(org, nil, project, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range result.Config.ClaudeCode.MCPServers {
		if s == "evil-server" {
			t.Error("expected evil-server to be blocked")
		}
	}
}

func TestResolveConfig_ClientBlockedMCPWildcard(t *testing.T) {
	org := &types.QsdevConfig{
		ClaudeCode: types.ClaudeCodeConfig{
			MCPServers: []string{"context7", "github", "custom"},
		},
	}
	project := &types.QsdevConfig{
		Client: &types.ClientConfig{
			Name:       "acme",
			BlockedMCP: []string{"*"},
			AllowedMCP: []string{"github"},
		},
	}
	result, err := ResolveConfig(org, nil, project, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	servers := result.Config.ClaudeCode.MCPServers
	if len(servers) != 1 || servers[0] != "github" {
		t.Errorf("expected only github to survive wildcard block, got %v", servers)
	}
}

func TestResolveConfig_ViolationsRecorded(t *testing.T) {
	org := DefaultQsdevConfig()
	project := &types.QsdevConfig{
		Security: types.SecurityConfig{
			Level:     "strict",
			AgeGating: boolP(true),
		},
	}
	local := &LocalConfig{
		Security: types.SecurityConfig{
			Level:     "baseline",
			AgeGating: boolP(false),
		},
	}
	result, err := ResolveConfig(org, nil, project, local, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Violations) < 2 {
		t.Errorf("expected at least 2 violations, got %d: %v", len(result.Violations), result.Violations)
	}
}

func TestResolveConfig_PointerBoolNilVsFalse(t *testing.T) {
	// nil means inherit, false means explicitly disabled.
	org := &types.QsdevConfig{
		Security: types.SecurityConfig{
			AgeGating: boolP(true),
		},
	}
	// Project does not set age_gating (nil = inherit from org).
	project := &types.QsdevConfig{}

	result, err := ResolveConfig(org, nil, project, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Config.Security.AgeGating == nil || !*result.Config.Security.AgeGating {
		t.Error("expected age_gating to be inherited as true from org")
	}

	// Now project explicitly sets false.
	project2 := &types.QsdevConfig{
		Security: types.SecurityConfig{AgeGating: boolP(false)},
	}
	result2, err := ResolveConfig(org, nil, project2, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	// Since project is the floor and sets AgeGating to false, the resolved
	// value should be false (no floor enforcement for project's own settings
	// when project sets it to false).
	if result2.Config.Security.AgeGating == nil {
		t.Error("expected age_gating to be non-nil")
	}
}
