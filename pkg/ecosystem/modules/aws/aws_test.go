package aws_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/aws"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*aws.Module)(nil)
var _ ecosystem.DenyRuleProvider = (*aws.Module)(nil)
var _ ecosystem.ReadDenyRuleProvider = (*aws.Module)(nil)
var _ ecosystem.PackageProvider = (*aws.Module)(nil)
var _ ecosystem.WizardFieldProvider = (*aws.Module)(nil)
var _ ecosystem.DoctorCheckProvider = (*aws.Module)(nil)

func newModule() *aws.Module {
	return &aws.Module{}
}

// --- Detection tests ---

func TestDetect_TerraformProviderAWS(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.tf"), []byte(`provider "aws" {
  region = "us-east-1"
}`), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true for directory with Terraform AWS provider")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("expected ConfidenceCertain, got %v", result.Confidence)
	}
	if !containsEvidence(result.Evidence, "Terraform provider") {
		t.Errorf("expected evidence about Terraform provider, got %v", result.Evidence)
	}
}

func TestDetect_CDKProject(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cdk.json"), []byte(`{"app": "npx ts-node bin/app.ts"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true for CDK project")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("expected ConfidenceCertain, got %v", result.Confidence)
	}
	if !containsEvidence(result.Evidence, "cdk.json") {
		t.Errorf("expected evidence about cdk.json, got %v", result.Evidence)
	}
	if result.SuggestedConfig.Extras["cdk"] != "true" {
		t.Errorf("expected extras[cdk]=true, got %q", result.SuggestedConfig.Extras["cdk"])
	}
}

func TestDetect_ServerlessFramework(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "serverless.yml"), []byte("service: my-service\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true for Serverless Framework project")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("expected ConfidenceCertain, got %v", result.Confidence)
	}
	if !containsEvidence(result.Evidence, "serverless.yml") {
		t.Errorf("expected evidence about serverless.yml, got %v", result.Evidence)
	}
}

func TestDetect_SAMTemplate(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "samconfig.toml"), []byte("[default]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true for SAM project")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("expected ConfidenceCertain, got %v", result.Confidence)
	}
	if !containsEvidence(result.Evidence, "samconfig.toml") {
		t.Errorf("expected evidence about samconfig.toml, got %v", result.Evidence)
	}
}

func TestDetect_SAMTemplateYaml(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "template.yaml"), []byte(`
AWSTemplateFormatVersion: '2010-09-09'
Resources:
  MyFunction:
    Type: AWS::Lambda::Function
`), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true for template.yaml with AWS:: resources")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("expected ConfidenceCertain, got %v", result.Confidence)
	}
	if !containsEvidence(result.Evidence, "template.yaml with AWS::") {
		t.Errorf("expected evidence about template.yaml with AWS::, got %v", result.Evidence)
	}
}

func TestDetect_CodeBuild(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "buildspec.yml"), []byte("version: 0.2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true for CodeBuild project")
	}
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("expected ConfidenceProbable for buildspec.yml alone, got %v", result.Confidence)
	}
	if !containsEvidence(result.Evidence, "buildspec.yml") {
		t.Errorf("expected evidence about buildspec.yml, got %v", result.Evidence)
	}
}

func TestDetect_NoAWSIndicators(t *testing.T) {
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
	if result.SuggestedConfig.Extras == nil {
		t.Error("expected Extras map to be initialized, got nil")
	}
}

func TestDetect_CodeDeploy(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "appspec.yml"), []byte("version: 0.0\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true for CodeDeploy project")
	}
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("expected ConfidenceProbable for appspec.yml alone, got %v", result.Confidence)
	}
}

func TestDetect_AWSSamDir(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".aws-sam"), 0o755); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true for .aws-sam/ directory")
	}
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("expected ConfidenceProbable for .aws-sam/ alone, got %v", result.Confidence)
	}
}

