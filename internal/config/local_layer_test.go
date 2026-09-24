package config

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestComparePermissionLevels(t *testing.T) {
	t.Parallel()
	tests := []struct {
		a, b           string
		want           int
		wantComparable bool
	}{
		{a: "minimal", b: "standard", want: 1, wantComparable: true},
		{a: "standard", b: "minimal", want: -1, wantComparable: true},
		{a: "permissive", b: "standard", want: -1, wantComparable: true},
		{a: "standard", b: "standard", want: 0, wantComparable: true},
		{a: "custom", b: "custom", want: 0, wantComparable: true},
		{a: "minimal", b: "custom"},
		{a: "supply-chain-only", b: "minimal"},
		{a: "nonexistent", b: "standard"},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			t.Parallel()
			got, ok := ComparePermissionLevels(tt.a, tt.b)
			if ok != tt.wantComparable || (ok && got != tt.want) {
				t.Errorf("ComparePermissionLevels(%q, %q) = %d, %v; want %d, %v", tt.a, tt.b, got, ok, tt.want, tt.wantComparable)
			}
		})
	}
}

// TestResolveConfig_LocalPermissionLevel checks a local permission level
// only ever tightens the committed (or tier-implied) one.
func TestResolveConfig_LocalPermissionLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name         string
		project      types.QsdevConfig
		local        string
		want         string
		wantViolated bool
		wantEnforced any
	}{
		{name: "tighten standard to minimal", project: withPermission("standard"), local: "minimal", want: "minimal"},
		{name: "same level", project: withPermission("standard"), local: "standard", want: "standard"},
		{
			name: "loosen standard to permissive", project: withPermission("standard"), local: "permissive",
			want: "standard", wantViolated: true, wantEnforced: "standard",
		},
		{
			// The finding's repro: a strict client implies minimal.
			name:    "client strict implies minimal",
			project: types.QsdevConfig{Client: &types.ClientConfig{Name: "acme", SecurityLevel: "strict"}},
			local:   "permissive", want: "minimal", wantViolated: true, wantEnforced: "minimal",
		},
		{
			name: "unset level uses the tier preset", project: types.QsdevConfig{Tier: "standard"},
			local: "permissive", want: "", wantViolated: true, wantEnforced: "standard",
		},
		{name: "unset level tightened", project: types.QsdevConfig{Tier: "standard"}, local: "minimal", want: "minimal"},
		{
			name: "custom is not comparable", project: withPermission("custom"), local: "minimal",
			want: "custom", wantViolated: true,
		},
		{
			name: "supply-chain-only tier is not comparable", project: types.QsdevConfig{Tier: "supply-chain-only"},
			local: "minimal", want: "", wantViolated: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			project := tt.project
			local := &LocalConfig{ClaudeCode: types.ClaudeCodeConfig{PermissionLevel: tt.local}}
			result, err := ResolveConfig(nil, nil, &project, local, false)
			if err != nil {
				t.Fatal(err)
			}
			if got := result.Config.ClaudeCode.PermissionLevel; got != tt.want {
				t.Errorf("permission level = %q, want %q", got, tt.want)
			}
			v, found := findViolation(result.Violations, "claude_code.permission_level")
			if found != tt.wantViolated {
				t.Fatalf("violation recorded = %v, want %v (%+v)", found, tt.wantViolated, result.Violations)
			}
			if found && v.Enforced != tt.wantEnforced {
				t.Errorf("violation enforced = %v, want %v", v.Enforced, tt.wantEnforced)
			}
		})
	}
}

