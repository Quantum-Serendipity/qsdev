package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestParseQsdevConfig_FullConfig(t *testing.T) {
	yaml := `
version: 1
qsdev_version: ">= 0.15.0"
profile: go-service
languages:
  - name: go
    version: "1.22"
  - name: javascript
    version: "20"
    package_manager: pnpm
services:
  - name: postgres
    version: "16"
    options:
      port: "5433"
  - name: redis
security:
  level: enhanced
  age_gating: true
  script_blocking: false
  lock_enforcement: true
  vuln_scanning: true
tools:
  enabled:
    - version-sentinel
    - postmortem
  disabled:
    - semble
  config:
    version-sentinel:
      hours: "24"
claude_code:
  enabled: true
  permission_level: standard
  skills:
    - code-review
  mcp_servers:
    - context7
infrastructure:
  registry_proxy: https://registry.example.com
  nix_cache: https://nix-cache.example.com
  build_cache: https://build-cache.example.com
client:
  name: acme-corp
  compliance:
    - soc2
    - hipaa
  security_level: strict
  registry_proxy: https://acme-registry.example.com
  nix_cache: https://acme-nix.example.com
  allowed_mcp_servers:
    - context7
  blocked_mcp_servers:
    - github
  data_classification: confidential
`
	cfg, err := ParseQsdevConfigBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.Version != 1 {
		t.Errorf("Version = %d, want 1", cfg.Version)
	}
	if cfg.QsdevVersion != ">= 0.15.0" {
		t.Errorf("QsdevVersion = %q, want %q", cfg.QsdevVersion, ">= 0.15.0")
	}
	if cfg.Profile != "go-service" {
		t.Errorf("Profile = %q, want %q", cfg.Profile, "go-service")
	}
	if len(cfg.Languages) != 2 {
		t.Fatalf("Languages count = %d, want 2", len(cfg.Languages))
	}
	if cfg.Languages[0].Name != "go" || cfg.Languages[0].Version != "1.22" {
		t.Errorf("Languages[0] = %+v", cfg.Languages[0])
	}
	if cfg.Languages[1].PackageManager != "pnpm" {
		t.Errorf("Languages[1].PackageManager = %q, want pnpm", cfg.Languages[1].PackageManager)
	}
	if len(cfg.Services) != 2 {
		t.Fatalf("Services count = %d, want 2", len(cfg.Services))
	}
	if cfg.Services[0].Options["port"] != "5433" {
		t.Errorf("Services[0].Options[port] = %q, want 5433", cfg.Services[0].Options["port"])
	}
	if cfg.Security.Level != "enhanced" {
		t.Errorf("Security.Level = %q, want enhanced", cfg.Security.Level)
	}
	if cfg.Security.AgeGating == nil || !*cfg.Security.AgeGating {
		t.Error("Security.AgeGating should be true")
	}
	if cfg.Security.ScriptBlocking == nil || *cfg.Security.ScriptBlocking {
		t.Error("Security.ScriptBlocking should be false")
	}
	if len(cfg.Tools.Enabled) != 2 {
		t.Errorf("Tools.Enabled count = %d, want 2", len(cfg.Tools.Enabled))
	}
	if cfg.ClaudeCode.Enabled == nil || !*cfg.ClaudeCode.Enabled {
		t.Error("ClaudeCode.Enabled should be true")
	}
	if cfg.ClaudeCode.PermissionLevel != "standard" {
		t.Errorf("ClaudeCode.PermissionLevel = %q, want standard", cfg.ClaudeCode.PermissionLevel)
	}
	if cfg.Infrastructure.RegistryProxy != "https://registry.example.com" {
		t.Errorf("Infrastructure.RegistryProxy = %q", cfg.Infrastructure.RegistryProxy)
	}
	if cfg.Client == nil {
		t.Fatal("Client should not be nil")
	}
	if cfg.Client.Name != "acme-corp" {
		t.Errorf("Client.Name = %q, want acme-corp", cfg.Client.Name)
	}
	if len(cfg.Client.Compliance) != 2 {
		t.Errorf("Client.Compliance count = %d, want 2", len(cfg.Client.Compliance))
	}
	if cfg.Client.DataClassification != "confidential" {
		t.Errorf("Client.DataClassification = %q, want confidential", cfg.Client.DataClassification)
	}
}

func TestParseQsdevConfig_MinimalConfig(t *testing.T) {
	yaml := `version: 1`

	cfg, err := ParseQsdevConfigBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("Version = %d, want 1", cfg.Version)
	}
	if len(cfg.Languages) != 0 {
		t.Errorf("Languages should be empty, got %d", len(cfg.Languages))
	}
	if cfg.Client != nil {
		t.Error("Client should be nil for minimal config")
	}
}