func TestDetect_WeakWithCertain(t *testing.T) {
	t.Parallel()

	// When both a definitive and weak indicator exist, confidence should be Certain.
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "cdk.json"), []byte(`{}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "buildspec.yml"), []byte("version: 0.2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := newModule()
	result := m.Detect(dir)

	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("expected ConfidenceCertain when both strong and weak indicators present, got %v", result.Confidence)
	}
}

// --- DenyRules tests ---

func TestDenyRules_AllPresent(t *testing.T) {
	t.Parallel()

	m := newModule()
	rules := m.DenyRules(ecosystem.ModuleConfig{})

	if len(rules) != 5 {
		t.Fatalf("expected 5 deny rules, got %d: %v", len(rules), rules)
	}

	expected := []string{
		"configure set",
		"sts get-session-token",
		"sts assume-role",
		"cat ~/.aws/credentials",
		"cat ~/.aws/config",
	}
	for _, exp := range expected {
		found := false
		for _, rule := range rules {
			if strings.Contains(rule, exp) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected deny rule containing %q, got %v", exp, rules)
		}
	}
}

// --- ReadDenyRules tests ---

func TestReadDenyRules_AllPresent(t *testing.T) {
	t.Parallel()

	m := newModule()
	paths := m.ReadDenyRules(ecosystem.ModuleConfig{})

	if len(paths) != 3 {
		t.Fatalf("expected 3 read deny paths, got %d: %v", len(paths), paths)
	}

	expected := []string{
		"credentials",
		"config",
		"sso/cache",
	}
	for _, exp := range expected {
		found := false
		for _, p := range paths {
			if strings.Contains(p, exp) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected read deny path containing %q, got %v", exp, paths)
		}
	}
}

// --- DevenvNix tests ---

func TestDevenvNix_ContainsAWSProfile(t *testing.T) {
	t.Parallel()

	m := newModule()
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(fragment, "AWS_PROFILE") {
		t.Errorf("expected fragment to contain AWS_PROFILE, got:\n%s", fragment)
	}
	if !strings.Contains(fragment, "AWS_DEFAULT_REGION") {
		t.Errorf("expected fragment to contain AWS_DEFAULT_REGION, got:\n%s", fragment)
	}
}

func TestDevenvNix_NoCredentialValues(t *testing.T) {
	t.Parallel()

	m := newModule()
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, forbidden := range []string{"ACCESS_KEY", "SECRET", "TOKEN"} {
		if strings.Contains(fragment, forbidden) {
			t.Errorf("fragment must not contain %q (credential leak risk), got:\n%s", forbidden, fragment)
		}
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
	if pkgs[0] != "awscli2" {
		t.Errorf("expected awscli2, got %q", pkgs[0])
	}
}

func TestDevenvPackages_WithVault(t *testing.T) {
	t.Parallel()

	m := newModule()
	config := ecosystem.ModuleConfig{
		Extras: map[string]string{"aws_vault": "true"},
	}
	pkgs := m.DevenvPackages(config)

	if len(pkgs) != 2 {
		t.Fatalf("expected 2 packages, got %d: %v", len(pkgs), pkgs)
	}
	if pkgs[0] != "awscli2" {
		t.Errorf("expected awscli2 as first package, got %q", pkgs[0])
	}
	if pkgs[1] != "aws-vault" {
		t.Errorf("expected aws-vault as second package, got %q", pkgs[1])
	}
}

// --- WizardFields tests ---

func TestWizardFields(t *testing.T) {
	t.Parallel()

	m := newModule()
	fields := m.WizardFields()

	if len(fields) != 2 {
		t.Fatalf("expected 2 wizard fields, got %d", len(fields))
	}

	// First field: region input.
	if fields[0].Key != "aws_default_region" {
		t.Errorf("expected first field key aws_default_region, got %q", fields[0].Key)
	}
	if fields[0].Type != ecosystem.FieldTypeInput {
		t.Errorf("expected first field to be Input, got %v", fields[0].Type)
	}
	if fields[0].Default != "us-east-1" {
		t.Errorf("expected default us-east-1, got %q", fields[0].Default)
	}

	// Second field: aws-vault confirm.
	if fields[1].Key != "aws_vault" {
		t.Errorf("expected second field key aws_vault, got %q", fields[1].Key)
	}
	if fields[1].Type != ecosystem.FieldTypeConfirm {
		t.Errorf("expected second field to be Confirm, got %v", fields[1].Type)
	}
}

// --- DoctorChecks tests ---

func TestDoctorChecks(t *testing.T) {
	t.Parallel()

	m := newModule()
	checks := m.DoctorChecks(ecosystem.ModuleConfig{})

	if len(checks) != 2 {
		t.Fatalf("expected 2 doctor checks, got %d", len(checks))
	}

	// First check: aws-auth command check.
	if checks[0].Name != "aws-auth" {
		t.Errorf("expected first check name aws-auth, got %q", checks[0].Name)
	}
	if checks[0].Command != "aws sts get-caller-identity" {
		t.Errorf("expected command 'aws sts get-caller-identity', got %q", checks[0].Command)
	}
	if checks[0].Timeout != 5 {
		t.Errorf("expected timeout 5, got %d", checks[0].Timeout)
	}
	if checks[0].Provider != "aws" {
		t.Errorf("expected provider aws, got %q", checks[0].Provider)
	}

	// Second check: AWS_PROFILE env check.
	if checks[1].Name != "aws-profile" {
		t.Errorf("expected second check name aws-profile, got %q", checks[1].Name)
	}
	if checks[1].EnvCheck != "AWS_PROFILE" {
		t.Errorf("expected env check AWS_PROFILE, got %q", checks[1].EnvCheck)
	}
}

// --- Nil/empty return tests ---

func TestSecurityConfigs_Nil(t *testing.T) {
	t.Parallel()

	m := newModule()
	if got := m.SecurityConfigs(ecosystem.ModuleConfig{}); got != nil {
		t.Errorf("expected nil SecurityConfigs, got %v", got)
	}
}

func TestPreCommitHooks_Nil(t *testing.T) {
	t.Parallel()

	m := newModule()
	if got := m.PreCommitHooks(ecosystem.ModuleConfig{}); got != nil {
		t.Errorf("expected nil PreCommitHooks, got %v", got)
	}
}

func TestCICommands_Nil(t *testing.T) {
	t.Parallel()

	m := newModule()
	if got := m.CICommands(ecosystem.ModuleConfig{}); got != nil {
		t.Errorf("expected nil CICommands, got %v", got)
	}
}

func TestPackageManagers_Nil(t *testing.T) {
	t.Parallel()

	m := newModule()
	if got := m.PackageManagers(); got != nil {
		t.Errorf("expected nil PackageManagers, got %v", got)
	}
}

func TestVerificationCommands_Empty(t *testing.T) {
	t.Parallel()

	m := newModule()
	vc := m.VerificationCommands(ecosystem.ModuleConfig{})
	if !vc.IsEmpty() {
		t.Errorf("expected empty VerificationCommands, got %+v", vc)
	}
}

// --- Metadata tests ---

func TestName(t *testing.T) {
	t.Parallel()

	m := newModule()
	if got := m.Name(); got != "aws" {
		t.Errorf("Name() = %q, want %q", got, "aws")
	}
}

func TestDisplayName(t *testing.T) {
	t.Parallel()

	m := newModule()
	if got := m.DisplayName(); got != "AWS CLI" {
		t.Errorf("DisplayName() = %q, want %q", got, "AWS CLI")
	}
}

func TestTier(t *testing.T) {
	t.Parallel()

	m := newModule()
	if got := m.Tier(); got != 2 {
		t.Errorf("Tier() = %d, want 2", got)
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
