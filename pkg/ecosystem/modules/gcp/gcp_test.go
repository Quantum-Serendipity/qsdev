package gcp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/gcp"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*gcp.Module)(nil)
var _ ecosystem.DenyRuleProvider = (*gcp.Module)(nil)
var _ ecosystem.ReadDenyRuleProvider = (*gcp.Module)(nil)
var _ ecosystem.PackageProvider = (*gcp.Module)(nil)
var _ ecosystem.DoctorCheckProvider = (*gcp.Module)(nil)

func newModule() *gcp.Module {
	return &gcp.Module{}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("writing fixture %s: %v", name, err)
	}
}

func assertEvidenceContains(t *testing.T, evidence []string, want string) {
	t.Helper()
	for _, e := range evidence {
		if e == want {
			return
		}
	}
	t.Errorf("evidence %v missing %q", evidence, want)
}

// ---------- Detection ----------

func TestDetect_TerraformProviderGoogle(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "main.tf", `provider "google" {}`)

	result := newModule().Detect(dir)
	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want Certain", result.Confidence)
	}
	assertEvidenceContains(t, result.Evidence, `Terraform provider "google" found`)
}

func TestDetect_Firebase(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "firebase.json", `{"hosting": {}}`)

	result := newModule().Detect(dir)
	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want Certain", result.Confidence)
	}
	assertEvidenceContains(t, result.Evidence, "firebase.json found")
}

func TestDetect_CloudBuild(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "cloudbuild.yaml", "steps:\n  - name: gcr.io/cloud-builders/docker\n")

	result := newModule().Detect(dir)
	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want Certain", result.Confidence)
	}
	assertEvidenceContains(t, result.Evidence, "cloudbuild.yaml found")
}

func TestDetect_AppEngine(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "app.yaml", "runtime: go121\n")

	result := newModule().Detect(dir)
	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want Certain", result.Confidence)
	}
	assertEvidenceContains(t, result.Evidence, "app.yaml found")
}

func TestDetect_Gcloudignore(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, ".gcloudignore", "node_modules/\n")

	result := newModule().Detect(dir)
	if !result.Detected {
		t.Fatal("expected Detected=true")
	}
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want Probable", result.Confidence)
	}
	assertEvidenceContains(t, result.Evidence, ".gcloudignore found")
}

func TestDetect_NoGCPIndicators(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()

	result := newModule().Detect(dir)
	if result.Detected {
		t.Fatal("expected Detected=false for empty directory")
	}
	if result.Confidence != ecosystem.ConfidenceAbsent {
		t.Errorf("Confidence = %v, want Absent", result.Confidence)
	}
}

// ---------- DenyRules ----------

func TestDenyRules_AllPresent(t *testing.T) {
	t.Parallel()
	rules := newModule().DenyRules(ecosystem.ModuleConfig{})
	if len(rules) != 5 {
		t.Fatalf("expected 5 deny rules, got %d: %v", len(rules), rules)
	}

	// Verify each rule contains "gcloud" or "Bash(".
	for _, r := range rules {
		if !strings.HasPrefix(r, "Bash(") {
			t.Errorf("deny rule %q should start with Bash(", r)
		}
	}
}

// ---------- ReadDenyRules ----------

func TestReadDenyRules_AllPresent(t *testing.T) {
	t.Parallel()
	rules := newModule().ReadDenyRules(ecosystem.ModuleConfig{})
	if len(rules) != 4 {
		t.Fatalf("expected 4 read deny paths, got %d: %v", len(rules), rules)
	}
}

// ---------- DevenvNixFragment ----------

func TestDevenvNix_ContainsCloudsdk(t *testing.T) {
	t.Parallel()
	frag, err := newModule().DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment error: %v", err)
	}
	if !strings.Contains(frag, "CLOUDSDK_ACTIVE_CONFIG_NAME") {
		t.Error("fragment should contain CLOUDSDK_ACTIVE_CONFIG_NAME")
	}
	if !strings.Contains(frag, "CLOUDSDK_CORE_PROJECT") {
		t.Error("fragment should contain CLOUDSDK_CORE_PROJECT")
	}
	if !strings.Contains(frag, "GOOGLE_CLOUD_PROJECT") {
		t.Error("fragment should contain GOOGLE_CLOUD_PROJECT")
	}
}

func TestDevenvNix_NoGACEnvVar(t *testing.T) {
	t.Parallel()
	frag, err := newModule().DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment error: %v", err)
	}
	if strings.Contains(frag, "GOOGLE_APPLICATION_CREDENTIALS =") {
		t.Error("fragment should NOT contain GOOGLE_APPLICATION_CREDENTIALS =")
	}
}

