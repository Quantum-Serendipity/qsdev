package cigeneration_test

import (
	"path"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/cigeneration"
	"github.com/Quantum-Serendipity/qsdev/internal/profile"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestGeneratedWorkflowsAllowlisted keeps every workflow qsdev generates
// startable under the Actions allowlist, as TestWorkflowActionsAllowlisted
// does for this repository's own workflows. Wave 1 generated a third-party
// Nix action into the ecosystem-ci job, so a generated workflow ended in
// startup_failure before any job ran.
func TestGeneratedWorkflowsAllowlisted(t *testing.T) {
	t.Parallel()

	scanners := []profile.VulnScannerType{
		profile.VulnScannerOSV, profile.VulnScannerSnyk, profile.VulnScannerGrype, profile.VulnScannerNone,
	}
	protections := []profile.CIProtectionType{profile.CIProtectionHardenRunner, profile.CIProtectionNone}
	ci := []ecosystem.CIPhaseGroup{{
		Phase:    ecosystem.CIPhaseTest,
		Commands: []ecosystem.CICommand{{Name: "test", Command: "true", Phase: ecosystem.CIPhaseTest}},
	}}

	checked := 0
	for _, builtin := range profile.DefaultProfileRegistry().List() {
		for _, vuln := range scanners {
			for _, protection := range protections {
				p := *builtin
				p.Scanning.Vulnerability = vuln
				p.Scanning.CIProtection = protection
				for _, in := range []profile.ProjectInputs{{}, {CI: ci}} {
					files, err := p.ConfigFiles(in)
					if err != nil {
						t.Fatalf("%s/%s/%s: ConfigFiles: %v", builtin.Name, vuln, protection, err)
					}
					for _, f := range files {
						if path.Dir(f.Path) != ".github/workflows" || !strings.HasSuffix(f.Path, ".yml") {
							continue
						}
						checked++
						for _, problem := range cigeneration.CheckWorkflowActions(f.Content) {
							if catalogImageProblem(problem) {
								continue
							}
							t.Errorf("%s/%s/%s (ecosystem CI %v): %s: %s",
								builtin.Name, vuln, protection, in.CI != nil, f.Path, problem)
						}
					}
				}
			}
		}
	}
	if checked == 0 {
		t.Fatal("no profile variant generated a workflow; the generator or the layout changed")
	}
}

// catalogImageProblem reports whether a CheckWorkflowActions problem is the
// Snyk step's digest-pinned catalog container image (uses: docker://...).
// The local allowlist copy refuses every docker:// step because GitHub does
// not document whether "Allow select actions" governs container steps;
// that open question belongs to the Snyk step's design, not to the
// action allowlist this test guards, so only that exact catalog image is
// passed over and any other docker:// step still fails.
func catalogImageProblem(problem string) bool {
	return strings.HasSuffix(problem, " uses "+cigeneration.ImageSnyk.String()+", which the Actions allowlist does not admit")
}