// TestResolveConfig_LocalCannotLoosenClientProject is the finding's repro: a
// local file over a strict-client project cannot disable its required
// scanners or add a blocked MCP server, while its additions still apply.
func TestResolveConfig_LocalCannotLoosenClientProject(t *testing.T) {
	t.Parallel()
	project := &types.QsdevConfig{
		Client: &types.ClientConfig{
			Name: "acme", SecurityLevel: "strict",
			BlockedMCP: []string{types.MCPWildcard}, AllowedMCP: []string{"context7"},
		},
	}
	local := &LocalConfig{
		ClaudeCode:    types.ClaudeCodeConfig{PermissionLevel: "permissive", MCPServers: []string{"evil", "context7"}},
		Tools:         types.ToolsConfig{Disabled: []string{"gitleaks", "semgrep"}},
		ExtraPackages: []string{"neovim"},
	}
	result, err := ResolveConfig(nil, nil, project, local, false)
	if err != nil {
		t.Fatal(err)
	}
	cfg := result.Config
	for _, tool := range []string{"gitleaks", "semgrep"} {
		if slices.Contains(cfg.Tools.Disabled, tool) || !slices.Contains(cfg.Tools.Enabled, tool) {
			t.Errorf("%s: enabled %v, disabled %v; want it enabled only", tool, cfg.Tools.Enabled, cfg.Tools.Disabled)
		}
	}
	if !slices.Equal(cfg.ClaudeCode.MCPServers, []string{"context7"}) {
		t.Errorf("MCP servers = %v, want only context7", cfg.ClaudeCode.MCPServers)
	}
	if cfg.ClaudeCode.PermissionLevel != "minimal" {
		t.Errorf("permission level = %q, want minimal", cfg.ClaudeCode.PermissionLevel)
	}
	if !slices.Contains(cfg.Packages, "neovim") || !slices.Contains(result.Local.Packages, "neovim") {
		t.Errorf("extra package lost: config %v, local %v", cfg.Packages, result.Local.Packages)
	}
	a := ConfigToAnswers(cfg, types.DetectedProject{}, "")
	if !a.EnabledTools["gitleaks"] || !a.EnabledTools["semgrep"] {
		t.Errorf("answers disable required tools: %v", a.EnabledTools)
	}
	for _, field := range []string{"tools.disabled", "claude_code.permission_level"} {
		if _, ok := findViolation(result.Violations, field); !ok {
			t.Errorf("no %s violation in %+v", field, result.Violations)
		}
	}
}

// TestResolveConfig_LocalLayerAddsOnly covers the add-only merge of the other
// local fields.
func TestResolveConfig_LocalLayerAddsOnly(t *testing.T) {
	t.Parallel()
	disabled := false
	project := &types.QsdevConfig{
		Languages: []types.LanguageConfig{{Name: "go", Version: "1.22"}, {Name: "python", PackageManager: "uv"}},
		Services:  []types.ServiceConfig{{Name: "postgres", Options: map[string]string{"port": "5432"}}},
		Tools:     types.ToolsConfig{Enabled: []string{"gitleaks"}, Disabled: []string{"changelog"}},
	}
	local := &LocalConfig{
		Languages: []types.LanguageConfig{{Name: "go", Version: "1.23"}, {Name: "python", PackageManager: "pip"}, {Name: "rust"}},
		Services: []types.ServiceConfig{
			{Name: "postgres", Options: map[string]string{"port": "6000", "max_connections": "50"}},
			{Name: "redis"},
		},
		Tools: types.ToolsConfig{
			Enabled: []string{"changelog"},
			Config:  map[string]map[string]any{"semgrep": {"severity": "low"}},
		},
		ClaudeCode: types.ClaudeCodeConfig{Enabled: &disabled},
	}
	result, err := ResolveConfig(nil, nil, project, local, false)
	if err != nil {
		t.Fatal(err)
	}
	cfg := result.Config

	wantLangs := []types.LanguageConfig{{Name: "go", Version: "1.23"}, {Name: "python", PackageManager: "uv"}, {Name: "rust"}}
	if !slices.Equal(cfg.Languages, wantLangs) {
		t.Errorf("languages = %+v, want %+v", cfg.Languages, wantLangs)
	}
	if len(cfg.Services) != 2 || cfg.Services[1].Name != "redis" {
		t.Fatalf("services = %+v, want postgres and redis", cfg.Services)
	}
	if opts := cfg.Services[0].Options; opts["port"] != "5432" || opts["max_connections"] != "50" {
		t.Errorf("postgres options = %v, want the committed port and the added max_connections", opts)
	}
	if !slices.Contains(cfg.Tools.Enabled, "changelog") || slices.Contains(cfg.Tools.Disabled, "changelog") {
		t.Errorf("tools enabled %v, disabled %v; want changelog enabled only", cfg.Tools.Enabled, cfg.Tools.Disabled)
	}
	if cfg.Tools.Config != nil {
		t.Errorf("local tools.config applied: %v", cfg.Tools.Config)
	}
	if cfg.ClaudeCode.Enabled != nil {
		t.Errorf("local claude_code.enabled applied: %v", *cfg.ClaudeCode.Enabled)
	}
	for _, field := range []string{
		"languages.python.package_manager", "services.postgres.options.port", "tools.config", "claude_code.enabled",
	} {
		if _, ok := findViolation(result.Violations, field); !ok {
			t.Errorf("no %s violation in %+v", field, result.Violations)
		}
	}
	// The project's own lists are untouched.
	if project.Services[0].Options["max_connections"] != "" || len(project.Languages) != 2 {
		t.Errorf("ResolveConfig mutated the project layer: %+v", project)
	}
}

