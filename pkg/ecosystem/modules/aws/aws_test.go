package aws_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
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
var _ ecosystem.EnvKeeper = (*aws.Module)(nil)

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

	if len(rules) != 42 {
		t.Fatalf("expected 42 deny rules, got %d: %v", len(rules), rules)
	}

	denied := []string{
		"aws configure set aws_secret_access_key x",
		"aws sts get-session-token",
		"aws sts assume-role --role-arn x",
		"aws sts get-federation-token --name n",
		"aws configure export-credentials",
		"cat ~/.aws/credentials",
		"cat ~/.aws/config",
	}
	for _, cmd := range denied {
		if _, ok := denyutil.FirstMatch(rules, "Bash("+cmd+")"); !ok {
			t.Errorf("no deny rule blocks %q, got %v", cmd, rules)
		}
	}
}

// TestDenyRules_GetSessionTokenArgless is a regression guard for F-CAP-11.1-1:
// the arg-less `aws sts get-session-token` prints temporary credentials to
// stdout and must be denied, not only the form that carries trailing arguments.
// Benign, unrelated AWS commands must remain allowed (no over-matching).
func TestDenyRules_GetSessionTokenArgless(t *testing.T) {
	t.Parallel()

	rules := newModule().DenyRules(ecosystem.ModuleConfig{})

	mustDeny := []string{
		"aws sts get-session-token",
		"aws sts get-session-token --duration-seconds 900",
		"aws sts get-session-token --serial-number arn:aws:iam::123:mfa/u --token-code 123456",
	}
	for _, op := range mustDeny {
		if _, ok := denyutil.FirstMatch(rules, "Bash("+op+")"); !ok {
			t.Errorf("expected %q to be denied by AWS deny rules %v", op, rules)
		}
	}

	mustAllow := []string{
		"aws s3 ls",
		"aws sts get-caller-identity",
	}
	for _, op := range mustAllow {
		if rule, ok := denyutil.FirstMatch(rules, "Bash("+op+")"); ok {
			t.Errorf("expected %q to be allowed (not denied), but AWS deny rule %q matches", op, rule)
		}
	}
}

// --- ReadDenyRules tests ---

