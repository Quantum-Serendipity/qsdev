package sectools_test

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/sectools"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestGenerateSyftYaml_Structure(t *testing.T) {
	t.Parallel()
	got, err := sectools.GenerateSyftYaml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateSyftYaml() error: %v", err)
	}
	if got.Path != ".syft.yaml" {
		t.Errorf("Path = %q, want %q", got.Path, ".syft.yaml")
	}
	if got.Mode != 0o644 {
		t.Errorf("Mode = %o, want %o", got.Mode, 0o644)
	}
	if got.Strategy != types.Overwrite {
		t.Errorf("Strategy = %v, want Overwrite", got.Strategy)
	}
	if got.Owner != "container-security" {
		t.Errorf("Owner = %q, want %q", got.Owner, "container-security")
	}
}

func TestGenerateSyftYaml_ValidYAML(t *testing.T) {
	t.Parallel()
	got, err := sectools.GenerateSyftYaml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateSyftYaml() error: %v", err)
	}
	var parsed map[string]any
	if err := yaml.Unmarshal(got.Content, &parsed); err != nil {
		t.Errorf("generated .syft.yaml is not valid YAML: %v", err)
	}
}

func TestGenerateSyftYaml_Content(t *testing.T) {
	t.Parallel()
	got, err := sectools.GenerateSyftYaml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateSyftYaml() error: %v", err)
	}
	content := string(got.Content)
	for _, want := range []string{"spdx-json", "catalogers", "all-layers"} {
		if !strings.Contains(content, want) {
			t.Errorf("content missing %q", want)
		}
	}
}
