package toolreg

import (
	"github.com/Quantum-Serendipity/qsdev/internal/sectools"
	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func init() {
	r := DefaultRegistry()
	for name, b := range builtinBehaviors() {
		r.AttachBehavior(name, b)
	}
}

func builtinBehaviors() map[string]ToolBehavior {
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
				var modules []ecosystem.EcosystemModule
				configFor := func(mod ecosystem.EcosystemModule) ecosystem.ModuleConfig {
					for _, lang := range answers.Languages {
						if lang.Name == mod.Name() {
							return ecosystem.ToModuleConfig(lang)
						}
					}
					return ecosystem.ModuleConfig{}
				}
				for _, lang := range answers.Languages {
					if mod, ok := ecoReg.ByName(lang.Name); ok {
						modules = append(modules, mod)
					}
				}
				report := ecosystem.AggregateManifestCoverage(modules, configFor)
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
				f, err := sectools.GenerateOpengrepConfigYaml(a)
				if err != nil {
					return nil, err
				}
				return []types.GeneratedFile{*f}, nil
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