func TestReadDenyRules_AllPresent(t *testing.T) {
	t.Parallel()

	m := newModule()
	paths := m.ReadDenyRules(ecosystem.ModuleConfig{})

	if len(paths) != 4 {
		t.Fatalf("expected 4 read deny paths, got %d: %v", len(paths), paths)
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

// TestDevenvNix_AWSEnvOnlyWhenConfigured verifies AWS_PROFILE, AWS_REGION and
// AWS_DEFAULT_REGION are exported only with real configured values. devenv env
// overrides the user's shell, so a placeholder would clobber a working profile.
// A configured region sets both region variables, because the JS v3 and Go v2
// SDKs read AWS_REGION while the CLI also honors AWS_DEFAULT_REGION.
func TestDevenvNix_AWSEnvOnlyWhenConfigured(t *testing.T) {
	t.Parallel()

	const inherited = "inherited from your shell through devenv.yaml clean.keep, or set in devenv.local.nix"
	tests := []struct {
		name    string
		extras  map[string]string
		want    []string
		notWant []string
	}{
		{
			name:    "unconfigured exports nothing",
			want:    []string{"# AWS_PROFILE, AWS_REGION, AWS_DEFAULT_REGION: not set here; " + inherited},
			notWant: []string{"env.AWS_PROFILE", "env.AWS_REGION", "env.AWS_DEFAULT_REGION", "PLACEHOLDER"},
		},
		{
			name:    "profile only",
			extras:  map[string]string{"aws_profile": "dev-sso"},
			want:    []string{`env.AWS_PROFILE = "dev-sso";`, "# AWS_REGION, AWS_DEFAULT_REGION: not set here; " + inherited},
			notWant: []string{"env.AWS_REGION", "env.AWS_DEFAULT_REGION", "PLACEHOLDER"},
		},
		{
			name:   "region only",
			extras: map[string]string{"aws_default_region": "eu-west-1"},
			want: []string{
				`env.AWS_REGION = "eu-west-1";`, `env.AWS_DEFAULT_REGION = "eu-west-1";`,
				"# AWS_PROFILE: not set here; " + inherited,
			},
			notWant: []string{"env.AWS_PROFILE", "PLACEHOLDER"},
		},
		{
			name:   "both configured",
			extras: map[string]string{"aws_profile": "dev-sso", "aws_default_region": "eu-west-1"},
			want: []string{
				`env.AWS_PROFILE = "dev-sso";`,
				`env.AWS_REGION = "eu-west-1";`, `env.AWS_DEFAULT_REGION = "eu-west-1";`,
			},
			notWant: []string{"not set here", "PLACEHOLDER"},
		},
		{
			name:    "value is Nix-escaped",
			extras:  map[string]string{"aws_profile": `a"${b}`},
			want:    []string{`env.AWS_PROFILE = "a\"\${b}";`},
			notWant: []string{"PLACEHOLDER"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			fragment, err := newModule().DevenvNixFragment(ecosystem.ModuleConfig{Extras: tt.extras})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			for _, w := range tt.want {
				if !strings.Contains(fragment, w) {
					t.Errorf("fragment missing %q, got:\n%s", w, fragment)
				}
			}
			for _, nw := range tt.notWant {
				if strings.Contains(fragment, nw) {
					t.Errorf("fragment must not contain %q, got:\n%s", nw, fragment)
				}
			}
		})
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

	fields := newModule().WizardFields()
	want := []struct {
		key  string
		typ  ecosystem.WizardFieldType
		desc []string
	}{
		{"aws_default_region", ecosystem.FieldTypeInput, []string{"inherit AWS_REGION and AWS_DEFAULT_REGION from your shell"}},
		// A profile entered here is committed for every teammate, so the
		// field must say so and point personal profiles elsewhere.
		{"aws_profile", ecosystem.FieldTypeInput, []string{"inherit AWS_PROFILE from your shell", "whole team", "devenv.local.nix"}},
		{"aws_vault", ecosystem.FieldTypeConfirm, []string{"aws-vault"}},
	}
	if len(fields) != len(want) {
		t.Fatalf("expected %d wizard fields, got %d", len(want), len(fields))
	}
	for i, w := range want {
		f := fields[i]
		if f.Key != w.key || f.Type != w.typ {
			t.Errorf("field %d = %q (%v), want %q (%v)", i, f.Key, f.Type, w.key, w.typ)
		}
		for _, d := range w.desc {
			if !strings.Contains(f.Description, d) {
				t.Errorf("field %q description %q does not mention %q", f.Key, f.Description, d)
			}
		}
		// An empty input means "inherit from the shell", so inputs have no default.
		if w.typ == ecosystem.FieldTypeInput && f.Default != "" {
			t.Errorf("field %q default = %q, want empty", f.Key, f.Default)
		}
	}
	if fields[0].Placeholder != "us-east-1" {
		t.Errorf("region placeholder = %q, want us-east-1", fields[0].Placeholder)
	}
}

// TestKeepEnvVars lists the selector variables the AWS module passes through
// devenv.yaml clean.keep: the profile and both region variables the CLI and
// SDKs read.
func TestKeepEnvVars(t *testing.T) {
	t.Parallel()

	got := newModule().KeepEnvVars()
	want := []string{"AWS_PROFILE", "AWS_REGION", "AWS_DEFAULT_REGION"}
	if !slices.Equal(got, want) {
		t.Errorf("KeepEnvVars() = %v, want %v", got, want)
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

func TestModuleIdentity(t *testing.T) {
	t.Parallel()
	ecosystem.AssertModuleIdentity(t, newModule(), "aws", "AWS CLI", 2)
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
