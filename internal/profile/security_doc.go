package profile

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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

// securityDocTmpl is parsed once from the embedded templates, so a broken
// template fails the tests (and startup) instead of being written over the
// committed security overview at generation time.
var securityDocTmpl = template.Must(
	template.New("security-overview.md.tmpl").Option("missingkey=error").
		ParseFS(templateFS, "templates/security-overview.md.tmpl"))

// generateSecurityDoc produces docs/security-overview.md.
func (p *InfraProfile) generateSecurityDoc() (types.GeneratedFile, error) {
	data := SecurityDocData{
		ProfileName:    p.Name,
		VulnScanner:    string(p.Scanning.Vulnerability),
		BehavioralTool: string(p.Scanning.Behavioral),
		CIProtection:   string(p.Scanning.CIProtection),
		UpdateTool:     string(p.Updates.Type),
		AgeGatingDays:  p.Updates.AgeGatingDays,
		SBOMGenerator:  string(p.SBOM.Generator),
	}

	var buf bytes.Buffer
	if err := securityDocTmpl.Execute(&buf, data); err != nil {
		return types.GeneratedFile{}, fmt.Errorf("rendering security overview: %w", err)
	}

	return types.GeneratedFile{
		Path:     "docs/security-overview.md",
		Content:  buf.Bytes(),
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.Overwrite,
	}, nil
}
