package devinit_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devinit"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestProfileToAnswers_BasicMapping(t *testing.T) {
	p := devinit.ExportProfile{
		Description: "test",
		Languages: []devinit.ExportLanguageSpec{
			{Name: "go", Version: "1.24"},
			{Name: "javascript", PackageManager: "pnpm"},
		},
		Services:        []string{"postgres", "redis"},
		Direnv:          true,
		ClaudeCode:      true,
		PermissionLevel: "standard",
		Skills:          []string{"deploy", "security-review"},
		Hooks:           []string{"safety-block", "pre-commit"},
		ExtraPackages:   []string{"jq", "curl"},
		MCPServers:      []string{"filesystem"},
		GitHooks:        []string{"pre-push"},
		InfraProfile:    "consulting-default",
	}

	answers := devinit.ExportProfileToAnswers(p, "/tmp/project", "my-project")

	if answers.ProjectName != "my-project" {
		t.Errorf("ProjectName = %q, want %q", answers.ProjectName, "my-project")
	}
	if answers.ProjectRoot != "/tmp/project" {
		t.Errorf("ProjectRoot = %q, want %q", answers.ProjectRoot, "/tmp/project")
	}
	if !answers.Direnv {
		t.Error("Direnv should be true")
	}
	if !answers.ClaudeCode {
		t.Error("ClaudeCode should be true")
	}
	if answers.PermissionLevel != "standard" {
		t.Errorf("PermissionLevel = %q, want %q", answers.PermissionLevel, "standard")
	}
	if !answers.Confirmed {
		t.Error("Confirmed should be true")
	}
	if answers.ProfileName != "consulting-default" {
		t.Errorf("ProfileName = %q, want %q", answers.ProfileName, "consulting-default")
	}
}

func TestProfileToAnswers_LanguageMapping(t *testing.T) {
	p := devinit.ExportProfile{
		Languages: []devinit.ExportLanguageSpec{
			{Name: "go", Version: "1.24"},
			{Name: "python", Version: "3.12", PackageManager: "uv"},
		},
	}

	answers := devinit.ExportProfileToAnswers(p, "/tmp", "test")

	if len(answers.Languages) != 2 {
		t.Fatalf("Languages length = %d, want 2", len(answers.Languages))
	}

	if answers.Languages[0].Name != "go" || answers.Languages[0].Version != "1.24" {
		t.Errorf("Languages[0] = %+v, want {go 1.24}", answers.Languages[0])
	}
	if answers.Languages[1].Name != "python" || answers.Languages[1].Version != "3.12" || answers.Languages[1].PackageManager != "uv" {
		t.Errorf("Languages[1] = %+v, want {python 3.12 uv}", answers.Languages[1])
	}
}

func TestProfileToAnswers_ServiceMapping(t *testing.T) {
	p := devinit.ExportProfile{
		Services: []string{"postgres", "redis", "mongodb"},
	}

	answers := devinit.ExportProfileToAnswers(p, "/tmp", "test")

	if len(answers.Services) != 3 {
		t.Fatalf("Services length = %d, want 3", len(answers.Services))
	}

	for i, want := range []string{"postgres", "redis", "mongodb"} {
		if answers.Services[i].Name != want {
			t.Errorf("Services[%d].Name = %q, want %q", i, answers.Services[i].Name, want)
		}
	}
}

func TestProfileToAnswers_HookMapping(t *testing.T) {
	tests := []struct {
		hooks []string
		want  types.HookChoices
	}{
		{
			hooks: []string{"auto-format", "safety-block", "pre-commit", "audit-log"},
			want:  types.HookChoices{AutoFormat: true, SafetyBlock: true, PreCommit: true, AuditLog: true},
		},
		{
			hooks: []string{"safety-block"},
			want:  types.HookChoices{SafetyBlock: true},
		},
		{
			hooks: nil,
			want:  types.HookChoices{},
		},
		{
			hooks: []string{"unknown-hook"},
			want:  types.HookChoices{},
		},
	}

	for _, tt := range tests {
		p := devinit.ExportProfile{Hooks: tt.hooks}
		answers := devinit.ExportProfileToAnswers(p, "/tmp", "test")
		if answers.Hooks != tt.want {
			t.Errorf("hooks %v: got %+v, want %+v", tt.hooks, answers.Hooks, tt.want)
		}
	}
}

func TestProfileToAnswers_EmptyProfile(t *testing.T) {
	p := devinit.ExportProfile{}
	answers := devinit.ExportProfileToAnswers(p, "/tmp", "empty")

	if len(answers.Languages) != 0 {
		t.Errorf("Languages should be empty, got %v", answers.Languages)
	}
	if len(answers.Services) != 0 {
		t.Errorf("Services should be empty, got %v", answers.Services)
	}
	if answers.Direnv {
		t.Error("Direnv should be false")
	}
	if answers.Confirmed != true {
		t.Error("Confirmed should always be true for profile-derived answers")
	}
}

