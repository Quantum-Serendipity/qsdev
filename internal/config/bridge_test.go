package config

import (
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestConfigToAnswers_LanguageMapping(t *testing.T) {
	cfg := &types.GdevConfig{
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
	cfg := &types.GdevConfig{
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
	cfg := &types.GdevConfig{
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
	cfg := &types.GdevConfig{
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
	cfg := &types.GdevConfig{
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
	cfg := &types.GdevConfig{}

	answers := ConfigToAnswers(cfg, detected, "/tmp/myproject")

	if !answers.Detected.HasGoMod {
		t.Error("expected HasGoMod to be true")
	}
	if !answers.Detected.HasPackageJSON {
		t.Error("expected HasPackageJSON to be true")
	}
}

func TestConfigToAnswers_ProjectRootAndName(t *testing.T) {
	cfg := &types.GdevConfig{}

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
