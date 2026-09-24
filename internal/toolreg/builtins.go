package toolreg

import (
	"maps"

	"github.com/Quantum-Serendipity/qsdev/internal/sectools"
	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// builtinBehaviors returns the Go behavior hooks this package attaches to
// catalog tools when the default registry is built (see Default). Tools
// whose only lifecycle effect is recording themselves in EnabledTools need
// no Enable/DisableFunc: the enable and disable commands already do that.
func builtinBehaviors() map[string]ToolBehavior {
	m := coreBehaviors()
	maps.Copy(m, gitWorkflowBehaviors())
	maps.Copy(m, shellBehaviors())
	maps.Copy(m, consultingWorkflowBehaviors())
	return m
}

func coreBehaviors() map[string]ToolBehavior {
	return map[string]ToolBehavior{
		ToolVersionSentinel: {
			EnableFunc: func(a *types.WizardAnswers) {
				a.AgentTools.VersionSentinel = true
				if a.AgentTools.VersionSentinelHours == 0 {
					a.AgentTools.VersionSentinelHours = 24
				}
			},
			DisableFunc: func(a *types.WizardAnswers) {
				a.AgentTools.VersionSentinel = false
			},
			SectionDataFunc: func(answers types.WizardAnswers, ecoReg *ecosystem.Registry) map[string]any {
				report := ecosystem.LanguageManifestCoverage(answers.Languages, ecoReg)
				covered := make([]string, len(report.Covered))
				for i, m := range report.Covered {
					covered[i] = m.Path
				}
				uncovered := make([]string, len(report.Uncovered))
				for i, m := range report.Uncovered {
					uncovered[i] = m.Path
				}
				return map[string]any{
					"Covered":   covered,
					"Uncovered": uncovered,
				}
			},
		},
		ToolSemble: {
			EnableFunc: func(a *types.WizardAnswers) {
				a.AgentTools.SembleEnabled = true
				if a.AgentTools.SembleMode == "" {
					a.AgentTools.SembleMode = "mcp"
				}
			},
			DisableFunc: func(a *types.WizardAnswers) {
				a.AgentTools.SembleEnabled = false
				a.MCPServers = sliceutil.Remove(a.MCPServers, ToolSemble)
			},
		},
		"semgrep": {
			GenerateFunc: func(a types.WizardAnswers) ([]types.GeneratedFile, error) {
				f, err := sectools.GenerateSemgrepYml(a, ecosystem.DefaultRegistry())
				if err != nil {
					return nil, err
				}
				return []types.GeneratedFile{*f}, nil
			},
		},
		"gitleaks": {
			GenerateFunc: func(a types.WizardAnswers) ([]types.GeneratedFile, error) {
				f, err := sectools.GenerateGitleaksToml(a)
				if err != nil {
					return nil, err
				}
				return []types.GeneratedFile{*f}, nil
			},
		},
		"opengrep": {
			GenerateFunc: func(a types.WizardAnswers) ([]types.GeneratedFile, error) {
				// Delivers the package derivation plus the embedded core rule
				// library the security-scan task runs `opengrep scan` against.
				return sectools.GenerateOpengrepFiles(a)
			},
		},
		"container-security": {
			DetectFunc: func(d types.DetectedProject) bool {
				return d.HasDockerfile
			},
			GenerateFunc: func(a types.WizardAnswers) ([]types.GeneratedFile, error) {
				syft, err := sectools.GenerateSyftYaml(a)
				if err != nil {
					return nil, err
				}
				grype, err := sectools.GenerateGrypeYaml(a)
				if err != nil {
					return nil, err
				}
				cosign, err := sectools.GenerateCosignPolicy(a)
				if err != nil {
					return nil, err
				}
				return []types.GeneratedFile{*syft, *grype, *cosign}, nil
			},
		},
		"license-compliance": {
			GenerateFunc: func(a types.WizardAnswers) ([]types.GeneratedFile, error) {
				scancode, err := sectools.GenerateScancodeYml(a)
				if err != nil {
					return nil, err
				}
				exceptions, err := sectools.GenerateLicenseExceptionsYml()
				if err != nil {
					return nil, err
				}
				return []types.GeneratedFile{*scancode, *exceptions}, nil
			},
		},
	}
}
