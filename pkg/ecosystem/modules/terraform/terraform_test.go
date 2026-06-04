package terraform_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/terraform"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*terraform.Module)(nil)

func newModule() *terraform.Module {
	return &terraform.Module{}
}

// --- Detection tests ---

func TestDetect_TfFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "null_resource" "x" {}`), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true for directory with .tf files")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("expected ConfidenceCertain, got %v", result.Confidence)
	}
	if !containsEvidence(result.Evidence, "*.tf files found") {
		t.Errorf("expected evidence about .tf files, got %v", result.Evidence)
	}
	if result.SuggestedConfig.Extras["variant"] != "terraform" {
		t.Errorf("expected variant=terraform, got %q", result.SuggestedConfig.Extras["variant"])
	}
}

func TestDetect_TfJsonFiles(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.tf.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true for directory with .tf.json files")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("expected ConfidenceCertain, got %v", result.Confidence)
	}
	if !containsEvidence(result.Evidence, "*.tf.json files found") {
		t.Errorf("expected evidence about .tf.json files, got %v", result.Evidence)
	}
}

func TestDetect_OpenTofu(t *testing.T) {
	dir := t.TempDir()
	// Create .opentofu/ directory.
	if err := os.MkdirAll(filepath.Join(dir, ".opentofu"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a .tf file to trigger detection.
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`resource "null_resource" "x" {}`), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true for OpenTofu project")
	}
	if result.SuggestedConfig.Extras["variant"] != "opentofu" {
		t.Errorf("expected variant=opentofu, got %q", result.SuggestedConfig.Extras["variant"])
	}
}

func TestDetect_LockfileOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".terraform.lock.hcl"), []byte("# lockfile"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when only lockfile is present")
	}
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("expected ConfidenceProbable for lockfile-only, got %v", result.Confidence)
	}
	if !containsEvidence(result.Evidence, ".terraform.lock.hcl found") {
		t.Errorf("expected evidence about lockfile, got %v", result.Evidence)
	}
}

func TestDetect_Empty(t *testing.T) {
	dir := t.TempDir()

	m := newModule()
	result := m.Detect(dir)

	if result.Detected {
		t.Error("expected Detected=false for empty directory")
	}
	if result.Confidence != ecosystem.ConfidenceAbsent {
		t.Errorf("expected ConfidenceAbsent, got %v", result.Confidence)
	}
	// Extras map should still be initialized.
	if result.SuggestedConfig.Extras == nil {
		t.Error("expected Extras map to be initialized, got nil")
	}
}

// --- DevenvNix tests ---

func TestDevenvNix_Terraform(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"variant": "terraform"},
	}
	fragment, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(fragment, "languages.terraform") {
		t.Errorf("expected fragment to contain languages.terraform, got:\n%s", fragment)
	}
	if strings.Contains(fragment, "languages.opentofu") {
		t.Errorf("expected fragment not to contain languages.opentofu, got:\n%s", fragment)
	}
}

func TestDevenvNix_OpenTofu(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"variant": "opentofu"},
	}
	fragment, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(fragment, "languages.opentofu") {
		t.Errorf("expected fragment to contain languages.opentofu, got:\n%s", fragment)
	}
	if strings.Contains(fragment, "languages.terraform") {
		t.Errorf("expected fragment not to contain languages.terraform, got:\n%s", fragment)
	}
}

func TestDevenvNix_WithVersion(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Version: "1.8.0",
		Extras:  map[string]string{"variant": "terraform"},
	}
	fragment, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(fragment, `version = "1.8.0"`) {
		t.Errorf("expected fragment to contain version = \"1.8.0\", got:\n%s", fragment)
	}
}

func TestDevenvNix_DefaultVariant(t *testing.T) {
	m := newModule()
	// No variant set, should default to terraform.
	config := ecosystem.ModuleConfig{}
	fragment, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(fragment, "languages.terraform") {
		t.Errorf("expected default variant to produce languages.terraform, got:\n%s", fragment)
	}
}

func TestDevenvNix_NoVersion(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"variant": "terraform"},
	}
	fragment, err := m.DevenvNixFragment(config)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(fragment, "version") {
		t.Errorf("expected no version line when Version is empty, got:\n%s", fragment)
	}
}

// --- SecurityConfigs tests ---

func TestSecurityConfigs_Basic(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{}
	files := m.SecurityConfigs(config)

	if len(files) != 1 {
		t.Fatalf("expected 1 generated file, got %d", len(files))
	}

	f := files[0]
	if f.Path != ".terraformrc" {
		t.Errorf("expected path .terraformrc, got %q", f.Path)
	}

	content := string(f.Content)
	if !strings.Contains(content, "disable_checkpoint = true") {
		t.Error("expected disable_checkpoint = true in content")
	}
	if !strings.Contains(content, "Terraform >= 0.13") {
		t.Error("expected version requirement comment for Terraform >= 0.13")
	}
	if strings.Contains(content, "provider_installation {") {
		t.Error("expected no provider_installation block without registry_mirror")
	}
}

func TestSecurityConfigs_WithMirror(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{
			"registry_mirror": "https://mirror.example.com/providers/",
		},
	}
	files := m.SecurityConfigs(config)

	if len(files) != 1 {
		t.Fatalf("expected 1 generated file, got %d", len(files))
	}

	content := string(files[0].Content)
	if !strings.Contains(content, "disable_checkpoint = true") {
		t.Error("expected disable_checkpoint = true in content")
	}
	if !strings.Contains(content, "provider_installation") {
		t.Error("expected provider_installation block with registry_mirror set")
	}
	if !strings.Contains(content, "network_mirror") {
		t.Error("expected network_mirror block")
	}
	if !strings.Contains(content, "https://mirror.example.com/providers/") {
		t.Error("expected mirror URL in content")
	}
	if !strings.Contains(content, `exclude = ["registry.terraform.io/*/*"]`) {
		t.Error("expected direct exclude for registry.terraform.io")
	}
}

// --- PreCommitHooks tests ---

func TestPreCommitHooks(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"variant": "terraform"},
	}
	hooks := m.PreCommitHooks(config)

	if len(hooks) != 4 {
		t.Fatalf("expected 4 hooks, got %d", len(hooks))
	}

	expectedIDs := []string{"terraform_fmt", "terraform_validate", "tflint", "tfsec"}
	for i, id := range expectedIDs {
		if hooks[i].ID != id {
			t.Errorf("hook[%d]: expected ID %q, got %q", i, id, hooks[i].ID)
		}
	}

	// terraform_fmt and terraform_validate should be BuiltIn.
	if !hooks[0].BuiltIn {
		t.Error("terraform_fmt should be BuiltIn")
	}
	if !hooks[1].BuiltIn {
		t.Error("terraform_validate should be BuiltIn")
	}
	// tflint and tfsec should NOT be BuiltIn.
	if hooks[2].BuiltIn {
		t.Error("tflint should not be BuiltIn")
	}
	if hooks[3].BuiltIn {
		t.Error("tfsec should not be BuiltIn")
	}

	// Default variant should use "terraform" in entry.
	if !strings.Contains(hooks[0].Entry, "terraform") {
		t.Errorf("expected terraform in fmt entry, got %q", hooks[0].Entry)
	}
	if !strings.Contains(hooks[1].Entry, "terraform") {
		t.Errorf("expected terraform in validate entry, got %q", hooks[1].Entry)
	}
}

func TestPreCommitHooks_OpenTofu(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"variant": "opentofu"},
	}
	hooks := m.PreCommitHooks(config)

	if len(hooks) != 4 {
		t.Fatalf("expected 4 hooks, got %d", len(hooks))
	}

	// OpenTofu variant should use "tofu" in entry for fmt and validate.
	if !strings.Contains(hooks[0].Entry, "tofu") {
		t.Errorf("expected tofu in fmt entry for opentofu variant, got %q", hooks[0].Entry)
	}
	if !strings.Contains(hooks[1].Entry, "tofu") {
		t.Errorf("expected tofu in validate entry for opentofu variant, got %q", hooks[1].Entry)
	}
}

// --- DenyRules tests ---

func TestDenyRules_Terraform(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"variant": "terraform"},
	}
	rules := m.DenyRules(config)

	if len(rules) != 3 {
		t.Fatalf("expected 3 deny rules for terraform, got %d", len(rules))
	}

	for _, rule := range rules {
		if !strings.Contains(rule, "terraform") {
			t.Errorf("expected terraform in deny rule, got %q", rule)
		}
	}
}

func TestDenyRules_OpenTofu(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"variant": "opentofu"},
	}
	rules := m.DenyRules(config)

	if len(rules) != 6 {
		t.Fatalf("expected 6 deny rules for opentofu, got %d", len(rules))
	}

	hasTerraformInit := false
	hasTerraformApply := false
	hasTerraformProviders := false
	hasTofuInit := false
	hasTofuApply := false
	hasTofuProviders := false

	for _, rule := range rules {
		switch {
		case strings.Contains(rule, "terraform init"):
			hasTerraformInit = true
		case strings.Contains(rule, "terraform apply"):
			hasTerraformApply = true
		case strings.Contains(rule, "terraform providers"):
			hasTerraformProviders = true
		case strings.Contains(rule, "tofu init"):
			hasTofuInit = true
		case strings.Contains(rule, "tofu apply"):
			hasTofuApply = true
		case strings.Contains(rule, "tofu providers"):
			hasTofuProviders = true
		}
	}

	if !hasTerraformInit {
		t.Error("expected deny rule for terraform init")
	}
	if !hasTerraformApply {
		t.Error("expected deny rule for terraform apply")
	}
	if !hasTerraformProviders {
		t.Error("expected deny rule for terraform providers")
	}
	if !hasTofuInit {
		t.Error("expected deny rule for tofu init")
	}
	if !hasTofuApply {
		t.Error("expected deny rule for tofu apply")
	}
	if !hasTofuProviders {
		t.Error("expected deny rule for tofu providers")
	}
}

// --- CICommands tests ---

func TestCICommands_Terraform(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"variant": "terraform"},
	}
	cmds := m.CICommands(config)

	if len(cmds) != 5 {
		t.Fatalf("expected 5 CI commands, got %d", len(cmds))
	}

	// First three commands should use the terraform binary.
	for i := 0; i < 3; i++ {
		if !strings.Contains(cmds[i].Command, "terraform") {
			t.Errorf("cmd[%d]: expected terraform in command, got %q", i, cmds[i].Command)
		}
	}

	// Verify phases.
	if cmds[0].Phase != ecosystem.CIPhaseInstall {
		t.Errorf("init command should be Install phase, got %v", cmds[0].Phase)
	}
	if cmds[1].Phase != ecosystem.CIPhaseTest {
		t.Errorf("validate command should be Test phase, got %v", cmds[1].Phase)
	}
	if cmds[2].Phase != ecosystem.CIPhaseTest {
		t.Errorf("plan command should be Test phase, got %v", cmds[2].Phase)
	}
	if cmds[3].Phase != ecosystem.CIPhaseScan {
		t.Errorf("tflint command should be Scan phase, got %v", cmds[3].Phase)
	}
	if cmds[4].Phase != ecosystem.CIPhaseScan {
		t.Errorf("tfsec command should be Scan phase, got %v", cmds[4].Phase)
	}

	// init should use -backend=false.
	if !strings.Contains(cmds[0].Command, "-backend=false") {
		t.Errorf("init command should contain -backend=false, got %q", cmds[0].Command)
	}
}

func TestCICommands_OpenTofu(t *testing.T) {
	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"variant": "opentofu"},
	}
	cmds := m.CICommands(config)

	if len(cmds) != 5 {
		t.Fatalf("expected 5 CI commands, got %d", len(cmds))
	}

	// First three commands should use the tofu binary.
	for i := 0; i < 3; i++ {
		if !strings.Contains(cmds[i].Command, "tofu") {
			t.Errorf("cmd[%d]: expected tofu in command for opentofu variant, got %q", i, cmds[i].Command)
		}
	}
}

// --- PackageManagers tests ---

func TestPackageManagers(t *testing.T) {
	m := newModule()
	pms := m.PackageManagers()

	if len(pms) != 1 {
		t.Fatalf("expected 1 package manager, got %d", len(pms))
	}

	pm := pms[0]
	if pm.Name != "terraform-registry" {
		t.Errorf("expected name terraform-registry, got %q", pm.Name)
	}
	if pm.LockFile != ".terraform.lock.hcl" {
		t.Errorf("expected lockfile .terraform.lock.hcl, got %q", pm.LockFile)
	}
	if pm.FrozenInstallCommand != "terraform init -lockfile=readonly" {
		t.Errorf("expected frozen install command terraform init -lockfile=readonly, got %q", pm.FrozenInstallCommand)
	}
	if pm.AgeGatingSupport {
		t.Error("expected AgeGatingSupport=false")
	}
}

// --- WizardFields tests ---

func TestWizardFields(t *testing.T) {
	m := newModule()
	fields := m.WizardFields()

	if len(fields) != 2 {
		t.Fatalf("expected 2 wizard fields, got %d", len(fields))
	}

	// First field should be variant select.
	if fields[0].Type != ecosystem.FieldTypeSelect {
		t.Errorf("expected first field to be Select, got %v", fields[0].Type)
	}
	if len(fields[0].Options) != 2 {
		t.Errorf("expected 2 options for variant field, got %d", len(fields[0].Options))
	}

	// Second field should be version input.
	if fields[1].Type != ecosystem.FieldTypeInput {
		t.Errorf("expected second field to be Input, got %v", fields[1].Type)
	}
}

// --- Metadata tests ---

func TestName(t *testing.T) {
	m := newModule()
	if got := m.Name(); got != "terraform" {
		t.Errorf("Name() = %q, want %q", got, "terraform")
	}
}

func TestDisplayName(t *testing.T) {
	m := newModule()
	if got := m.DisplayName(); got != "Terraform/OpenTofu" {
		t.Errorf("DisplayName() = %q, want %q", got, "Terraform/OpenTofu")
	}
}

func TestTier(t *testing.T) {
	m := newModule()
	if got := m.Tier(); got != 1 {
		t.Errorf("Tier() = %d, want 1", got)
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
