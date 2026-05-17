package devenv_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestGenerateSecretSpecToml_WithServices(t *testing.T) {
	answers := types.WizardAnswers{
		Services: []types.ServiceChoice{
			{Name: "postgres"},
			{Name: "redis"},
		},
	}

	gf, err := devenv.GenerateSecretSpecToml(answers, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf == nil {
		t.Fatal("expected non-nil GeneratedFile")
	}

	content := string(gf.Content)

	// Should contain postgres secrets.
	if !strings.Contains(content, "[secrets.DATABASE_URL]") {
		t.Error("expected DATABASE_URL secret block")
	}
	if !strings.Contains(content, "[secrets.POSTGRES_PASSWORD]") {
		t.Error("expected POSTGRES_PASSWORD secret block")
	}

	// Should contain redis secrets.
	if !strings.Contains(content, "[secrets.REDIS_URL]") {
		t.Error("expected REDIS_URL secret block")
	}

	// Should have providers section.
	if !strings.Contains(content, "[providers]") {
		t.Error("expected [providers] section")
	}
	if !strings.Contains(content, `postgres = "env"`) {
		t.Error("expected postgres provider")
	}
	if !strings.Contains(content, `redis = "env"`) {
		t.Error("expected redis provider")
	}

	// Auto-generate attributes for POSTGRES_PASSWORD.
	if !strings.Contains(content, "auto_generate = true") {
		t.Error("expected auto_generate = true for POSTGRES_PASSWORD")
	}
	if !strings.Contains(content, `generate_length = 32`) {
		t.Error("expected generate_length = 32")
	}
	if !strings.Contains(content, `generate_charset = "alphanumeric"`) {
		t.Error("expected generate_charset = alphanumeric")
	}

	// File metadata.
	if gf.Path != "secretspec.toml" {
		t.Errorf("expected path secretspec.toml, got %q", gf.Path)
	}
	if gf.Mode != 0o644 {
		t.Errorf("expected mode 0o644, got %04o", gf.Mode)
	}
	if gf.Strategy != types.Overwrite {
		t.Errorf("expected strategy Overwrite, got %v", gf.Strategy)
	}
	if gf.Owner != "secretspec" {
		t.Errorf("expected owner secretspec, got %q", gf.Owner)
	}
}

func TestGenerateSecretSpecToml_NoDeclarations(t *testing.T) {
	answers := types.WizardAnswers{}

	gf, err := devenv.GenerateSecretSpecToml(answers, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf != nil {
		t.Errorf("expected nil when no declarations, got %+v", gf)
	}
}

func TestGenerateSecretSpecToml_Dedup(t *testing.T) {
	// If the same secret name appears from service and ecosystem, only keep first.
	answers := types.WizardAnswers{
		Services: []types.ServiceChoice{
			{Name: "postgres"},
			{Name: "postgres"}, // duplicate
		},
	}

	gf, err := devenv.GenerateSecretSpecToml(answers, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf == nil {
		t.Fatal("expected non-nil GeneratedFile")
	}

	content := string(gf.Content)

	// Count occurrences of DATABASE_URL — should appear exactly once.
	count := strings.Count(content, "[secrets.DATABASE_URL]")
	if count != 1 {
		t.Errorf("expected 1 occurrence of DATABASE_URL block, got %d", count)
	}

	count = strings.Count(content, "[secrets.POSTGRES_PASSWORD]")
	if count != 1 {
		t.Errorf("expected 1 occurrence of POSTGRES_PASSWORD block, got %d", count)
	}
}

func TestGenerateSecretSpecToml_WithTerraformSecrets(t *testing.T) {
	reg := ecosystem.NewRegistry()

	// Create a mock that also implements SecretDeclarer.
	mod := &mockSecretDeclarerModule{
		MockModule: ecosystem.MockModule{
			NameVal:        "terraform",
			DisplayNameVal: "Terraform",
			TierVal:        1,
		},
		secrets: []ecosystem.SecretDecl{
			{
				Name:        "AWS_ACCESS_KEY_ID",
				Description: "AWS access key",
				Required:    true,
				Source:      "terraform",
			},
			{
				Name:        "AWS_SECRET_ACCESS_KEY",
				Description: "AWS secret key",
				Required:    true,
				Source:      "terraform",
			},
		},
	}
	_ = reg.Register(mod)

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "terraform"},
		},
	}

	gf, err := devenv.GenerateSecretSpecToml(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf == nil {
		t.Fatal("expected non-nil GeneratedFile")
	}

	content := string(gf.Content)

	if !strings.Contains(content, "[secrets.AWS_ACCESS_KEY_ID]") {
		t.Error("expected AWS_ACCESS_KEY_ID secret block")
	}
	if !strings.Contains(content, "[secrets.AWS_SECRET_ACCESS_KEY]") {
		t.Error("expected AWS_SECRET_ACCESS_KEY secret block")
	}
	if !strings.Contains(content, `terraform = "env"`) {
		t.Error("expected terraform provider")
	}
}

func TestGenerateSecretSpecToml_ServicesAndModulesCombined(t *testing.T) {
	reg := ecosystem.NewRegistry()

	mod := &mockSecretDeclarerModule{
		MockModule: ecosystem.MockModule{
			NameVal:        "docker",
			DisplayNameVal: "Docker",
			TierVal:        1,
		},
		secrets: []ecosystem.SecretDecl{
			{
				Name:        "DOCKER_REGISTRY_TOKEN",
				Description: "Docker registry token",
				Required:    true,
				Source:      "docker",
			},
		},
	}
	_ = reg.Register(mod)

	answers := types.WizardAnswers{
		Services: []types.ServiceChoice{
			{Name: "redis"},
		},
		Languages: []types.LanguageChoice{
			{Name: "docker"},
		},
	}

	gf, err := devenv.GenerateSecretSpecToml(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf == nil {
		t.Fatal("expected non-nil GeneratedFile")
	}

	content := string(gf.Content)

	if !strings.Contains(content, "[secrets.REDIS_URL]") {
		t.Error("expected REDIS_URL from service")
	}
	if !strings.Contains(content, "[secrets.DOCKER_REGISTRY_TOKEN]") {
		t.Error("expected DOCKER_REGISTRY_TOKEN from module")
	}
}

func TestGenerateSecretSpecToml_UnknownServiceReturnsNil(t *testing.T) {
	answers := types.WizardAnswers{
		Services: []types.ServiceChoice{
			{Name: "mongodb"}, // no secrets declared for mongodb
		},
	}

	gf, err := devenv.GenerateSecretSpecToml(answers, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf != nil {
		t.Errorf("expected nil for service with no secrets, got %+v", gf)
	}
}

// mockSecretDeclarerModule embeds MockModule and adds SecretDeclarer.
type mockSecretDeclarerModule struct {
	ecosystem.MockModule
	secrets []ecosystem.SecretDecl
}

func (m *mockSecretDeclarerModule) SecretDeclarations(_ ecosystem.ModuleConfig) []ecosystem.SecretDecl {
	return m.secrets
}
