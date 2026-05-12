package profile

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGenerateSecurityScanWorkflow_ConsultingDefault(t *testing.T) {
	f := ConsultingDefault.generateSecurityScanWorkflow()

	if f.Path != ".github/workflows/security-scan.yml" {
		t.Errorf("Path = %q, want .github/workflows/security-scan.yml", f.Path)
	}

	content := string(f.Content)

	// ConsultingDefault has OSV + Harden-Runner, no Snyk.
	if !strings.Contains(content, "OSV Scanner") {
		t.Error("expected OSV Scanner step in consulting-default workflow")
	}
	if !strings.Contains(content, "Harden Runner") {
		t.Error("expected Harden Runner step in consulting-default workflow")
	}
	if strings.Contains(content, "Snyk") {
		t.Error("consulting-default workflow should not contain Snyk step")
	}

	// Lock file validation step is always present.
	if !strings.Contains(content, "Validate lock files") {
		t.Error("expected lock file validation step in workflow")
	}

	// Should be valid YAML.
	var parsed map[string]any
	if err := yaml.Unmarshal(f.Content, &parsed); err != nil {
		t.Errorf("generated workflow is not valid YAML: %v", err)
	}
}

func TestGenerateSecurityScanWorkflow_Enterprise(t *testing.T) {
	f := Enterprise.generateSecurityScanWorkflow()

	content := string(f.Content)

	// Enterprise has Snyk + Harden-Runner, no OSV.
	if !strings.Contains(content, "Snyk") {
		t.Error("expected Snyk step in enterprise workflow")
	}
	if !strings.Contains(content, "Harden Runner") {
		t.Error("expected Harden Runner step in enterprise workflow")
	}
	if strings.Contains(content, "OSV Scanner") {
		t.Error("enterprise workflow should not contain OSV Scanner step")
	}

	// Lock file validation step is always present.
	if !strings.Contains(content, "Validate lock files") {
		t.Error("expected lock file validation step in workflow")
	}

	// Should be valid YAML.
	var parsed map[string]any
	if err := yaml.Unmarshal(f.Content, &parsed); err != nil {
		t.Errorf("generated workflow is not valid YAML: %v", err)
	}
}

func TestGenerateSecurityScanWorkflow_HardenRunnerAbsent(t *testing.T) {
	p := &InfraProfile{
		Scanning: ScanningConfig{
			Vulnerability: VulnScannerOSV,
			Behavioral:    BehavioralSocket,
			CIProtection:  CIProtectionNone,
		},
	}

	f := p.generateSecurityScanWorkflow()
	content := string(f.Content)

	if strings.Contains(content, "Harden Runner") {
		t.Error("workflow should not contain Harden Runner when CIProtection is none")
	}
	if !strings.Contains(content, "OSV Scanner") {
		t.Error("expected OSV Scanner step")
	}
}

func TestGenerateSecurityScanWorkflow_GrypeScanner(t *testing.T) {
	p := &InfraProfile{
		Scanning: ScanningConfig{
			Vulnerability: VulnScannerGrype,
			CIProtection:  CIProtectionNone,
		},
	}

	f := p.generateSecurityScanWorkflow()
	content := string(f.Content)

	if !strings.Contains(content, "Grype") {
		t.Error("expected Grype step in workflow")
	}
	if strings.Contains(content, "OSV Scanner") {
		t.Error("workflow should not contain OSV Scanner when using Grype")
	}
	if strings.Contains(content, "Snyk") {
		t.Error("workflow should not contain Snyk when using Grype")
	}
}

func TestGenerateSecurityScanWorkflow_LockFileValidationAlwaysPresent(t *testing.T) {
	profiles := []*InfraProfile{ConsultingDefault, StartupGitHub, Enterprise}

	for _, p := range profiles {
		t.Run(p.Name, func(t *testing.T) {
			f := p.generateSecurityScanWorkflow()
			content := string(f.Content)

			if !strings.Contains(content, "Validate lock files") {
				t.Error("expected lock file validation step in workflow")
			}
		})
	}
}

func TestGenerateSecurityScanWorkflow_ValidYAML(t *testing.T) {
	profiles := []*InfraProfile{ConsultingDefault, StartupGitHub, Enterprise}

	for _, p := range profiles {
		t.Run(p.Name, func(t *testing.T) {
			f := p.generateSecurityScanWorkflow()
			var parsed map[string]any
			if err := yaml.Unmarshal(f.Content, &parsed); err != nil {
				t.Errorf("generated workflow is not valid YAML: %v\nContent:\n%s", err, string(f.Content))
			}
		})
	}
}
