package sectools_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/sectools"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestGenerateScancodeYml_Structure(t *testing.T) {
	f, err := sectools.GenerateScancodeYml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateScancodeYml() error: %v", err)
	}
	if f.Path != ".scancode.yml" {
		t.Errorf("Path = %q, want %q", f.Path, ".scancode.yml")
	}
	if f.Mode != 0644 {
		t.Errorf("Mode = %#o, want %#o", f.Mode, 0644)
	}
	if f.Strategy != types.Overwrite {
		t.Errorf("Strategy = %v, want Overwrite", f.Strategy)
	}
	if f.Owner != "license-compliance" {
		t.Errorf("Owner = %q, want %q", f.Owner, "license-compliance")
	}
}

func TestGenerateScancodeYml_AllowedLicenses(t *testing.T) {
	f, err := sectools.GenerateScancodeYml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateScancodeYml() error: %v", err)
	}
	content := string(f.Content)

	for _, id := range []string{"MIT", "Apache-2.0", "BSD-2-Clause", "BSD-3-Clause", "ISC"} {
		if !strings.Contains(content, id) {
			t.Errorf("content should list allowed license %q", id)
		}
	}
}

func TestGenerateScancodeYml_BlockedLicenses(t *testing.T) {
	f, err := sectools.GenerateScancodeYml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateScancodeYml() error: %v", err)
	}
	content := string(f.Content)

	for _, id := range []string{"GPL-2.0-only", "GPL-3.0-only", "AGPL-3.0-only", "SSPL-1.0", "BUSL-1.1"} {
		if !strings.Contains(content, id) {
			t.Errorf("content should list blocked license %q", id)
		}
	}
}

func TestGenerateScancodeYml_ReviewLicenses(t *testing.T) {
	f, err := sectools.GenerateScancodeYml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateScancodeYml() error: %v", err)
	}
	content := string(f.Content)

	for _, id := range []string{"LGPL-2.1-only", "MPL-2.0", "EPL-2.0"} {
		if !strings.Contains(content, id) {
			t.Errorf("content should list review license %q", id)
		}
	}
}

func TestGenerateScancodeYml_PathExclusions(t *testing.T) {
	f, err := sectools.GenerateScancodeYml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateScancodeYml() error: %v", err)
	}
	content := string(f.Content)

	for _, path := range []string{"vendor/", "node_modules/", ".devenv/", "dist/", "testdata/"} {
		if !strings.Contains(content, path) {
			t.Errorf("content should exclude path %q", path)
		}
	}
}

func TestGenerateLicenseExceptionsYml_Structure(t *testing.T) {
	f, err := sectools.GenerateLicenseExceptionsYml()
	if err != nil {
		t.Fatalf("GenerateLicenseExceptionsYml() error: %v", err)
	}
	if f.Path != ".license-exceptions.yml" {
		t.Errorf("Path = %q, want %q", f.Path, ".license-exceptions.yml")
	}
	if f.Mode != 0644 {
		t.Errorf("Mode = %#o, want %#o", f.Mode, 0644)
	}
	if f.Strategy != types.Skip {
		t.Errorf("Strategy = %v, want Skip", f.Strategy)
	}
	if f.Owner != "license-compliance" {
		t.Errorf("Owner = %q, want %q", f.Owner, "license-compliance")
	}
}

func TestGenerateLicenseExceptionsYml_Content(t *testing.T) {
	f, err := sectools.GenerateLicenseExceptionsYml()
	if err != nil {
		t.Fatalf("GenerateLicenseExceptionsYml() error: %v", err)
	}
	content := string(f.Content)

	if !strings.Contains(content, "exceptions: []") {
		t.Error("content should contain empty exceptions list")
	}
	if !strings.Contains(content, "justification") {
		t.Error("content should contain example with justification field")
	}
}

func TestGenerateScancodeYml_Sections(t *testing.T) {
	f, err := sectools.GenerateScancodeYml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateScancodeYml() error: %v", err)
	}
	content := string(f.Content)

	if !strings.Contains(content, "allowed:") {
		t.Error("content should contain 'allowed:' section")
	}
	if !strings.Contains(content, "blocked:") {
		t.Error("content should contain 'blocked:' section")
	}
	if !strings.Contains(content, "review:") {
		t.Error("content should contain 'review:' section")
	}
	if !strings.Contains(content, "paths:") {
		t.Error("content should contain 'paths:' section")
	}
}
