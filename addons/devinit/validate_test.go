package devinit_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devinit"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestValidateAnswers_Valid(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go", Version: "1.24"},
			{Name: "javascript", PackageManager: "pnpm"},
			{Name: "python", PackageManager: "uv"},
		},
		Services: []types.ServiceChoice{
			{Name: "postgres"},
			{Name: "redis"},
		},
		PermissionLevel: "standard",
		EnvVars: map[string]string{
			"FOO": "bar",
		},
	}

	if err := devinit.ExportValidateAnswers(answers); err != nil {
		t.Errorf("expected no error for valid answers, got: %v", err)
	}
}

func TestValidateAnswers_UnknownLanguage(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go"},
			{Name: "cobol"},
		},
	}

	err := devinit.ExportValidateAnswers(answers)
	if err == nil {
		t.Fatal("expected error for unknown language")
	}
	if !strings.Contains(err.Error(), "cobol") {
		t.Errorf("error should mention 'cobol', got: %v", err)
	}
}

func TestValidateAnswers_UnknownService(t *testing.T) {
	answers := types.WizardAnswers{
		Services: []types.ServiceChoice{
			{Name: "postgres"},
			{Name: "cassandra"},
		},
	}

	err := devinit.ExportValidateAnswers(answers)
	if err == nil {
		t.Fatal("expected error for unknown service")
	}
	if !strings.Contains(err.Error(), "cassandra") {
		t.Errorf("error should mention 'cassandra', got: %v", err)
	}
}

func TestValidateAnswers_InvalidPermissionPreset(t *testing.T) {
	answers := types.WizardAnswers{
		PermissionLevel: "superadmin",
	}

	err := devinit.ExportValidateAnswers(answers)
	if err == nil {
		t.Fatal("expected error for invalid permission preset")
	}
	if !strings.Contains(err.Error(), "superadmin") {
		t.Errorf("error should mention 'superadmin', got: %v", err)
	}
}

func TestValidateAnswers_InvalidNodePkgMgr(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "javascript", PackageManager: "bower"},
		},
	}

	err := devinit.ExportValidateAnswers(answers)
	if err == nil {
		t.Fatal("expected error for invalid node package manager")
	}
	if !strings.Contains(err.Error(), "bower") {
		t.Errorf("error should mention 'bower', got: %v", err)
	}
}

func TestValidateAnswers_InvalidPythonPkgMgr(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "python", PackageManager: "conda"},
		},
	}

	err := devinit.ExportValidateAnswers(answers)
	if err == nil {
		t.Fatal("expected error for invalid python package manager")
	}
	if !strings.Contains(err.Error(), "conda") {
		t.Errorf("error should mention 'conda', got: %v", err)
	}
}

func TestValidateAnswers_EmptyPermissionLevelIsValid(t *testing.T) {
	answers := types.WizardAnswers{
		PermissionLevel: "",
	}

	if err := devinit.ExportValidateAnswers(answers); err != nil {
		t.Errorf("expected no error for empty permission level, got: %v", err)
	}
}

func TestValidateAnswers_MultipleErrors(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "fortran"},
		},
		Services: []types.ServiceChoice{
			{Name: "memcached"},
		},
		PermissionLevel: "root",
	}

	err := devinit.ExportValidateAnswers(answers)
	if err == nil {
		t.Fatal("expected errors for multiple invalid values")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "fortran") {
		t.Error("error should mention 'fortran'")
	}
	if !strings.Contains(errStr, "memcached") {
		t.Error("error should mention 'memcached'")
	}
	if !strings.Contains(errStr, "root") {
		t.Error("error should mention 'root'")
	}
}

func TestValidateAnswers_EmptyEnvVarKey(t *testing.T) {
	answers := types.WizardAnswers{
		EnvVars: map[string]string{
			"":    "value",
			"FOO": "bar",
		},
	}

	err := devinit.ExportValidateAnswers(answers)
	if err == nil {
		t.Fatal("expected error for empty env var key")
	}
	if !strings.Contains(err.Error(), "empty key") {
		t.Errorf("error should mention empty key, got: %v", err)
	}
}

func TestValidateAnswers_ValidPermissionPresets(t *testing.T) {
	for _, preset := range []string{"minimal", "standard", "permissive", "custom"} {
		t.Run(preset, func(t *testing.T) {
			answers := types.WizardAnswers{
				PermissionLevel: preset,
			}
			if err := devinit.ExportValidateAnswers(answers); err != nil {
				t.Errorf("expected %q to be valid, got: %v", preset, err)
			}
		})
	}
}

func TestValidateAnswers_AllValidLanguages(t *testing.T) {
	for _, lang := range []string{"go", "javascript", "python", "rust", "java", "dotnet", "container", "terraform"} {
		t.Run(lang, func(t *testing.T) {
			answers := types.WizardAnswers{
				Languages: []types.LanguageChoice{{Name: lang}},
			}
			if err := devinit.ExportValidateAnswers(answers); err != nil {
				t.Errorf("expected %q to be valid, got: %v", lang, err)
			}
		})
	}
}

func TestValidateAnswers_InvalidTier(t *testing.T) {
	answers := types.WizardAnswers{
		Tier: "nonexistent",
	}

	err := devinit.ExportValidateAnswers(answers)
	if err == nil {
		t.Fatal("expected error for invalid tier")
	}
	if !strings.Contains(err.Error(), "nonexistent") {
		t.Errorf("error should mention 'nonexistent', got: %v", err)
	}
}

func TestValidateAnswers_EmptyTierIsValid(t *testing.T) {
	answers := types.WizardAnswers{
		Tier: "",
	}

	if err := devinit.ExportValidateAnswers(answers); err != nil {
		t.Errorf("expected no error for empty tier, got: %v", err)
	}
}

func TestValidateAnswers_AllValidTiers(t *testing.T) {
	for _, tier := range []string{"supply-chain-only", "standard", "full"} {
		t.Run(tier, func(t *testing.T) {
			answers := types.WizardAnswers{
				Tier: tier,
			}
			if err := devinit.ExportValidateAnswers(answers); err != nil {
				t.Errorf("expected %q to be valid, got: %v", tier, err)
			}
		})
	}
}

func TestValidateAnswers_TierInMultipleErrors(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "fortran"},
		},
		Tier: "nonexistent",
	}

	err := devinit.ExportValidateAnswers(answers)
	if err == nil {
		t.Fatal("expected errors for multiple invalid values")
	}
	errStr := err.Error()
	if !strings.Contains(errStr, "fortran") {
		t.Error("error should mention 'fortran'")
	}
	if !strings.Contains(errStr, "nonexistent") {
		t.Error("error should mention 'nonexistent'")
	}
}

func TestValidateAnswers_AllValidServices(t *testing.T) {
	for _, svc := range []string{"postgres", "redis", "mysql", "mongodb", "elasticsearch", "rabbitmq"} {
		t.Run(svc, func(t *testing.T) {
			answers := types.WizardAnswers{
				Services: []types.ServiceChoice{{Name: svc}},
			}
			if err := devinit.ExportValidateAnswers(answers); err != nil {
				t.Errorf("expected %q to be valid, got: %v", svc, err)
			}
		})
	}
}
