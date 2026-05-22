package toolreg

import (
	"github.com/Quantum-Serendipity/qsdev/internal/sectools"
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
		"version-sentinel": {
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
		"semble": {
			EnableFunc: func(a *types.WizardAnswers) {
				a.AgentTools.SembleEnabled = true
				if a.AgentTools.SembleMode == "" {
					a.AgentTools.SembleMode = "mcp"
				}
			},
			DisableFunc: func(a *types.WizardAnswers) {
				a.AgentTools.SembleEnabled = false
				a.MCPServers = removeStr(a.MCPServers, "semble")
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
		"container-security": {
			DetectFunc: func(d types.DetectedProject) bool {
				return d.HasDockerfile
			},
			GenerateFunc: func(a types.WizardAnswers) ([]types.GeneratedFile, error) {
				grype, err := sectools.GenerateGrypeYaml(a)
				if err != nil {
					return nil, err
				}
				cosign, err := sectools.GenerateCosignPolicy(a)
				if err != nil {
					return nil, err
				}
				return []types.GeneratedFile{*grype, *cosign}, nil
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

func removeStr(ss []string, s string) []string {
	var result []string
	for _, v := range ss {
		if v != s {
			result = append(result, v)
		}
	}
	return result
}