func TestParseQsdevConfig_MissingVersion(t *testing.T) {
	yaml := `languages:
  - name: go`

	_, err := ParseQsdevConfigBytes([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for missing version")
	}
	if !strings.Contains(err.Error(), "missing required field") {
		t.Errorf("error = %q, want mention of missing field", err.Error())
	}
}

func TestParseQsdevConfig_VersionTooHigh(t *testing.T) {
	yaml := `version: 999`

	_, err := ParseQsdevConfigBytes([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for high version")
	}
	if !strings.Contains(err.Error(), "update qsdev") {
		t.Errorf("error = %q, want self-update suggestion", err.Error())
	}
}

func TestParseQsdevConfig_UnknownFieldsIgnored(t *testing.T) {
	yaml := `
version: 1
future_field: some_value
languages:
  - name: go
    future_lang_field: ignored
`
	cfg, err := ParseQsdevConfigBytes([]byte(yaml))
	if err != nil {
		t.Fatalf("unknown fields should be silently ignored, got: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("Version = %d, want 1", cfg.Version)
	}
	if len(cfg.Languages) != 1 || cfg.Languages[0].Name != "go" {
		t.Errorf("Languages = %+v, want [{Name:go}]", cfg.Languages)
	}
}

func TestParseQsdevConfig_InvalidYAML(t *testing.T) {
	yaml := `{invalid: yaml: [broken`

	_, err := ParseQsdevConfigBytes([]byte(yaml))
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "invalid YAML") {
		t.Errorf("error = %q, want invalid YAML message", err.Error())
	}
}

func TestParseQsdevConfig_FileNotFound(t *testing.T) {
	_, err := ParseQsdevConfig("/nonexistent/path/.qsdev.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
	if !strings.Contains(err.Error(), "reading config file") {
		t.Errorf("error = %q, want file reading error", err.Error())
	}
}

func TestParseQsdevConfig_FromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".qsdev.yaml")
	if err := os.WriteFile(path, []byte("version: 1\nlanguages:\n  - name: go\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := ParseQsdevConfig(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("Version = %d, want 1", cfg.Version)
	}
	if len(cfg.Languages) != 1 || cfg.Languages[0].Name != "go" {
		t.Errorf("Languages = %+v", cfg.Languages)
	}
}

func TestValidateQsdevConfig_InvalidSecurityLevel(t *testing.T) {
	cfg := &types.QsdevConfig{
		Version: 1,
		Security: types.SecurityConfig{
			Level: "maximum",
		},
	}

	errs := ValidateQsdevConfig(cfg, ValidateOptions{})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].Field != "security.level" {
		t.Errorf("Field = %q, want security.level", errs[0].Field)
	}
	if errs[0].Value != "maximum" {
		t.Errorf("Value = %q, want maximum", errs[0].Value)
	}
}

func TestValidateQsdevConfig_InvalidLanguageName(t *testing.T) {
	cfg := &types.QsdevConfig{
		Version: 1,
		Languages: []types.LanguageConfig{
			{Name: "go"},
			{Name: "cobol"},
		},
	}

	errs := ValidateQsdevConfig(cfg, ValidateOptions{})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0].Message, "unknown language") {
		t.Errorf("Message = %q, want unknown language message", errs[0].Message)
	}
}

func TestValidateQsdevConfig_InvalidServiceName(t *testing.T) {
	cfg := &types.QsdevConfig{
		Version: 1,
		Services: []types.ServiceConfig{
			{Name: "postgres"},
			{Name: "couchdb"},
		},
	}

	errs := ValidateQsdevConfig(cfg, ValidateOptions{})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if !strings.Contains(errs[0].Message, "unknown service") {
		t.Errorf("Message = %q, want unknown service message", errs[0].Message)
	}
}

func TestValidateQsdevConfig_InvalidToolName(t *testing.T) {
	cfg := &types.QsdevConfig{
		Version: 1,
		Tools: types.ToolsConfig{
			Enabled:  []string{"version-sentinel", "nonexistent-tool"},
			Disabled: []string{"also-fake"},
		},
	}

	errs := ValidateQsdevConfig(cfg, ValidateOptions{
		ToolNames: []string{"version-sentinel", "postmortem", "semble"},
	})
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d: %v", len(errs), errs)
	}
}

func TestValidateQsdevConfig_PointerBoolDistinction(t *testing.T) {
	// Test that nil (omitted) and explicit false are distinguishable.
	yamlNil := `
version: 1
security: {}
`
	yamlFalse := `
version: 1
security:
  age_gating: false
`
	cfgNil, err := ParseQsdevConfigBytes([]byte(yamlNil))
	if err != nil {
		t.Fatal(err)
	}
	cfgFalse, err := ParseQsdevConfigBytes([]byte(yamlFalse))
	if err != nil {
		t.Fatal(err)
	}

	if cfgNil.Security.AgeGating != nil {
		t.Error("nil config: AgeGating should be nil (omitted)")
	}
	if cfgFalse.Security.AgeGating == nil {
		t.Fatal("false config: AgeGating should not be nil")
	}
	if *cfgFalse.Security.AgeGating {
		t.Error("false config: AgeGating should be false")
	}
}

