package profile

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Quantum-Serendipity/qsdev/internal/cigeneration"
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
}

// generateSecurityScanWorkflow produces .github/workflows/security-scan.yml.
func (p *InfraProfile) generateSecurityScanWorkflow() types.GeneratedFile {
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
	}

	// Parse and render template
	tmplContent, err := templateFS.ReadFile("templates/security-scan-workflow.yml.tmpl")
	if err != nil {
		// Fallback: return a comment-only file
		return types.GeneratedFile{
			Path:     ".github/workflows/security-scan.yml",
			Content:  []byte("# Error: could not load workflow template\n"),
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.Overwrite,
		}
	}

	tmpl, err := template.New("workflow").Parse(string(tmplContent))
	if err != nil {
		return types.GeneratedFile{
			Path:     ".github/workflows/security-scan.yml",
			Content:  []byte(fmt.Sprintf("# Error parsing template: %v\n", err)),
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.Overwrite,
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return types.GeneratedFile{
			Path:     ".github/workflows/security-scan.yml",
			Content:  []byte(fmt.Sprintf("# Error rendering template: %v\n", err)),
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.Overwrite,
		}
	}

	return types.GeneratedFile{
		Path:     ".github/workflows/security-scan.yml",
		Content:  buf.Bytes(),
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.Overwrite,
	}
}
