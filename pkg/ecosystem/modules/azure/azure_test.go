package azure_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/azure"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*azure.Module)(nil)
var _ ecosystem.DenyRuleProvider = (*azure.Module)(nil)
var _ ecosystem.ReadDenyRuleProvider = (*azure.Module)(nil)
var _ ecosystem.PackageProvider = (*azure.Module)(nil)
var _ ecosystem.DoctorCheckProvider = (*azure.Module)(nil)

func newModule() *azure.Module {
	return &azure.Module{}
}

// --- Detection tests ---

func TestDetect_TerraformProviderAzurerm(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`provider "azurerm" {}`), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true for directory with azurerm provider")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("expected ConfidenceCertain, got %v", result.Confidence)
	}
	if !containsEvidence(result.Evidence, "azurerm") {
		t.Errorf("expected evidence about azurerm provider, got %v", result.Evidence)
	}
}

func TestDetect_AzurePipelines(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "azure-pipelines.yml"), []byte("trigger:\n  - main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true for directory with azure-pipelines.yml")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("expected ConfidenceCertain, got %v", result.Confidence)
	}
	if !containsEvidence(result.Evidence, "azure-pipelines.yml") {
		t.Errorf("expected evidence about azure-pipelines.yml, got %v", result.Evidence)
	}
}

func TestDetect_BicepFiles(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.bicep"), []byte("resource rg 'Microsoft.Resources/resourceGroups@2021-04-01' = {}"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true for directory with .bicep files")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("expected ConfidenceCertain, got %v", result.Confidence)
	}
	if !containsEvidence(result.Evidence, "*.bicep") {
		t.Errorf("expected evidence about .bicep files, got %v", result.Evidence)
	}
}

func TestDetect_AzdProject(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "azure.yaml"), []byte("name: myapp\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true for directory with azure.yaml")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("expected ConfidenceCertain, got %v", result.Confidence)
	}
	if !containsEvidence(result.Evidence, "azure.yaml") {
		t.Errorf("expected evidence about azure.yaml, got %v", result.Evidence)
	}
}

func TestDetect_AzureDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".azure"), 0o755); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true for directory with .azure/ directory")
	}
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("expected ConfidenceProbable, got %v", result.Confidence)
	}
	if !containsEvidence(result.Evidence, ".azure/") {
		t.Errorf("expected evidence about .azure/ directory, got %v", result.Evidence)
	}
}

func TestDetect_NoAzureIndicators(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	m := newModule()
	result := m.Detect(dir)

	if result.Detected {
		t.Error("expected Detected=false for empty directory")
	}
	if result.Confidence != ecosystem.ConfidenceAbsent {
		t.Errorf("expected ConfidenceAbsent, got %v", result.Confidence)
	}
}

// --- DenyRules tests ---

func TestDenyRules_AllPresent(t *testing.T) {
	t.Parallel()
	m := newModule()
	rules := m.DenyRules(ecosystem.ModuleConfig{})

	if len(rules) != 4 {
		t.Fatalf("expected 4 deny rules, got %d: %v", len(rules), rules)
	}

	expected := []string{
		"az account get-access-token",
		"az ad sp credential",
		"cat ~/.azure/",
		"az login --service-principal",
	}
	for _, want := range expected {
		found := false
		for _, rule := range rules {
			if strings.Contains(rule, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected deny rule containing %q, got %v", want, rules)
		}
	}
}

// --- ReadDenyRules tests ---

func TestReadDenyRules_AllPresent(t *testing.T) {
	t.Parallel()
	m := newModule()
	paths := m.ReadDenyRules(ecosystem.ModuleConfig{})

	if len(paths) != 4 {
		t.Fatalf("expected 4 read-deny paths, got %d: %v", len(paths), paths)
	}

	expected := []string{
		"accessTokens.json",
		"msal_token_cache.json",
		"service_principal_entries.json",
		"azureProfile.json",
	}
	for _, want := range expected {
		found := false
		for _, path := range paths {
			if strings.Contains(path, want) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected read-deny path containing %q, got %v", want, paths)
		}
	}
}

// --- DevenvNixFragment tests ---

func TestDevenvNix_ContainsSubscription(t *testing.T) {
	t.Parallel()
	m := newModule()
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(fragment, "ARM_SUBSCRIPTION_ID") {
		t.Errorf("expected fragment to contain ARM_SUBSCRIPTION_ID, got:\n%s", fragment)
	}
	if !strings.Contains(fragment, "ARM_TENANT_ID") {
		t.Errorf("expected fragment to contain ARM_TENANT_ID, got:\n%s", fragment)
	}
}

func TestDevenvNix_NoClientSecret(t *testing.T) {
	t.Parallel()
	m := newModule()
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(fragment, "ARM_CLIENT_SECRET") {
		t.Errorf("expected fragment NOT to contain ARM_CLIENT_SECRET, got:\n%s", fragment)
	}
}

// --- DevenvPackages tests ---

func TestDevenvPackages_Default(t *testing.T) {
	t.Parallel()
	m := newModule()
	pkgs := m.DevenvPackages(ecosystem.ModuleConfig{})

	if len(pkgs) != 1 {
		t.Fatalf("expected 1 package, got %d: %v", len(pkgs), pkgs)
	}
	if pkgs[0] != "azure-cli" {
		t.Errorf("expected azure-cli, got %q", pkgs[0])
	}
}

func TestDevenvPackages_WithK8s(t *testing.T) {
	t.Parallel()
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"k8s": "true"},
	}
	pkgs := m.DevenvPackages(config)

	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d: %v", len(pkgs), pkgs)
	}
	if pkgs[0] != "azure-cli" {
		t.Errorf("expected azure-cli as first package, got %q", pkgs[0])
	}
	if pkgs[1] != "kubelogin" {
		t.Errorf("expected kubelogin as second package, got %q", pkgs[1])
	}
}

// --- helpers ---

func containsEvidence(evidence []string, substr string) bool {
	for _, e := range evidence {
		if strings.Contains(e, substr) {
			return true
		}
	}
	return false
}
