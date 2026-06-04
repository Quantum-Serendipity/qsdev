package sectools_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sectools"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestGenerateOpengrepConfigYaml_Structure(t *testing.T) {
	t.Parallel()

	f, err := sectools.GenerateOpengrepConfigYaml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateOpengrepConfigYaml() error: %v", err)
	}
	if f.Path != ".opengrep/config.yaml" {
		t.Errorf("Path = %q, want %q", f.Path, ".opengrep/config.yaml")
	}
	if f.Mode != 0o644 {
		t.Errorf("Mode = %#o, want %#o", f.Mode, 0o644)
	}
	if f.Strategy != types.Overwrite {
		t.Errorf("Strategy = %v, want Overwrite", f.Strategy)
	}
	if f.Owner != "opengrep" {
		t.Errorf("Owner = %q, want %q", f.Owner, "opengrep")
	}
}

func TestGenerateOpengrepConfigYaml_YAMLContent(t *testing.T) {
	t.Parallel()

	f, err := sectools.GenerateOpengrepConfigYaml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateOpengrepConfigYaml() error: %v", err)
	}
	content := string(f.Content)

	for _, key := range []string{"rules:", "exclude:", "severity:", "timeout:"} {
		if !strings.Contains(content, key) {
			t.Errorf("content should contain %q key", key)
		}
	}
}

func TestGenerateOpengrepConfigYaml_RulePaths(t *testing.T) {
	t.Parallel()

	f, err := sectools.GenerateOpengrepConfigYaml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateOpengrepConfigYaml() error: %v", err)
	}
	content := string(f.Content)

	if !strings.Contains(content, "rules/core") {
		t.Error("content should reference rules/core path")
	}
}

func TestGenerateOpengrepConfigYaml_PathExclusions(t *testing.T) {
	t.Parallel()

	f, err := sectools.GenerateOpengrepConfigYaml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateOpengrepConfigYaml() error: %v", err)
	}
	content := string(f.Content)

	for _, path := range []string{"vendor/", "node_modules/", "dist/", ".devenv/", "__pycache__/"} {
		if !strings.Contains(content, path) {
			t.Errorf("content should exclude path %q", path)
		}
	}
}

func TestGenerateOpengrepConfigYaml_Defaults(t *testing.T) {
	t.Parallel()

	f, err := sectools.GenerateOpengrepConfigYaml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateOpengrepConfigYaml() error: %v", err)
	}
	content := string(f.Content)

	if !strings.Contains(content, "severity: warning") {
		t.Error("default severity should be warning")
	}
	if !strings.Contains(content, "timeout: 300") {
		t.Error("default timeout should be 300")
	}
}
