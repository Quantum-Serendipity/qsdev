package config

import (
	"reflect"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestConfigToAnswers_LanguageMapping(t *testing.T) {
	cfg := &types.QsdevConfig{
		Languages: []types.LanguageConfig{
			{Name: "go", Version: "1.22"},
			{Name: "javascript", Version: "20", PackageManager: "pnpm"},
		},
	}

	answers := ConfigToAnswers(cfg, types.DetectedProject{}, "/tmp/myproject")

	if len(answers.Languages) != 2 {
		t.Fatalf("expected 2 languages, got %d", len(answers.Languages))
	}
	if answers.Languages[0].Name != "go" || answers.Languages[0].Version != "1.22" {
		t.Errorf("unexpected go language: %+v", answers.Languages[0])
	}
	if answers.Languages[1].PackageManager != "pnpm" {
		t.Errorf("expected pnpm package manager, got %q", answers.Languages[1].PackageManager)
	}
}

func TestConfigToAnswers_ServiceMapping(t *testing.T) {
	cfg := &types.QsdevConfig{
		Services: []types.ServiceConfig{
			{Name: "postgres", Version: "16", Options: map[string]string{"port": "5433"}},
		},
	}

	answers := ConfigToAnswers(cfg, types.DetectedProject{}, "/tmp/myproject")

	if len(answers.Services) != 1 {
		t.Fatalf("expected 1 service, got %d", len(answers.Services))
	}
	if answers.Services[0].Name != "postgres" || answers.Services[0].Version != "16" {
		t.Errorf("unexpected service: %+v", answers.Services[0])
	}
	if answers.Services[0].Settings["port"] != "5433" {
		t.Errorf("expected port=5433, got %v", answers.Services[0].Settings)
	}
}

func TestConfigToAnswers_ClaudeCodeFields(t *testing.T) {
	enabled := true
	cfg := &types.QsdevConfig{
		ClaudeCode: types.ClaudeCodeConfig{
			Enabled:         &enabled,
			PermissionLevel: "standard",
			Skills:          []string{"deploy", "security-review"},
			MCPServers:      []string{"context7", "github"},
		},
	}

	answers := ConfigToAnswers(cfg, types.DetectedProject{}, "/tmp/myproject")

	if !answers.ClaudeCode {
		t.Error("expected ClaudeCode to be true")
	}
	if answers.PermissionLevel != "standard" {
		t.Errorf("expected standard, got %q", answers.PermissionLevel)
	}
	if len(answers.Skills) != 2 {
		t.Errorf("expected 2 skills, got %d", len(answers.Skills))
	}
	if len(answers.MCPServers) != 2 {
		t.Errorf("expected 2 MCP servers, got %d", len(answers.MCPServers))
	}
}

func TestConfigToAnswers_ToolEnablement(t *testing.T) {
	cfg := &types.QsdevConfig{
		Tools: types.ToolsConfig{
			Enabled:  []string{"gitleaks", "semgrep"},
			Disabled: []string{"changelog"},
		},
	}

	answers := ConfigToAnswers(cfg, types.DetectedProject{}, "/tmp/myproject")

	if answers.EnabledTools == nil {
		t.Fatal("expected EnabledTools to be non-nil")
	}
	if !answers.EnabledTools["gitleaks"] {
		t.Error("expected gitleaks to be enabled")
	}
	if !answers.EnabledTools["semgrep"] {
		t.Error("expected semgrep to be enabled")
	}
	if answers.EnabledTools["changelog"] != false {
		t.Error("expected changelog to be disabled (false)")
	}
}

func TestConfigToAnswers_ComplianceLevelPropagated(t *testing.T) {
	cfg := &types.QsdevConfig{
		Security: types.SecurityConfig{Level: "strict"},
	}

	answers := ConfigToAnswers(cfg, types.DetectedProject{}, "/tmp/myproject")

	if answers.HookTier != "strict" {
		t.Errorf("expected hook tier strict, got %q", answers.HookTier)
	}
}

func TestConfigToAnswers_DetectedProjectSet(t *testing.T) {
	detected := types.DetectedProject{
		HasGoMod:       true,
		HasPackageJSON: true,
	}
	cfg := &types.QsdevConfig{}

	answers := ConfigToAnswers(cfg, detected, "/tmp/myproject")

	if !answers.Detected.HasGoMod {
		t.Error("expected HasGoMod to be true")
	}
	if !answers.Detected.HasPackageJSON {
		t.Error("expected HasPackageJSON to be true")
	}
}

func TestConfigToAnswers_ProjectRootAndName(t *testing.T) {
	cfg := &types.QsdevConfig{}

	answers := ConfigToAnswers(cfg, types.DetectedProject{}, "/home/user/projects/myapp")

	if answers.ProjectRoot != "/home/user/projects/myapp" {
		t.Errorf("expected /home/user/projects/myapp, got %q", answers.ProjectRoot)
	}
	if answers.ProjectName != "myapp" {
		t.Errorf("expected myapp, got %q", answers.ProjectName)
	}
	if !answers.Confirmed {
		t.Error("expected Confirmed to be true")
	}
	if !answers.Direnv {
		t.Error("expected Direnv to be true")
	}
}

// TestConfigToAnswers_JoinSemantics locks the semantics join mode relies on
// now that ConfigToAnswers is the single config-to-answers converter.
func TestConfigToAnswers_JoinSemantics(t *testing.T) {
	t.Parallel()
	enabled, disabled := true, false
	infra := types.InfraConfig{RegistryProxy: "https://proxy.example.com"}

	tests := []struct {
		name  string
		cfg   types.QsdevConfig
		check func(t *testing.T, a types.WizardAnswers)
	}{
		{"absent enabled means legacy default on", types.QsdevConfig{}, func(t *testing.T, a types.WizardAnswers) {
			if !a.ClaudeCode || a.PermissionLevel != "standard" {
				t.Errorf("ClaudeCode=%v PermissionLevel=%q, want true/standard", a.ClaudeCode, a.PermissionLevel)
			}
		}},
		{"explicit tier supplies the permission preset", types.QsdevConfig{Tier: "supply-chain-only"}, func(t *testing.T, a types.WizardAnswers) {
			if a.PermissionLevel != "" || a.Tier != "supply-chain-only" {
				t.Errorf("PermissionLevel=%q Tier=%q, want empty/supply-chain-only", a.PermissionLevel, a.Tier)
			}
		}},
		{"legacy config with default MCP servers infers the default tier", types.QsdevConfig{
			ClaudeCode: types.ClaudeCodeConfig{MCPServers: []string{"context7", "github", "socket", "semble"}},
		}, func(t *testing.T, a types.WizardAnswers) {
			if a.Tier != "standard" {
				t.Errorf("Tier = %q, want standard (default MCP servers never imply full)", a.Tier)
			}
		}},
		{"explicit false is honoured", types.QsdevConfig{ClaudeCode: types.ClaudeCodeConfig{Enabled: &disabled}}, func(t *testing.T, a types.WizardAnswers) {
			if a.ClaudeCode || a.PermissionLevel != "" || a.Hooks.SelfProtection {
				t.Errorf("ClaudeCode=%v PermissionLevel=%q SelfProtection=%v, want all off", a.ClaudeCode, a.PermissionLevel, a.Hooks.SelfProtection)
			}
		}},
		{"self-protection and safety-block on with Claude", types.QsdevConfig{ClaudeCode: types.ClaudeCodeConfig{Enabled: &enabled}}, func(t *testing.T, a types.WizardAnswers) {
			if !a.Hooks.SelfProtection || !a.Hooks.SafetyBlock {
				t.Errorf("hooks = %+v", a.Hooks)
			}
		}},
		{"security level is the compliance level", types.QsdevConfig{Security: types.SecurityConfig{Level: "enhanced"}}, func(t *testing.T, a types.WizardAnswers) {
			if a.ComplianceLevel != "enhanced" {
				t.Errorf("ComplianceLevel = %q", a.ComplianceLevel)
			}
		}},
		{"client can raise but not lower the level", types.QsdevConfig{
			Security: types.SecurityConfig{Level: "strict"},
			Client:   &types.ClientConfig{Name: "acme", SecurityLevel: "baseline"},
		}, func(t *testing.T, a types.WizardAnswers) {
			if a.ComplianceLevel != "strict" || !a.Hooks.AuditLog {
				t.Errorf("ComplianceLevel=%q AuditLog=%v, want strict/true", a.ComplianceLevel, a.Hooks.AuditLog)
			}
		}},
		{"profile is the project-type profile", types.QsdevConfig{Profile: "go-web"}, func(t *testing.T, a types.WizardAnswers) {
			if a.ProjectTypeProfile != "go-web" || a.ProfileName != "" {
				t.Errorf("ProjectTypeProfile=%q ProfileName=%q", a.ProjectTypeProfile, a.ProfileName)
			}
		}},
		{"infra_profile is the infrastructure profile", types.QsdevConfig{Profile: "go-web", InfraProfile: "enterprise", Infrastructure: infra}, func(t *testing.T, a types.WizardAnswers) {
			if a.ProfileName != "enterprise" || a.ProjectTypeProfile != "go-web" || a.Infrastructure.RegistryProxy != infra.RegistryProxy {
				t.Errorf("ProfileName=%q ProjectTypeProfile=%q Infrastructure=%+v", a.ProfileName, a.ProjectTypeProfile, a.Infrastructure)
			}
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.check(t, ConfigToAnswers(&tt.cfg, types.DetectedProject{}, "/tmp/p"))
		})
	}
}

// TestBranchPattern_RoundTrip checks that git.branch_pattern survives the
// config -> answers -> config round trip join and init perform.
func TestBranchPattern_RoundTrip(t *testing.T) {
	t.Parallel()
	for _, pattern := range []string{"", `^(feat|fix)/[a-z0-9._-]+$`} {
		t.Run(pattern, func(t *testing.T) {
			t.Parallel()
			cfg := &types.QsdevConfig{Version: types.ConfigVersionCurrent, Git: types.GitConfig{BranchPattern: pattern}}
			answers := ConfigToAnswers(cfg, types.DetectedProject{}, "/tmp/myproject")
			if answers.BranchPattern != pattern {
				t.Fatalf("ConfigToAnswers BranchPattern = %q, want %q", answers.BranchPattern, pattern)
			}
			if got := AnswersToConfig(answers, "").Git.BranchPattern; got != pattern {
				t.Errorf("AnswersToConfig git.branch_pattern = %q, want %q", got, pattern)
			}
		})
	}
}

// TestConfigToAnswers_HookPolicyRoundTrip covers the hooks block surviving
// join (ConfigToAnswers) and re-creation (AnswersToConfig) as a copy.
func TestConfigToAnswers_HookPolicyRoundTrip(t *testing.T) {
	t.Parallel()
	cfg := &types.QsdevConfig{Hooks: types.HooksConfig{
		FileBoundary: types.FileBoundaryConfig{ExtraReadPaths: []string{"/opt/sdk", "~/.m2/repository"}},
		ToolGates:    types.ToolGatesConfig{Allowed: []string{"Read"}, Denied: []string{"WebFetch"}},
	}}
	answers := ConfigToAnswers(cfg, types.DetectedProject{}, "/tmp/myproject")
	if !slices.Equal(answers.HookPolicy.FileBoundary.ExtraReadPaths, cfg.Hooks.FileBoundary.ExtraReadPaths) {
		t.Fatalf("HookPolicy = %+v, want %+v", answers.HookPolicy, cfg.Hooks)
	}
	if !reflect.DeepEqual(answers.HookPolicy.ToolGates, cfg.Hooks.ToolGates) {
		t.Fatalf("HookPolicy.ToolGates = %+v, want %+v", answers.HookPolicy.ToolGates, cfg.Hooks.ToolGates)
	}
	answers.HookPolicy.FileBoundary.ExtraReadPaths[0] = "/changed"
	answers.HookPolicy.ToolGates.Denied[0] = "Bash"
	if cfg.Hooks.FileBoundary.ExtraReadPaths[0] != "/opt/sdk" || cfg.Hooks.ToolGates.Denied[0] != "WebFetch" {
		t.Error("ConfigToAnswers aliased the config's hook policy lists")
	}
	back := AnswersToConfig(ConfigToAnswers(cfg, types.DetectedProject{}, "/tmp/myproject"), "")
	if !reflect.DeepEqual(back.Hooks, cfg.Hooks) {
		t.Errorf("AnswersToConfig hooks = %+v, want %+v", back.Hooks, cfg.Hooks)
	}
}
