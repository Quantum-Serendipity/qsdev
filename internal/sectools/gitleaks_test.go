package sectools_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/sectools"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestGenerateGitleaksToml_Structure(t *testing.T) {
	f, err := sectools.GenerateGitleaksToml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateGitleaksToml() error: %v", err)
	}
	if f.Path != ".gitleaks.toml" {
		t.Errorf("Path = %q, want %q", f.Path, ".gitleaks.toml")
	}
	if f.Mode != 0644 {
		t.Errorf("Mode = %#o, want %#o", f.Mode, 0644)
	}
	if f.Strategy != types.Overwrite {
		t.Errorf("Strategy = %v, want Overwrite", f.Strategy)
	}
	if f.Owner != "gitleaks" {
		t.Errorf("Owner = %q, want %q", f.Owner, "gitleaks")
	}
}

func TestGenerateGitleaksToml_TOMLContent(t *testing.T) {
	f, err := sectools.GenerateGitleaksToml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateGitleaksToml() error: %v", err)
	}
	content := string(f.Content)

	if !strings.Contains(content, "title = \"gitleaks config\"") {
		t.Error("content should contain title")
	}
	if !strings.Contains(content, "[allowlist]") {
		t.Error("content should contain [allowlist] section")
	}
	if !strings.Contains(content, "regexes = []") {
		t.Error("content should contain empty regexes list")
	}
}

func TestGenerateGitleaksToml_AllowlistPaths(t *testing.T) {
	f, err := sectools.GenerateGitleaksToml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateGitleaksToml() error: %v", err)
	}
	content := string(f.Content)

	for _, path := range []string{"vendor/", "node_modules/", ".devenv/", "docs/", "testdata/"} {
		if !strings.Contains(content, path) {
			t.Errorf("content should contain allowlisted path %q", path)
		}
	}
}
