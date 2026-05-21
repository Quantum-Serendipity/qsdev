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
		"attach-guard": {
			EnableFunc: func(a *types.WizardAnswers) {
				a.Hooks.SafetyBlock = true
			},
			DisableFunc: func(a *types.WizardAnswers) {
				a.Hooks.SafetyBlock = false
			},
		},
		"agent-postmortem": {
			EnableFunc: func(a *types.WizardAnswers) {
				a.AgentTools.PostmortemEnabled = true
			},
			DisableFunc: func(a *types.WizardAnswers) {
				a.AgentTools.PostmortemEnabled = false
			},
		},
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
		"trail-of-bits-skills": {
			EnableFunc: func(a *types.WizardAnswers) {
				if !containsStr(a.Skills, "security-review") {
					a.Skills = append(a.Skills, "security-review")
				}
			},
			DisableFunc: func(a *types.WizardAnswers) {
				a.Skills = removeStr(a.Skills, "security-review")
			},
		},
		"context7": {
			EnableFunc: func(a *types.WizardAnswers) {
				if !containsStr(a.MCPServers, "context7") {
					a.MCPServers = append(a.MCPServers, "context7")
				}
			},
			DisableFunc: func(a *types.WizardAnswers) {
				a.MCPServers = removeStr(a.MCPServers, "context7")
			},
		},
		"github-mcp": {
			EnableFunc: func(a *types.WizardAnswers) {
				if !containsStr(a.MCPServers, "github") {
					a.MCPServers = append(a.MCPServers, "github")
				}
			},
			DisableFunc: func(a *types.WizardAnswers) {
				a.MCPServers = removeStr(a.MCPServers, "github")
			},
		},
		"socket-dev-mcp": {
			EnableFunc: func(a *types.WizardAnswers) {
				if !containsStr(a.MCPServers, "socket") {
					a.MCPServers = append(a.MCPServers, "socket")
				}
			},
			DisableFunc: func(a *types.WizardAnswers) {
				a.MCPServers = removeStr(a.MCPServers, "socket")
			},
		},
		"postgres-mcp": {
			EnableFunc: func(a *types.WizardAnswers) {
				if !containsStr(a.MCPServers, "postgres") {
					a.MCPServers = append(a.MCPServers, "postgres")
				}
			},
			DisableFunc: func(a *types.WizardAnswers) {
				a.MCPServers = removeStr(a.MCPServers, "postgres")
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
