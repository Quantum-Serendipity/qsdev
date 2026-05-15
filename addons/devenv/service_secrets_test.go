package devenv_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
)

func TestServiceSecretDeclarations_Postgres(t *testing.T) {
	decls := devenv.ServiceSecretDeclarations("postgres")
	if len(decls) != 2 {
		t.Fatalf("expected 2 declarations for postgres, got %d", len(decls))
	}

	// DATABASE_URL
	if decls[0].Name != "DATABASE_URL" {
		t.Errorf("expected first secret DATABASE_URL, got %q", decls[0].Name)
	}
	if !decls[0].Required {
		t.Error("expected DATABASE_URL to be Required")
	}
	if decls[0].AutoGenerate {
		t.Error("expected DATABASE_URL to not AutoGenerate")
	}
	if decls[0].Source != "postgres" {
		t.Errorf("expected Source postgres, got %q", decls[0].Source)
	}

	// POSTGRES_PASSWORD
	if decls[1].Name != "POSTGRES_PASSWORD" {
		t.Errorf("expected second secret POSTGRES_PASSWORD, got %q", decls[1].Name)
	}
	if !decls[1].Required {
		t.Error("expected POSTGRES_PASSWORD to be Required")
	}
	if !decls[1].AutoGenerate {
		t.Error("expected POSTGRES_PASSWORD to AutoGenerate")
	}
	if decls[1].GenerateSpec == nil {
		t.Fatal("expected POSTGRES_PASSWORD to have GenerateSpec")
	}
	if decls[1].GenerateSpec.Length != 32 {
		t.Errorf("expected GenerateSpec.Length 32, got %d", decls[1].GenerateSpec.Length)
	}
	if decls[1].GenerateSpec.Charset != "alphanumeric" {
		t.Errorf("expected GenerateSpec.Charset alphanumeric, got %q", decls[1].GenerateSpec.Charset)
	}
}

func TestServiceSecretDeclarations_Redis(t *testing.T) {
	decls := devenv.ServiceSecretDeclarations("redis")
	if len(decls) != 1 {
		t.Fatalf("expected 1 declaration for redis, got %d", len(decls))
	}
	if decls[0].Name != "REDIS_URL" {
		t.Errorf("expected REDIS_URL, got %q", decls[0].Name)
	}
	if !decls[0].Required {
		t.Error("expected REDIS_URL to be Required")
	}
	if decls[0].AutoGenerate {
		t.Error("expected REDIS_URL to not AutoGenerate")
	}
	if decls[0].Source != "redis" {
		t.Errorf("expected Source redis, got %q", decls[0].Source)
	}
}

func TestServiceSecretDeclarations_MySQL(t *testing.T) {
	decls := devenv.ServiceSecretDeclarations("mysql")
	if len(decls) != 2 {
		t.Fatalf("expected 2 declarations for mysql, got %d", len(decls))
	}

	if decls[0].Name != "MYSQL_ROOT_PASSWORD" {
		t.Errorf("expected MYSQL_ROOT_PASSWORD, got %q", decls[0].Name)
	}
	if !decls[0].AutoGenerate {
		t.Error("expected MYSQL_ROOT_PASSWORD to AutoGenerate")
	}
	if decls[0].GenerateSpec == nil {
		t.Fatal("expected MYSQL_ROOT_PASSWORD to have GenerateSpec")
	}
	if decls[0].GenerateSpec.Length != 32 {
		t.Errorf("expected GenerateSpec.Length 32, got %d", decls[0].GenerateSpec.Length)
	}

	if decls[1].Name != "MYSQL_URL" {
		t.Errorf("expected MYSQL_URL, got %q", decls[1].Name)
	}
	if !decls[1].Required {
		t.Error("expected MYSQL_URL to be Required")
	}
	if decls[1].AutoGenerate {
		t.Error("expected MYSQL_URL to not AutoGenerate")
	}
}

func TestServiceSecretDeclarations_RabbitMQ(t *testing.T) {
	decls := devenv.ServiceSecretDeclarations("rabbitmq")
	if len(decls) != 1 {
		t.Fatalf("expected 1 declaration for rabbitmq, got %d", len(decls))
	}
	if decls[0].Name != "RABBITMQ_DEFAULT_PASS" {
		t.Errorf("expected RABBITMQ_DEFAULT_PASS, got %q", decls[0].Name)
	}
	if !decls[0].AutoGenerate {
		t.Error("expected RABBITMQ_DEFAULT_PASS to AutoGenerate")
	}
	if decls[0].GenerateSpec == nil {
		t.Fatal("expected RABBITMQ_DEFAULT_PASS to have GenerateSpec")
	}
	if decls[0].GenerateSpec.Length != 32 {
		t.Errorf("expected GenerateSpec.Length 32, got %d", decls[0].GenerateSpec.Length)
	}
	if decls[0].GenerateSpec.Charset != "alphanumeric" {
		t.Errorf("expected Charset alphanumeric, got %q", decls[0].GenerateSpec.Charset)
	}
}

func TestServiceSecretDeclarations_UnknownService(t *testing.T) {
	decls := devenv.ServiceSecretDeclarations("mongodb")
	if decls != nil {
		t.Errorf("expected nil for unknown service, got %v", decls)
	}
}

func TestServiceSecretDeclarations_EmptyString(t *testing.T) {
	decls := devenv.ServiceSecretDeclarations("")
	if decls != nil {
		t.Errorf("expected nil for empty string, got %v", decls)
	}
}
