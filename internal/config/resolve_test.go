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

// TestResolveConfig_RegistryProxyPathsSurvives is a regression test for
// BL-P1-11: cloneQsdevConfig and deepMerge previously copied every InfraConfig
// field EXCEPT RegistryProxyPaths, so the field was silently dropped on the
// first clone during resolution even though pkg/ecosystem/helpers.go consumes it.
func TestResolveConfig_RegistryProxyPathsSurvives(t *testing.T) {
	project := &types.QsdevConfig{
		Version: types.ConfigVersionCurrent,
		Infrastructure: types.InfraConfig{
			RegistryProxy:      "https://proxy.example.com",
			RegistryProxyPaths: map[string]string{"npm": "/repository/npm", "pypi": "/repository/pypi"},
		},
	}

	result, err := ResolveConfig(nil, nil, project, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	got := result.Config.Infrastructure.RegistryProxyPaths
	if got == nil {
		t.Fatal("RegistryProxyPaths was dropped during ResolveConfig (want it to survive)")
	}
	if got["npm"] != "/repository/npm" {
		t.Errorf("RegistryProxyPaths[npm] = %q, want %q", got["npm"], "/repository/npm")
	}
	if got["pypi"] != "/repository/pypi" {
		t.Errorf("RegistryProxyPaths[pypi] = %q, want %q", got["pypi"], "/repository/pypi")
	}

	// The clone must be independent of the input map (no aliasing).
	got["npm"] = "mutated"
	if project.Infrastructure.RegistryProxyPaths["npm"] != "/repository/npm" {
		t.Error("mutating resolved RegistryProxyPaths mutated the input project config (aliased, not cloned)")
	}
}

// TestResolveConfig_RegistryProxyPathsMerge verifies deepMerge unions
// RegistryProxyPaths from a lower-priority layer with an overlay (BL-P1-11).
func TestResolveConfig_RegistryProxyPathsMerge(t *testing.T) {
	org := &types.QsdevConfig{
		Infrastructure: types.InfraConfig{
			RegistryProxyPaths: map[string]string{"npm": "/org/npm"},
		},
	}
	project := &types.QsdevConfig{
		Version: types.ConfigVersionCurrent,
		Infrastructure: types.InfraConfig{
			RegistryProxyPaths: map[string]string{"cargo": "/project/cargo"},
		},
	}

	result, err := ResolveConfig(org, nil, project, nil, false)
	if err != nil {
		t.Fatal(err)
	}

	got := result.Config.Infrastructure.RegistryProxyPaths
	if got["npm"] != "/org/npm" {
		t.Errorf("RegistryProxyPaths[npm] = %q, want %q (org layer lost)", got["npm"], "/org/npm")
	}
	if got["cargo"] != "/project/cargo" {
		t.Errorf("RegistryProxyPaths[cargo] = %q, want %q (project layer lost)", got["cargo"], "/project/cargo")
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

// hasViolation reports whether a FloorViolation was recorded for the given field.
func hasViolation(vs []FloorViolation, field string) bool {
	for _, v := range vs {
		if v.Field == field {
			return true
		}
	}
	return false
}

// TestResolveConfig_ComplianceBoolFloor is a regression test for #S6: the four
// security bools must be floored against the compliance level a project
// declares (client.security_level), mirroring how security.level itself is
// floored. Before the fix, enforceBoolFloor only saw the project's own bool
// (nil when unset), so a layer-5 local override to false silently disabled a
// compliance-mandated control while security.level still read "strict".
func TestResolveConfig_ComplianceBoolFloor(t *testing.T) {
	tests := []struct {
		name        string
		field       string
		localOff    func(*types.SecurityConfig)
		getResolved func(*types.QsdevConfig) *bool
	}{
		{
			name:        "script_blocking",
			field:       "security.script_blocking",
			localOff:    func(s *types.SecurityConfig) { s.ScriptBlocking = boolP(false) },
			getResolved: func(c *types.QsdevConfig) *bool { return c.Security.ScriptBlocking },
		},
		{
			name:        "age_gating",
			field:       "security.age_gating",
			localOff:    func(s *types.SecurityConfig) { s.AgeGating = boolP(false) },
			getResolved: func(c *types.QsdevConfig) *bool { return c.Security.AgeGating },
		},
		{
			name:        "lock_enforcement",
			field:       "security.lock_enforcement",
			localOff:    func(s *types.SecurityConfig) { s.LockEnforcement = boolP(false) },
			getResolved: func(c *types.QsdevConfig) *bool { return c.Security.LockEnforcement },
		},
		{
			name:        "vuln_scanning",
			field:       "security.vuln_scanning",
			localOff:    func(s *types.SecurityConfig) { s.VulnScanning = boolP(false) },
			getResolved: func(c *types.QsdevConfig) *bool { return c.Security.VulnScanning },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org := DefaultQsdevConfig()
			// Project declares a strict compliance level but leaves the
			// individual security bools unset.
			project := &types.QsdevConfig{
				Client: &types.ClientConfig{Name: "acme", SecurityLevel: "strict"},
			}
			// Local override tries to disable the compliance-mandated control.
			local := &LocalConfig{}
			tt.localOff(&local.Security)

			result, err := ResolveConfig(org, nil, project, local, false)
			if err != nil {
				t.Fatal(err)
			}

			// Level must still read strict.
			if result.Config.Security.Level != "strict" {
				t.Errorf("security.level = %q, want strict", result.Config.Security.Level)
			}
			// The control must be enforced back on.
			got := tt.getResolved(result.Config)
			if got == nil || !*got {
				t.Errorf("%s: resolved value not enforced to true (got %v)", tt.field, got)
			}
			// A floor violation must be recorded.
			if !hasViolation(result.Violations, tt.field) {
				t.Errorf("%s: expected a floor violation, got %v", tt.field, result.Violations)
			}
		})
	}
}

// TestResolveConfig_ProjectBoolFloorStillEnforced is a regression guard: a
// project that sets a bool true itself must continue to floor local overrides,
// independent of the new compliance floor.
func TestResolveConfig_ProjectBoolFloorStillEnforced(t *testing.T) {
	org := DefaultQsdevConfig()
	project := &types.QsdevConfig{
		// No client / compliance level here — the project's own true floor
		// must stand on its own.
		Security: types.SecurityConfig{ScriptBlocking: boolP(true)},
	}
	local := &LocalConfig{
		Security: types.SecurityConfig{ScriptBlocking: boolP(false)},
	}
	result, err := ResolveConfig(org, nil, project, local, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Config.Security.ScriptBlocking == nil || !*result.Config.Security.ScriptBlocking {
		t.Error("expected script_blocking enforced to true by project floor")
	}
	if !hasViolation(result.Violations, "security.script_blocking") {
		t.Errorf("expected script_blocking floor violation, got %v", result.Violations)
	}
}

// TestResolveConfig_NonComplianceBoolUnaffected verifies a project with no
// compliance level (and no project-level bool floor) does not gain a compliance
// bool floor: a local override that disables a setting stays disabled and
// records no violation.
func TestResolveConfig_NonComplianceBoolUnaffected(t *testing.T) {
	org := &types.QsdevConfig{
		Security: types.SecurityConfig{ScriptBlocking: boolP(true)},
	}
	// Project declares neither a security level nor a client compliance level,
	// and does not require script_blocking itself.
	project := &types.QsdevConfig{}
	local := &LocalConfig{
		Security: types.SecurityConfig{ScriptBlocking: boolP(false)},
	}
	result, err := ResolveConfig(org, nil, project, local, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Config.Security.ScriptBlocking == nil || *result.Config.Security.ScriptBlocking {
		t.Errorf("expected script_blocking to remain false (no floor), got %v", result.Config.Security.ScriptBlocking)
	}
	if hasViolation(result.Violations, "security.script_blocking") {
		t.Errorf("did not expect a script_blocking violation, got %v", result.Violations)
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