func TestProjectPolicy_ApplyLocal(t *testing.T) {
	t.Parallel()
	project := &types.QsdevConfig{
		Languages:  []types.LanguageConfig{{Name: "go"}},
		ClaudeCode: types.ClaudeCodeConfig{PermissionLevel: "standard"},
		Client:     &types.ClientConfig{Name: "acme", BlockedMCP: []string{"evil"}},
	}
	local := &LocalConfig{
		Languages:     []types.LanguageConfig{{Name: "go", Version: "1.23"}, {Name: "rust"}},
		Services:      []types.ServiceConfig{{Name: "redis", Options: map[string]string{"port": "6380"}}},
		Tools:         types.ToolsConfig{Enabled: []string{"changelog"}, Disabled: []string{"gitleaks"}},
		ClaudeCode:    types.ClaudeCodeConfig{PermissionLevel: "minimal", MCPServers: []string{"evil", "context7"}, Skills: []string{"review"}},
		ExtraPackages: []string{"neovim"},
	}
	policy, err := ResolveProjectPolicy(project, local)
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		level     string
		wantLevel string
	}{
		{name: "tightens the committed level", level: "standard", wantLevel: "minimal"},
		{name: "keeps an incomparable answer", level: "custom", wantLevel: "custom"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := types.WizardAnswers{
				ClaudeCode:      true,
				PermissionLevel: tt.level,
				Languages:       []types.LanguageChoice{{Name: "go", Version: "1.22"}},
				ExtraPackages:   []string{"jq"},
				EnabledTools:    map[string]bool{"gitleaks": true, "changelog": false},
			}
			policy.Apply(&a)
			policy.ApplyLocal(&a)

			if a.PermissionLevel != tt.wantLevel {
				t.Errorf("PermissionLevel = %q, want %q", a.PermissionLevel, tt.wantLevel)
			}
			if !slices.Equal(a.ExtraPackages, []string{"jq", "neovim"}) {
				t.Errorf("ExtraPackages = %v", a.ExtraPackages)
			}
			wantLangs := []types.LanguageChoice{{Name: "go", Version: "1.23"}, {Name: "rust"}}
			if !slices.EqualFunc(a.Languages, wantLangs, func(x, y types.LanguageChoice) bool {
				return x.Name == y.Name && x.Version == y.Version
			}) {
				t.Errorf("Languages = %+v, want %+v", a.Languages, wantLangs)
			}
			if len(a.Services) != 1 || a.Services[0].Settings["port"] != "6380" {
				t.Errorf("Services = %+v, want redis with its port", a.Services)
			}
			if !a.EnabledTools["gitleaks"] || !a.EnabledTools["changelog"] {
				t.Errorf("EnabledTools = %v, want gitleaks kept and changelog added", a.EnabledTools)
			}
			if !slices.Equal(a.MCPServers, []string{"context7"}) {
				t.Errorf("MCPServers = %v, want the blocked server filtered", a.MCPServers)
			}
			if !slices.Equal(a.Skills, []string{"review"}) {
				t.Errorf("Skills = %v", a.Skills)
			}
		})
	}
}

func TestProjectPolicy_ApplyLocalWithoutLocalFile(t *testing.T) {
	t.Parallel()
	policy, err := ResolveProjectPolicy(&types.QsdevConfig{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	a := types.WizardAnswers{ExtraPackages: []string{"jq"}}
	policy.ApplyLocal(&a)
	if !slices.Equal(a.ExtraPackages, []string{"jq"}) {
		t.Errorf("ExtraPackages = %v, want unchanged", a.ExtraPackages)
	}
}

func TestProjectPolicy_WarningsForIgnoredOverrides(t *testing.T) {
	t.Parallel()
	policy, err := ResolveProjectPolicy(
		&types.QsdevConfig{ClaudeCode: types.ClaudeCodeConfig{PermissionLevel: "standard"}},
		&LocalConfig{
			Tools:      types.ToolsConfig{Disabled: []string{"gitleaks"}},
			ClaudeCode: types.ClaudeCodeConfig{PermissionLevel: "permissive"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(policy.Warnings(), "\n")
	for _, want := range []string{
		"tools.disabled: gitleaks ignored (" + reasonLocalToolDisable + ")",
		"claude_code.permission_level: permissive raised to standard",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("warnings lack %q:\n%s", want, got)
		}
	}
}

func withPermission(level string) types.QsdevConfig {
	return types.QsdevConfig{ClaudeCode: types.ClaudeCodeConfig{PermissionLevel: level}}
}

func findViolation(vs []FloorViolation, field string) (FloorViolation, bool) {
	i := slices.IndexFunc(vs, func(v FloorViolation) bool { return v.Field == field })
	if i < 0 {
		return FloorViolation{}, false
	}
	return vs[i], true
}
