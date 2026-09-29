package profile

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/internal/cigeneration"
)

func TestGenerateSecurityScanWorkflow_ConsultingDefault(t *testing.T) {
	f := mustWorkflow(t, ConsultingDefault)

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
	f := mustWorkflow(t, Enterprise)

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

	f := mustWorkflow(t, p)
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

	f := mustWorkflow(t, p)
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
			f := mustWorkflow(t, p)
			content := string(f.Content)

			if !strings.Contains(content, "Validate lock files") {
				t.Error("expected lock file validation step in workflow")
			}
		})
	}
}

// TestGenerateSecurityScanWorkflow_LockFileValidationCanFail guards a property
// that TestGenerateSecurityScanWorkflow_LockFileValidationAlwaysPresent cannot:
// that the step is capable of failing at all. The original implementation
// asserted working-tree cleanliness via `git diff --exit-code` and swallowed the
// result into an echoed warning. Both halves were broken — actions/checkout
// produces a pristine tree, so the comparison is clean by construction and the
// check could never fire regardless of what was committed.
func TestGenerateSecurityScanWorkflow_LockFileValidationCanFail(t *testing.T) {
	profiles := []*InfraProfile{ConsultingDefault, StartupGitHub, Enterprise}

	for _, p := range profiles {
		t.Run(p.Name, func(t *testing.T) {
			content := string(mustWorkflow(t, p).Content)

			if !strings.Contains(content, "Lock file validation failed.") {
				t.Error("lock file validation must report failure and exit non-zero")
			}
			if strings.Contains(content, "git diff --exit-code") {
				t.Error("lock file validation must not assert working-tree cleanliness; " +
					"actions/checkout makes that comparison vacuous")
			}
			// The invariant worth enforcing: a committed manifest implies a
			// committed lock file, so dependencies cannot resolve at build time.
			if !strings.Contains(content, "require_lock package.json") {
				t.Error("expected package.json to require a lock file")
			}
			if !strings.Contains(content, `require_lock go.mod "" go.sum`) {
				t.Error("expected go.mod to require go.sum")
			}
			// A dependency-free module correctly has no go.sum; requiring one
			// unconditionally would be a false positive.
			if !strings.Contains(content, "go.mod) grep -qE '^[[:space:]]*require'") {
				t.Error("go.sum should only be required when go.mod declares requirements")
			}
		})
	}
}

func TestGenerateSecurityScanWorkflow_ValidYAML(t *testing.T) {
	profiles := []*InfraProfile{ConsultingDefault, StartupGitHub, Enterprise}

	for _, p := range profiles {
		t.Run(p.Name, func(t *testing.T) {
			f := mustWorkflow(t, p)
			var parsed map[string]any
			if err := yaml.Unmarshal(f.Content, &parsed); err != nil {
				t.Errorf("generated workflow is not valid YAML: %v\nContent:\n%s", err, string(f.Content))
			}
		})
	}
}

// workflowSteps parses the generated security-scan job's steps.
func workflowSteps(t *testing.T, p *InfraProfile) []workflowStep {
	t.Helper()
	var wf struct {
		Jobs map[string]struct {
			Steps []workflowStep `yaml:"steps"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(mustWorkflow(t, p).Content, &wf); err != nil {
		t.Fatalf("parsing workflow: %v", err)
	}
	return wf.Jobs["security-scan"].Steps
}

type workflowStep struct {
	Name string         `yaml:"name"`
	Uses string         `yaml:"uses"`
	Env  map[string]any `yaml:"env"`
	With map[string]any `yaml:"with"`
}

// TestGenerateSecurityScanWorkflow_SnykImageDigestPinned is the F483
// regression test. The Snyk step used snyk/actions@<sha>, whose action.yml runs
// the mutable docker://snyk/snyk:node tag, so the image that received
// SNYK_TOKEN was not pinned by the SHA. The step now runs the image itself,
// pinned by digest from the catalog.
func TestGenerateSecurityScanWorkflow_SnykImageDigestPinned(t *testing.T) {
	t.Parallel()

	var snyk *workflowStep
	steps := workflowSteps(t, Enterprise)
	for i := range steps {
		if steps[i].Name == "Snyk Security Scan" {
			snyk = &steps[i]
		}
	}
	if snyk == nil {
		t.Fatal("enterprise workflow has no Snyk Security Scan step")
	}

	checks := []struct {
		name string
		ok   bool
		msg  string
	}{
		{"uses catalog image", snyk.Uses == cigeneration.ImageSnyk.String(),
			"uses = " + snyk.Uses + ", want " + cigeneration.ImageSnyk.String()},
		{"not the tag-running action", !strings.Contains(snyk.Uses, "snyk/actions"),
			"step must not use snyk/actions, which runs a mutable image tag"},
		{"runs snyk test over all projects", snyk.With["args"] == "snyk test --all-projects",
			fmt.Sprintf("with.args = %v", snyk.With["args"])},
		{"passes the token", snyk.Env["SNYK_TOKEN"] == "${{ secrets.SNYK_TOKEN }}",
			fmt.Sprintf("env.SNYK_TOKEN = %v", snyk.Env["SNYK_TOKEN"])},
	}
	for _, c := range checks {
		t.Run(c.name, func(t *testing.T) {
			if !c.ok {
				t.Error(c.msg)
			}
		})
	}
}

// immutableUsesRe matches the two immutable forms a generated step may use: an
// action pinned to a full commit SHA, or a container image pinned by digest.
var immutableUsesRe = regexp.MustCompile(`^([^@\s]+@[0-9a-f]{40}|docker://[^@\s]+@sha256:[0-9a-f]{64})$`)

// TestGenerateSecurityScanWorkflow_EveryUsesImmutable guards every scanner
// variant, so a step cannot reach generated projects on a tag or branch.
func TestGenerateSecurityScanWorkflow_EveryUsesImmutable(t *testing.T) {
	t.Parallel()

	scanners := []VulnScannerType{VulnScannerOSV, VulnScannerSnyk, VulnScannerGrype}
	for _, v := range scanners {
		t.Run(string(v), func(t *testing.T) {
			t.Parallel()
			p := &InfraProfile{Name: string(v), Scanning: ScanningConfig{
				Vulnerability: v,
				CIProtection:  CIProtectionHardenRunner,
			}}
			for _, s := range workflowSteps(t, p) {
				if s.Uses != "" && !immutableUsesRe.MatchString(s.Uses) {
					t.Errorf("step %q uses %q, which is not SHA- or digest-pinned", s.Name, s.Uses)
				}
			}
		})
	}
}
