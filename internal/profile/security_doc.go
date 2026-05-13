package profile

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// SecurityDocData holds data for rendering the security overview documentation template.
type SecurityDocData struct {
	ProfileName    string
	VulnScanner    string
	BehavioralTool string
	CIProtection   string
	UpdateTool     string
	AgeGatingDays  int
	SBOMGenerator  string
}

// generateSecurityDoc produces docs/security-overview.md.
func (p *InfraProfile) generateSecurityDoc() types.GeneratedFile {
	data := SecurityDocData{
		ProfileName:    p.Name,
		VulnScanner:    string(p.Scanning.Vulnerability),
		BehavioralTool: string(p.Scanning.Behavioral),
		CIProtection:   string(p.Scanning.CIProtection),
		UpdateTool:     string(p.Updates.Type),
		AgeGatingDays:  p.Updates.AgeGatingDays,
		SBOMGenerator:  string(p.SBOM.Generator),
	}

	tmplContent, err := templateFS.ReadFile("templates/security-overview.md.tmpl")
	if err != nil {
		return types.GeneratedFile{
			Path:     "docs/security-overview.md",
			Content:  []byte("# Error: could not load security overview template\n"),
			Mode:     0o644,
			Strategy: types.Overwrite,
		}
	}

	tmpl, err := template.New("security-doc").Parse(string(tmplContent))
	if err != nil {
		return types.GeneratedFile{
			Path:     "docs/security-overview.md",
			Content:  []byte(fmt.Sprintf("# Error parsing template: %v\n", err)),
			Mode:     0o644,
			Strategy: types.Overwrite,
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return types.GeneratedFile{
			Path:     "docs/security-overview.md",
			Content:  []byte(fmt.Sprintf("# Error rendering template: %v\n", err)),
			Mode:     0o644,
			Strategy: types.Overwrite,
		}
	}

	return types.GeneratedFile{
		Path:     "docs/security-overview.md",
		Content:  buf.Bytes(),
		Mode:     0o644,
		Strategy: types.Overwrite,
	}
}
