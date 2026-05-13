package sectools_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/sectools"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestGenerateGrypeYaml_Structure(t *testing.T) {
	f, err := sectools.GenerateGrypeYaml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateGrypeYaml() error: %v", err)
	}
	if f.Path != ".grype.yaml" {
		t.Errorf("Path = %q, want %q", f.Path, ".grype.yaml")
	}
	if f.Mode != 0644 {
		t.Errorf("Mode = %#o, want %#o", f.Mode, 0644)
	}
	if f.Strategy != types.Overwrite {
		t.Errorf("Strategy = %v, want Overwrite", f.Strategy)
	}
	if f.Owner != "container-security" {
		t.Errorf("Owner = %q, want %q", f.Owner, "container-security")
	}
}

func TestGenerateGrypeYaml_Content(t *testing.T) {
	f, err := sectools.GenerateGrypeYaml(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateGrypeYaml() error: %v", err)
	}
	content := string(f.Content)

	if !strings.Contains(content, "fail-on-severity: high") {
		t.Error("content should specify fail-on-severity: high")
	}
	if !strings.Contains(content, "auto-update: true") {
		t.Error("content should enable auto-update")
	}
	if !strings.Contains(content, "max-allowed-built-age: 120h") {
		t.Error("content should set max-allowed-built-age to 120h")
	}
}

func TestGenerateCosignPolicy_Structure(t *testing.T) {
	f, err := sectools.GenerateCosignPolicy(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateCosignPolicy() error: %v", err)
	}
	if f.Path != ".cosign/policy.yaml" {
		t.Errorf("Path = %q, want %q", f.Path, ".cosign/policy.yaml")
	}
	if f.Mode != 0644 {
		t.Errorf("Mode = %#o, want %#o", f.Mode, 0644)
	}
	if f.Strategy != types.Overwrite {
		t.Errorf("Strategy = %v, want Overwrite", f.Strategy)
	}
	if f.Owner != "container-security" {
		t.Errorf("Owner = %q, want %q", f.Owner, "container-security")
	}
}

func TestGenerateCosignPolicy_Content(t *testing.T) {
	f, err := sectools.GenerateCosignPolicy(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateCosignPolicy() error: %v", err)
	}
	content := string(f.Content)

	if !strings.Contains(content, "apiVersion: policy.sigstore.dev/v1beta1") {
		t.Error("content should contain Sigstore policy API version")
	}
	if !strings.Contains(content, "kind: ClusterImagePolicy") {
		t.Error("content should contain ClusterImagePolicy kind")
	}
	if !strings.Contains(content, "fulcio.sigstore.dev") {
		t.Error("content should reference Fulcio for keyless signing")
	}
	if !strings.Contains(content, "rekor.sigstore.dev") {
		t.Error("content should reference Rekor transparency log")
	}
}
