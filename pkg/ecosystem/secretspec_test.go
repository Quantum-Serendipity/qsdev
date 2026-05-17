package ecosystem_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// mockSecretDeclarer implements both EcosystemModule and SecretDeclarer.
type mockSecretDeclarer struct {
	ecosystem.MockModule
	Secrets []ecosystem.SecretDecl
}

func (m *mockSecretDeclarer) SecretDeclarations(_ ecosystem.ModuleConfig) []ecosystem.SecretDecl {
	return m.Secrets
}

func TestSecretDeclarer_InterfaceCheck(t *testing.T) {
	// Verify that mockSecretDeclarer satisfies both interfaces.
	var _ ecosystem.EcosystemModule = (*mockSecretDeclarer)(nil)
	var _ ecosystem.SecretDeclarer = (*mockSecretDeclarer)(nil)
}

func TestSecretDeclarer_CastFromModule(t *testing.T) {
	mod := &mockSecretDeclarer{
		MockModule: ecosystem.MockModule{
			NameVal:        "test",
			DisplayNameVal: "Test Module",
			TierVal:        1,
		},
		Secrets: []ecosystem.SecretDecl{
			{Name: "TEST_SECRET", Required: true, Source: "test"},
		},
	}

	// Should be castable to SecretDeclarer.
	declarer, ok := (ecosystem.EcosystemModule)(mod).(ecosystem.SecretDeclarer)
	if !ok {
		t.Fatal("expected mockSecretDeclarer to implement SecretDeclarer")
	}

	decls := declarer.SecretDeclarations(ecosystem.ModuleConfig{})
	if len(decls) != 1 {
		t.Fatalf("expected 1 declaration, got %d", len(decls))
	}
	if decls[0].Name != "TEST_SECRET" {
		t.Errorf("expected Name TEST_SECRET, got %q", decls[0].Name)
	}
}

func TestSecretDeclarer_NonImplementor(t *testing.T) {
	// A plain MockModule does NOT implement SecretDeclarer.
	mod := &ecosystem.MockModule{
		NameVal:        "plain",
		DisplayNameVal: "Plain Module",
		TierVal:        1,
	}

	_, ok := (ecosystem.EcosystemModule)(mod).(ecosystem.SecretDeclarer)
	if ok {
		t.Error("plain MockModule should not implement SecretDeclarer")
	}
}

func TestGenerateSpec_Fields(t *testing.T) {
	spec := &ecosystem.GenerateSpec{
		Length:  32,
		Charset: "alphanumeric",
	}

	if spec.Length != 32 {
		t.Errorf("expected Length 32, got %d", spec.Length)
	}
	if spec.Charset != "alphanumeric" {
		t.Errorf("expected Charset alphanumeric, got %q", spec.Charset)
	}
}

func TestSecretDecl_Fields(t *testing.T) {
	decl := ecosystem.SecretDecl{
		Name:         "DB_PASSWORD",
		Description:  "Database password",
		Required:     true,
		AutoGenerate: true,
		GenerateSpec: &ecosystem.GenerateSpec{
			Length:  32,
			Charset: "hex",
		},
		Source: "postgres",
	}

	if decl.Name != "DB_PASSWORD" {
		t.Errorf("expected Name DB_PASSWORD, got %q", decl.Name)
	}
	if !decl.Required {
		t.Error("expected Required to be true")
	}
	if !decl.AutoGenerate {
		t.Error("expected AutoGenerate to be true")
	}
	if decl.GenerateSpec.Length != 32 {
		t.Errorf("expected GenerateSpec.Length 32, got %d", decl.GenerateSpec.Length)
	}
	if decl.Source != "postgres" {
		t.Errorf("expected Source postgres, got %q", decl.Source)
	}
}
