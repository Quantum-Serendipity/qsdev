package profile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/Quantum-Serendipity/qsdev/internal/cigeneration"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// CIWorkflowData holds data for rendering the security scan workflow template.
type CIWorkflowData struct {
	HasHardenRunner bool
	HasOSV          bool
	HasSnyk         bool
	HasGrype        bool

	// Action pins come from the shared catalog rather than being written into
	// the template. Hardcoding them there created a second source of truth that
	// Dependabot cannot see and no test compared, which is how the OSV scanner
	// pin sat four releases stale while its govulncheck analysis silently failed.
	ActionHardenRunner cigeneration.ActionRef
	ActionCheckout     cigeneration.ActionRef
	ActionOSVScanner   cigeneration.ActionRef
	ActionGrype        cigeneration.ActionRef
	ActionInstallNix   cigeneration.ActionRef

	// ImageSnyk is run as a digest-pinned container step rather than through
	// snyk/actions, whose action.yml runs a mutable image tag.
	ImageSnyk cigeneration.ImageRef

	// LockChecks are the manifest/lock-file alternatives the lock file step
	// enforces, derived from the ecosystem catalog so the CI gate and drift
	// detection cover the same ecosystems.
	LockChecks []ecosystem.ManifestLockfiles

	// EcosystemCI are the project's ecosystem CI commands grouped by phase
	// (ProjectInputs.CI). A non-empty list adds the ecosystem-ci job.
	EcosystemCI []ecosystem.CIPhaseGroup
}

// securityScanWorkflowTmpl is parsed once from the embedded templates, so a
// broken template fails the tests (and startup) instead of being written
// over a working workflow at generation time.
var securityScanWorkflowTmpl = template.Must(
	template.New("security-scan-workflow.yml.tmpl").Option("missingkey=error").
		Funcs(template.FuncMap{
			"yamlString":  yamlString,
			"indentBlock": indentBlock,
			"oneLine":     oneLine,
		}).
		ParseFS(templateFS, "templates/security-scan-workflow.yml.tmpl"))

// yamlString renders s as a double-quoted YAML scalar. JSON string syntax is
// a subset of YAML's double-quoted style, so every character is escaped
// correctly.
func yamlString(s string) (string, error) {
	b, err := json.Marshal(s)
	if err != nil {
		return "", fmt.Errorf("quoting %q for YAML: %w", s, err)
	}
	return string(b), nil
}

// indentBlock prefixes every line of s after the first with n spaces, for
// the body of a YAML block scalar whose first line the template indents.
func indentBlock(n int, s string) string {
	return strings.ReplaceAll(strings.TrimRight(s, "\n"), "\n", "\n"+strings.Repeat(" ", n))
}

// oneLine collapses s to a single line for use in a YAML comment.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// validateEcosystemCI rejects CI commands the workflow cannot carry
// verbatim. GitHub evaluates ${{ }} expressions in run blocks before the
// shell sees them, so a command containing one would run something other
// than what the module declared.
func validateEcosystemCI(groups []ecosystem.CIPhaseGroup) error {
	for _, g := range groups {
		for _, c := range g.Commands {
			if strings.Contains(c.Command, "${{") || strings.Contains(c.Name, "${{") {
				return fmt.Errorf("CI command %q contains a GitHub Actions expression (${{), which the workflow would evaluate", c.Name)
			}
			if strings.TrimSpace(c.Command) == "" {
				return fmt.Errorf("CI command %q has an empty command line", c.Name)
			}
		}
	}
	return nil
}

// generatesSecurityScanWorkflow reports whether the profile's scanning
// settings call for the CI security-scan workflow.
func (p *InfraProfile) generatesSecurityScanWorkflow() bool {
	return p.Scanning.Vulnerability != VulnScannerNone || p.Scanning.CIProtection != CIProtectionNone
}

// generateSecurityScanWorkflow produces .github/workflows/security-scan.yml:
// the lock-file and scanner job, plus the ecosystem-ci job that runs the
// project's ecosystem CI commands (in.CI) when it has any.
func (p *InfraProfile) generateSecurityScanWorkflow(in ProjectInputs) (types.GeneratedFile, error) {
	if err := validateEcosystemCI(in.CI); err != nil {
		return types.GeneratedFile{}, fmt.Errorf("rendering security-scan workflow: %w", err)
	}

	data := CIWorkflowData{
		HasHardenRunner: p.Scanning.CIProtection == CIProtectionHardenRunner,
		HasOSV:          p.Scanning.Vulnerability == VulnScannerOSV,
		HasSnyk:         p.Scanning.Vulnerability == VulnScannerSnyk,
		HasGrype:        p.Scanning.Vulnerability == VulnScannerGrype,

		ActionHardenRunner: cigeneration.ActionHardenRunner,
		ActionCheckout:     cigeneration.ActionCheckout,
		ActionOSVScanner:   cigeneration.ActionOSVScanner,
		ActionGrype:        cigeneration.ActionGrype,
		ActionInstallNix:   cigeneration.ActionInstallNix,
		ImageSnyk:          cigeneration.ImageSnyk,

		LockChecks:  ecosystem.GroupedManifestLockfiles(),
		EcosystemCI: in.CI,
	}

	var buf bytes.Buffer
	if err := securityScanWorkflowTmpl.Execute(&buf, data); err != nil {
		return types.GeneratedFile{}, fmt.Errorf("rendering security-scan workflow: %w", err)
	}

	return types.GeneratedFile{
		Path:     ".github/workflows/security-scan.yml",
		Content:  buf.Bytes(),
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.Overwrite,
	}, nil
}