func TestValidateQsdevConfig_ClientPresentVsAbsent(t *testing.T) {
	yamlWithClient := `
version: 1
client:
  name: acme
`
	yamlWithout := `version: 1`

	cfgWith, err := ParseQsdevConfigBytes([]byte(yamlWithClient))
	if err != nil {
		t.Fatal(err)
	}
	cfgWithout, err := ParseQsdevConfigBytes([]byte(yamlWithout))
	if err != nil {
		t.Fatal(err)
	}

	if cfgWith.Client == nil {
		t.Error("Client should not be nil when present")
	}
	if cfgWithout.Client != nil {
		t.Error("Client should be nil when absent")
	}
}

func TestValidateQsdevConfig_MultipleErrors(t *testing.T) {
	cfg := &types.QsdevConfig{
		Version: 1,
		Languages: []types.LanguageConfig{
			{Name: "cobol"},
		},
		Services: []types.ServiceConfig{
			{Name: "couchdb"},
		},
		Security: types.SecurityConfig{
			Level: "maximum",
		},
		ClaudeCode: types.ClaudeCodeConfig{
			PermissionLevel: "root",
		},
	}

	errs := ValidateQsdevConfig(cfg, ValidateOptions{})
	if len(errs) < 4 {
		t.Errorf("expected at least 4 errors, got %d: %v", len(errs), errs)
	}
}

func TestValidateQsdevConfig_ClientMissingName(t *testing.T) {
	cfg := &types.QsdevConfig{
		Version: 1,
		Client: &types.ClientConfig{
			SecurityLevel:      "invalid",
			DataClassification: "invalid",
		},
	}

	errs := ValidateQsdevConfig(cfg, ValidateOptions{})
	// Should have 3 errors: missing name, invalid security level, invalid data classification.
	if len(errs) != 3 {
		t.Errorf("expected 3 errors, got %d: %v", len(errs), errs)
	}
}

func TestValidateQsdevConfig_ValidConfig(t *testing.T) {
	cfg := &types.QsdevConfig{
		Version: 1,
		Languages: []types.LanguageConfig{
			{Name: "go", Version: "1.22"},
			{Name: "javascript"},
		},
		Services: []types.ServiceConfig{
			{Name: "postgres"},
		},
		Security: types.SecurityConfig{
			Level: "enhanced",
		},
		ClaudeCode: types.ClaudeCodeConfig{
			PermissionLevel: "standard",
		},
	}

	errs := ValidateQsdevConfig(cfg, ValidateOptions{})
	if len(errs) != 0 {
		t.Errorf("expected no errors, got: %v", errs)
	}
}

func TestDefaultQsdevConfig(t *testing.T) {
	cfg := DefaultQsdevConfig()

	if cfg.Version != types.ConfigVersionCurrent {
		t.Errorf("Version = %d, want %d", cfg.Version, types.ConfigVersionCurrent)
	}
	if cfg.Security.Level != "enhanced" {
		t.Errorf("Security.Level = %q, want enhanced", cfg.Security.Level)
	}
	if cfg.Security.AgeGating == nil || !*cfg.Security.AgeGating {
		t.Error("Security.AgeGating should be true")
	}
	if cfg.Security.ScriptBlocking == nil || !*cfg.Security.ScriptBlocking {
		t.Error("Security.ScriptBlocking should be true")
	}
	if cfg.Security.LockEnforce == nil || !*cfg.Security.LockEnforce {
		t.Error("Security.LockEnforce should be true")
	}
	if cfg.Security.VulnScanning == nil || !*cfg.Security.VulnScanning {
		t.Error("Security.VulnScanning should be true")
	}
	if cfg.ClaudeCode.Enabled == nil || !*cfg.ClaudeCode.Enabled {
		t.Error("ClaudeCode.Enabled should be true")
	}
	if cfg.ClaudeCode.PermissionLevel != "standard" {
		t.Errorf("ClaudeCode.PermissionLevel = %q, want standard", cfg.ClaudeCode.PermissionLevel)
	}
}

func TestValidationError_Error(t *testing.T) {
	// With value.
	e := ValidationError{Field: "security.level", Value: "maximum", Message: "invalid"}
	got := e.Error()
	if !strings.Contains(got, "security.level") || !strings.Contains(got, "maximum") {
		t.Errorf("Error() = %q, want field and value", got)
	}

	// Without value.
	e2 := ValidationError{Field: "client.name", Message: "required"}
	got2 := e2.Error()
	if !strings.Contains(got2, "client.name") || !strings.Contains(got2, "required") {
		t.Errorf("Error() = %q, want field and message", got2)
	}
}

func TestValidateQsdevConfig_ProfileValidation(t *testing.T) {
	cfg := &types.QsdevConfig{
		Version: 1,
		Profile: "nonexistent-profile",
	}

	errs := ValidateQsdevConfig(cfg, ValidateOptions{
		ProfileNames: []string{"go-service", "web-app"},
	})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].Field != "profile" {
		t.Errorf("Field = %q, want profile", errs[0].Field)
	}
}

func TestValidateQsdevConfig_QsdevVersionValidation(t *testing.T) {
	cfg := &types.QsdevConfig{
		Version:     1,
		QsdevVersion: "not a valid constraint !!!",
	}

	errs := ValidateQsdevConfig(cfg, ValidateOptions{})
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].Field != "qsdev_version" {
		t.Errorf("Field = %q, want qsdev_version", errs[0].Field)
	}
}