func TestDevenvNix_K8sExtra(t *testing.T) {
	t.Parallel()
	cfg := ecosystem.ModuleConfig{
		Extras: map[string]string{"k8s": "true"},
	}
	frag, err := newModule().DevenvNixFragment(cfg)
	if err != nil {
		t.Fatalf("DevenvNixFragment error: %v", err)
	}
	if !strings.Contains(frag, "gke-gcloud-auth-plugin") {
		t.Error("k8s=true fragment should mention gke-gcloud-auth-plugin")
	}
}

func TestDevenvNix_NoK8s(t *testing.T) {
	t.Parallel()
	frag, err := newModule().DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("DevenvNixFragment error: %v", err)
	}
	if strings.Contains(frag, "gke-gcloud-auth-plugin") {
		t.Error("default fragment should NOT mention gke-gcloud-auth-plugin")
	}
}

// ---------- DevenvPackages ----------

func TestDevenvPackages_Default(t *testing.T) {
	t.Parallel()
	pkgs := newModule().DevenvPackages(ecosystem.ModuleConfig{})
	want := []string{"google-cloud-sdk"}
	if len(pkgs) != len(want) {
		t.Fatalf("DevenvPackages() = %v, want %v", pkgs, want)
	}
	for i, w := range want {
		if pkgs[i] != w {
			t.Errorf("DevenvPackages()[%d] = %q, want %q", i, pkgs[i], w)
		}
	}
}

// ---------- DoctorChecks ----------

func TestDoctorChecks(t *testing.T) {
	t.Parallel()
	checks := newModule().DoctorChecks(ecosystem.ModuleConfig{})
	if len(checks) != 2 {
		t.Fatalf("expected 2 doctor checks, got %d", len(checks))
	}

	authCheck := checks[0]
	if authCheck.Name != "gcp-auth" {
		t.Errorf("check[0].Name = %q, want gcp-auth", authCheck.Name)
	}
	if authCheck.Command != "gcloud auth print-access-token" {
		t.Errorf("check[0].Command = %q, want gcloud auth print-access-token", authCheck.Command)
	}
	if authCheck.Timeout != 5 {
		t.Errorf("check[0].Timeout = %d, want 5", authCheck.Timeout)
	}
	if authCheck.Provider != "gcp" {
		t.Errorf("check[0].Provider = %q, want gcp", authCheck.Provider)
	}

	configCheck := checks[1]
	if configCheck.Name != "gcp-config" {
		t.Errorf("check[1].Name = %q, want gcp-config", configCheck.Name)
	}
	if configCheck.EnvCheck != "CLOUDSDK_ACTIVE_CONFIG_NAME" {
		t.Errorf("check[1].EnvCheck = %q, want CLOUDSDK_ACTIVE_CONFIG_NAME", configCheck.EnvCheck)
	}
	if configCheck.Provider != "gcp" {
		t.Errorf("check[1].Provider = %q, want gcp", configCheck.Provider)
	}
}

// ---------- Identity ----------

func TestName(t *testing.T) {
	t.Parallel()
	if got := newModule().Name(); got != "gcp" {
		t.Errorf("Name() = %q, want gcp", got)
	}
}

func TestDisplayName(t *testing.T) {
	t.Parallel()
	if got := newModule().DisplayName(); got != "Google Cloud CLI" {
		t.Errorf("DisplayName() = %q, want %q", got, "Google Cloud CLI")
	}
}

func TestTier(t *testing.T) {
	t.Parallel()
	if got := newModule().Tier(); got != 2 {
		t.Errorf("Tier() = %d, want 2", got)
	}
}

// ---------- Nil returns ----------

func TestSecurityConfigs_Nil(t *testing.T) {
	t.Parallel()
	if got := newModule().SecurityConfigs(ecosystem.ModuleConfig{}); got != nil {
		t.Errorf("SecurityConfigs() = %v, want nil", got)
	}
}

func TestPreCommitHooks_Nil(t *testing.T) {
	t.Parallel()
	if got := newModule().PreCommitHooks(ecosystem.ModuleConfig{}); got != nil {
		t.Errorf("PreCommitHooks() = %v, want nil", got)
	}
}

func TestCICommands_Nil(t *testing.T) {
	t.Parallel()
	if got := newModule().CICommands(ecosystem.ModuleConfig{}); got != nil {
		t.Errorf("CICommands() = %v, want nil", got)
	}
}

func TestPackageManagers_Nil(t *testing.T) {
	t.Parallel()
	if got := newModule().PackageManagers(); got != nil {
		t.Errorf("PackageManagers() = %v, want nil", got)
	}
}

func TestVerificationCommands_Empty(t *testing.T) {
	t.Parallel()
	vc := newModule().VerificationCommands(ecosystem.ModuleConfig{})
	if !vc.IsEmpty() {
		t.Errorf("VerificationCommands() should be empty, got %+v", vc)
	}
}