func TestProfileToAnswers_BuiltinGoWeb(t *testing.T) {
	answers := devinit.ExportProfileToAnswers(devinit.ExportGoWeb, "/projects/myapp", "myapp")

	if len(answers.Languages) != 1 || answers.Languages[0].Name != "go" {
		t.Errorf("Languages = %v, want [{go 1.24}]", answers.Languages)
	}
	if len(answers.Services) != 2 {
		t.Errorf("Services length = %d, want 2", len(answers.Services))
	}
	if !answers.Hooks.SafetyBlock {
		t.Error("Hooks.SafetyBlock should be true")
	}
	if !answers.Hooks.PreCommit {
		t.Error("Hooks.PreCommit should be true")
	}
	if answers.Hooks.AutoFormat {
		t.Error("Hooks.AutoFormat should be false for go-web")
	}
}

func TestMergeProfileWithFlags_NoOverrides(t *testing.T) {
	base := types.WizardAnswers{
		ProjectName:     "base-project",
		ProjectRoot:     "/base",
		Direnv:          true,
		ClaudeCode:      true,
		PermissionLevel: "standard",
		Languages: []types.LanguageChoice{
			{Name: "go", Version: "1.24"},
		},
		Services: []types.ServiceChoice{
			{Name: "postgres"},
		},
	}

	result := devinit.ExportMergeProfileWithFlags(base, types.WizardAnswers{}, nil)

	if result.ProjectName != "base-project" {
		t.Errorf("ProjectName = %q, want %q", result.ProjectName, "base-project")
	}
	if !result.Direnv {
		t.Error("Direnv should remain true")
	}
	if len(result.Languages) != 1 || result.Languages[0].Name != "go" {
		t.Errorf("Languages should remain from base, got %v", result.Languages)
	}
}

func TestMergeProfileWithFlags_LanguageReplace(t *testing.T) {
	base := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go", Version: "1.24"},
		},
	}
	overrides := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "rust"},
		},
	}
	changed := map[string]bool{"languages": true}

	result := devinit.ExportMergeProfileWithFlags(base, overrides, changed)

	if len(result.Languages) != 1 || result.Languages[0].Name != "rust" {
		t.Errorf("Languages should be replaced with rust, got %v", result.Languages)
	}
}

func TestMergeProfileWithFlags_ServiceAppend(t *testing.T) {
	base := types.WizardAnswers{
		Services: []types.ServiceChoice{
			{Name: "postgres"},
		},
	}
	overrides := types.WizardAnswers{
		Services: []types.ServiceChoice{
			{Name: "redis"},
			{Name: "postgres"}, // duplicate, should be deduplicated
		},
	}
	changed := map[string]bool{"services": true}

	result := devinit.ExportMergeProfileWithFlags(base, overrides, changed)

	if len(result.Services) != 2 {
		t.Fatalf("Services length = %d, want 2 (postgres + redis)", len(result.Services))
	}

	names := make(map[string]bool)
	for _, svc := range result.Services {
		names[svc.Name] = true
	}
	if !names["postgres"] || !names["redis"] {
		t.Errorf("Services should contain postgres and redis, got %v", result.Services)
	}
}

func TestMergeProfileWithFlags_DirenvOverride(t *testing.T) {
	base := types.WizardAnswers{
		Direnv: true,
	}
	overrides := types.WizardAnswers{
		Direnv: false,
	}
	changed := map[string]bool{"direnv": true}

	result := devinit.ExportMergeProfileWithFlags(base, overrides, changed)

	if result.Direnv {
		t.Error("Direnv should be overridden to false")
	}
}

func TestMergeProfileWithFlags_MultipleOverrides(t *testing.T) {
	base := types.WizardAnswers{
		ProjectName:     "base",
		Direnv:          true,
		ClaudeCode:      true,
		PermissionLevel: "standard",
		Skills:          []string{"deploy"},
	}
	overrides := types.WizardAnswers{
		ProjectName:     "override",
		PermissionLevel: "minimal",
		Skills:          []string{"security-review"},
	}
	changed := map[string]bool{
		"project_name":     true,
		"permission_level": true,
		"skills":           true,
	}

	result := devinit.ExportMergeProfileWithFlags(base, overrides, changed)

	if result.ProjectName != "override" {
		t.Errorf("ProjectName = %q, want %q", result.ProjectName, "override")
	}
	if result.PermissionLevel != "minimal" {
		t.Errorf("PermissionLevel = %q, want %q", result.PermissionLevel, "minimal")
	}
	if len(result.Skills) != 1 || result.Skills[0] != "security-review" {
		t.Errorf("Skills = %v, want [security-review]", result.Skills)
	}
	// Unchanged fields should retain base values.
	if !result.Direnv {
		t.Error("Direnv should remain true (not in changed)")
	}
	if !result.ClaudeCode {
		t.Error("ClaudeCode should remain true (not in changed)")
	}
}

func TestHooksFromStrings(t *testing.T) {
	hc := devinit.ExportHooksFromStrings([]string{"auto-format", "audit-log"})

	if !hc.AutoFormat {
		t.Error("AutoFormat should be true")
	}
	if hc.SafetyBlock {
		t.Error("SafetyBlock should be false")
	}
	if hc.PreCommit {
		t.Error("PreCommit should be false")
	}
	if !hc.AuditLog {
		t.Error("AuditLog should be true")
	}
}
