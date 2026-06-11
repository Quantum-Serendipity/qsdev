package claudecode

import (
	"encoding/json"
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/tier"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func init() {
	r := toolreg.DefaultRegistry()

	registerMCPServerContent(r)
	registerAgentToolGenerators(r)
	registerConsultingAgentGenerators(r)
	registerConsultingWorkflowGenerators(r)
	registerDocToolBehaviors(r)
}

func registerMCPServerContent(r *toolreg.Registry) {
	cat, err := catalog.Default()
	if err != nil {
		return
	}
	for name, def := range cat.Tools() {
		if def.MCPServerName == "" {
			continue
		}
		serverName := def.MCPServerName

		var mcpSectionID string
		for _, of := range def.OwnedFiles {
			if of.Path == ".mcp.json" && of.Ownership == "shared" {
				mcpSectionID = of.SectionID
				break
			}
		}
		if mcpSectionID == "" {
			continue
		}

		r.AttachBehavior(name, toolreg.ToolBehavior{
			SharedContent: map[string]toolreg.SharedContentFunc{
				mcpSectionID: mcpServerContentFunc(serverName),
			},
		})
	}
}

func mcpServerContentFunc(serverName string) toolreg.SharedContentFunc {
	return func(_ types.WizardAnswers) ([]byte, error) {
		cat, err := catalog.Default()
		if err != nil {
			return nil, fmt.Errorf("loading catalog for MCP server %q: %w", serverName, err)
		}
		def, ok := cat.MCPServer(serverName)
		if !ok {
			return nil, fmt.Errorf("unknown MCP server %q", serverName)
		}
		entry := MCPServerEntry{
			Command: def.Command,
			Args:    def.Args,
			Env:     def.Env,
		}
		return json.Marshal(entry)
	}
}

func registerAgentToolGenerators(r *toolreg.Registry) {
	r.AttachBehavior("agent-postmortem", toolreg.ToolBehavior{
		GenerateFunc: func(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
			if resolveTier(answers) < tier.Full {
				return nil, nil
			}
			registry := ecosystem.DefaultRegistry()
			f, err := generatePostmortemSkill(answers, registry)
			if err != nil {
				return nil, err
			}
			if f == nil {
				return nil, nil
			}
			return []types.GeneratedFile{*f}, nil
		},
	})

	r.AttachBehavior("version-sentinel", toolreg.ToolBehavior{
		GenerateFunc: func(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
			if resolveTier(answers) < tier.Full {
				return nil, nil
			}
			registry := ecosystem.DefaultRegistry()
			return generateVersionSentinelFiles(answers, registry)
		},
	})
}

func registerConsultingAgentGenerators(r *toolreg.Registry) {
	manifest, err := loadAgentManifest()
	if err != nil {
		return
	}

	for _, a := range manifest.Agents {
		agentName := a.Name
		toolKey := "consulting-agent-" + agentName

		r.AttachBehavior(toolKey, toolreg.ToolBehavior{
			GenerateFunc: func(_ types.WizardAnswers) ([]types.GeneratedFile, error) {
				content, err := templateFS.ReadFile("templates/agents/" + agentName + ".md")
				if err != nil {
					return nil, fmt.Errorf("reading agent template %q: %w", agentName, err)
				}
				return []types.GeneratedFile{{
					Path:    ".claude/agents/" + agentName + ".md",
					Content: content,
					Mode:    0o644,
					Owner:   toolKey,
				}}, nil
			},
		})
	}
}

func registerConsultingWorkflowGenerators(r *toolreg.Registry) {
	manifest, err := loadConsultingSkillManifest()
	if err != nil {
		return
	}

	for _, skill := range manifest.Skills {
		skillName := skill.Name
		toolKey := "consulting-workflow-" + skillName

		r.AttachBehavior(toolKey, toolreg.ToolBehavior{
			GenerateFunc: func(_ types.WizardAnswers) ([]types.GeneratedFile, error) {
				content, err := templateFS.ReadFile("templates/skills/" + skillName + "/SKILL.md")
				if err != nil {
					return nil, fmt.Errorf("reading workflow skill template %q: %w", skillName, err)
				}
				return []types.GeneratedFile{{
					Path:    ".claude/skills/" + skillName + "/SKILL.md",
					Content: content,
					Mode:    0o644,
					Owner:   toolKey,
				}}, nil
			},
		})
	}
}

func registerDocToolBehaviors(r *toolreg.Registry) {
	r.AttachBehavior("local-docs-devdocs", toolreg.ToolBehavior{
		DetectFunc: func(d types.DetectedProject) bool {
			return d.HasGoMod || d.HasPackageJSON || d.HasPyProject ||
				d.HasCargoToml || d.HasPomXML || d.HasBuildGradle || d.HasCsproj
		},
	})

	r.AttachBehavior("mcp-nixos", toolreg.ToolBehavior{
		DetectFunc: func(d types.DetectedProject) bool {
			return d.HasDevenvNix
		},
	})

	r.AttachBehavior("lookup-docs", toolreg.ToolBehavior{
		GenerateFunc: func(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
			if resolveTier(answers) < tier.Full {
				return nil, nil
			}
			f, err := generateLookupDocsSkill(answers)
			if err != nil {
				return nil, err
			}
			if f == nil {
				return nil, nil
			}
			return []types.GeneratedFile{*f}, nil
		},
	})
}
