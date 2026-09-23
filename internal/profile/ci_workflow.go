package profile

import (
	"bytes"
	"fmt"
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
	ActionSnyk         cigeneration.ActionRef
	ActionGrype        cigeneration.ActionRef

	// LockChecks are the manifest/lock-file alternatives the lock file step
	// enforces, derived from the ecosystem catalog so the CI gate and drift
	// detection cover the same ecosystems.
	LockChecks []ecosystem.ManifestLockfiles
}

// securityScanWorkflowTmpl is parsed once from the embedded templates, so a
// broken template fails the tests (and startup) instead of being written
// over a working workflow at generation time.
var securityScanWorkflowTmpl = template.Must(
	template.New("security-scan-workflow.yml.tmpl").Option("missingkey=error").
		ParseFS(templateFS, "templates/security-scan-workflow.yml.tmpl"))

// generatesSecurityScanWorkflow reports whether the profile's scanning
// settings call for the CI security-scan workflow.
func (p *InfraProfile) generatesSecurityScanWorkflow() bool {
	return p.Scanning.Vulnerability != VulnScannerNone || p.Scanning.CIProtection != CIProtectionNone
}

// generateSecurityScanWorkflow produces .github/workflows/security-scan.yml.
func (p *InfraProfile) generateSecurityScanWorkflow() (types.GeneratedFile, error) {
	data := CIWorkflowData{
		HasHardenRunner: p.Scanning.CIProtection == CIProtectionHardenRunner,
		HasOSV:          p.Scanning.Vulnerability == VulnScannerOSV,
		HasSnyk:         p.Scanning.Vulnerability == VulnScannerSnyk,
		HasGrype:        p.Scanning.Vulnerability == VulnScannerGrype,

		ActionHardenRunner: cigeneration.ActionHardenRunner,
		ActionCheckout:     cigeneration.ActionCheckout,
		ActionOSVScanner:   cigeneration.ActionOSVScanner,
		ActionSnyk:         cigeneration.ActionSnyk,
		ActionGrype:        cigeneration.ActionGrype,

		LockChecks: ecosystem.GroupedManifestLockfiles(),
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
