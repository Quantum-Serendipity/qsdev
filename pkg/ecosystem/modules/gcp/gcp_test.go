package gcp_test

import (
	"os"
	"path/filepath"
	"slices"
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

// TestDetect_TerraformProviderGoogleBeta covers W136: a configuration that
// uses only the google-beta provider is detected as GCP.
func TestDetect_TerraformProviderGoogleBeta(t *testing.T) {
	t.Parallel()
	for name, content := range map[string]string{
		"provider block":  "provider \"google-beta\" {\n  project = \"p\"\n}\n",
		"required source": "terraform {\n  required_providers {\n    google-beta = { source = \"hashicorp/google-beta\" }\n  }\n}\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
			result := newModule().Detect(dir)
			if !result.Detected || result.Confidence != ecosystem.ConfidenceCertain {
				t.Errorf("Detect = (%v, %v), want detected with certain confidence", result.Detected, result.Confidence)
			}
		})
	}
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

// TestDetect_AppYamlWithoutRuntime verifies a generic app.yaml (for example
// a Kubernetes Deployment) does not enable the GCP ecosystem.
func TestDetect_AppYamlWithoutRuntime(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, dir, "app.yaml", "apiVersion: apps/v1\nkind: Deployment\nspec:\n  template:\n    spec:\n      runtime: x\n")

	if result := newModule().Detect(dir); result.Detected {
		t.Errorf("Detected = true for a non-App Engine app.yaml (evidence %v)", result.Evidence)
	}
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
	if len(rules) != 17 {
		t.Fatalf("expected 17 deny rules, got %d: %v", len(rules), rules)
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
	if len(rules) != 6 {
		t.Fatalf("expected 6 read deny paths, got %d: %v", len(rules), rules)
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

// TestDevenvNix_NoLiveEnvAssignments guards against exporting placeholder
// values: a fake CLOUDSDK_ACTIVE_CONFIG_NAME/CLOUDSDK_CORE_PROJECT breaks every
// gcloud command in the shell, and any generated env.X definition collides
// with the user's own definition from devenv.local.nix or --env.
func TestDevenvNix_NoLiveEnvAssignments(t *testing.T) {
	t.Parallel()
	for _, extras := range []map[string]string{nil, {"k8s": "true"}} {
		frag, err := newModule().DevenvNixFragment(ecosystem.ModuleConfig{Extras: extras})
		if err != nil {
			t.Fatalf("DevenvNixFragment error: %v", err)
		}
		for line := range strings.SplitSeq(strings.TrimRight(frag, "\n"), "\n") {
			if !strings.HasPrefix(line, "  #") {
				t.Errorf("fragment line is live Nix, want comment only: %q", line)
			}
		}
		if strings.Contains(frag, "PLACEHOLDER") {
			t.Errorf("fragment contains a placeholder value:\n%s", frag)
		}
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

// TestDevenvPackages_K8sProvidesAuthPlugin asserts GKE projects get the
// gke-gcloud-auth-plugin binary, which the plain google-cloud-sdk package
// lacks, and never two conflicting google-cloud-sdk builds.
func TestDevenvPackages_K8sProvidesAuthPlugin(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		extras    map[string]string
		wantPkgs  []string
		wantExprs []string
	}{
		{
			name:     "no k8s",
			wantPkgs: []string{"google-cloud-sdk"},
		},
		{
			name:   "k8s",
			extras: map[string]string{"k8s": "true"},
			wantExprs: []string{
				"(pkgs.google-cloud-sdk.withExtraComponents [ pkgs.google-cloud-sdk.components.gke-gcloud-auth-plugin ])",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := ecosystem.ModuleConfig{Extras: tt.extras}
			if got := newModule().DevenvPackages(cfg); !slices.Equal(got, tt.wantPkgs) {
				t.Errorf("DevenvPackages() = %v, want %v", got, tt.wantPkgs)
			}
			if got := newModule().DevenvPackageExprs(cfg); !slices.Equal(got, tt.wantExprs) {
				t.Errorf("DevenvPackageExprs() = %v, want %v", got, tt.wantExprs)
			}
		})
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

func TestModuleIdentity(t *testing.T) {
	t.Parallel()
	ecosystem.AssertModuleIdentity(t, newModule(), "gcp", "Google Cloud CLI", 2)
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
