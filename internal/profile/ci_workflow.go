package profile

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// CIWorkflowData holds data for rendering the security scan workflow template.
type CIWorkflowData struct {
	HasHardenRunner bool
	HasOSV          bool
	HasSnyk         bool
	HasGrype        bool
}

// generateSecurityScanWorkflow produces .github/workflows/security-scan.yml.
func (p *InfraProfile) generateSecurityScanWorkflow() types.GeneratedFile {
	data := CIWorkflowData{
		HasHardenRunner: p.Scanning.CIProtection == CIProtectionHardenRunner,
		HasOSV:          p.Scanning.Vulnerability == VulnScannerOSV,
		HasSnyk:         p.Scanning.Vulnerability == VulnScannerSnyk,
		HasGrype:        p.Scanning.Vulnerability == VulnScannerGrype,
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
