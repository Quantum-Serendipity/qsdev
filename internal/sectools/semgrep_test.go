package sectools_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/sectools"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"

	// Import modules so they register with the default registry.
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/docker"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/dotnet"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/golang"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/java"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/javascript"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/python"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/rust"
	_ "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem/modules/terraform"
)

func TestGenerateSemgrepYml_GoProject(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{{Name: "go"}},
	}
	f, err := sectools.GenerateSemgrepYml(answers, ecosystem.DefaultRegistry())
	if err != nil {
		t.Fatalf("GenerateSemgrepYml() error: %v", err)
	}
	if f.Path != ".semgrep.yml" {
		t.Errorf("Path = %q, want %q", f.Path, ".semgrep.yml")
	}
	if f.Mode != 0644 {
		t.Errorf("Mode = %#o, want %#o", f.Mode, 0644)
	}
	if f.Strategy != types.Overwrite {
		t.Errorf("Strategy = %v, want Overwrite", f.Strategy)
	}
	if f.Owner != "semgrep" {
		t.Errorf("Owner = %q, want %q", f.Owner, "semgrep")
	}

	content := string(f.Content)
	if !strings.Contains(content, "p/golang") {
		t.Error("content should contain p/golang rule set")
	}
	if !strings.Contains(content, "p/owasp-top-ten") {
		t.Error("content should contain p/owasp-top-ten rule set")
	}
}

func TestGenerateSemgrepYml_MultiEcosystemDedup(t *testing.T) {
	// Both Go and Python include p/owasp-top-ten; it should appear only once.
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go"},
			{Name: "python"},
		},
	}
	f, err := sectools.GenerateSemgrepYml(answers, ecosystem.DefaultRegistry())
	if err != nil {
		t.Fatalf("GenerateSemgrepYml() error: %v", err)
	}
	content := string(f.Content)

	// Count occurrences of p/owasp-top-ten.
	count := strings.Count(content, "p/owasp-top-ten")
	if count != 1 {
		t.Errorf("p/owasp-top-ten appears %d times, want exactly 1 (dedup)", count)
	}

	// Both ecosystem-specific rules should be present.
	if !strings.Contains(content, "p/golang") {
		t.Error("content should contain p/golang")
	}
	if !strings.Contains(content, "p/python") {
		t.Error("content should contain p/python")
	}
}

func TestGenerateSemgrepYml_PathExclusions(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{{Name: "go"}},
	}
	f, err := sectools.GenerateSemgrepYml(answers, ecosystem.DefaultRegistry())
	if err != nil {
		t.Fatalf("GenerateSemgrepYml() error: %v", err)
	}
	content := string(f.Content)

	for _, path := range []string{"vendor/", "node_modules/", "dist/", ".devenv/"} {
		if !strings.Contains(content, path) {
			t.Errorf("content should exclude path %q", path)
		}
	}
}

func TestGenerateSemgrepYml_NoLanguages(t *testing.T) {
	// When no languages are selected, should still produce a valid config
	// with the fallback owasp-top-ten rule set.
	answers := types.WizardAnswers{}
	f, err := sectools.GenerateSemgrepYml(answers, ecosystem.DefaultRegistry())
	if err != nil {
		t.Fatalf("GenerateSemgrepYml() error: %v", err)
	}
	content := string(f.Content)
	if !strings.Contains(content, "p/owasp-top-ten") {
		t.Error("content should contain fallback p/owasp-top-ten")
	}
}

func TestGenerateSemgrepYml_NilRegistry(t *testing.T) {
	answers := types.WizardAnswers{}
	_, err := sectools.GenerateSemgrepYml(answers, nil)
	if err == nil {
		t.Error("expected error for nil registry, got nil")
	}
}

func TestGenerateSemgrepYml_YAMLStructure(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{{Name: "javascript"}},
	}
	f, err := sectools.GenerateSemgrepYml(answers, ecosystem.DefaultRegistry())
	if err != nil {
		t.Fatalf("GenerateSemgrepYml() error: %v", err)
	}
	content := string(f.Content)

	if !strings.Contains(content, "rules:") {
		t.Error("content should contain 'rules:' key")
	}
	if !strings.Contains(content, "paths:") {
		t.Error("content should contain 'paths:' key")
	}
	if !strings.Contains(content, "exclude:") {
		t.Error("content should contain 'exclude:' key")
	}
}
