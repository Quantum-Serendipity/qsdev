package claudecode

import (
	"encoding/json"
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func init() {
	r := toolreg.DefaultRegistry()

	registerMCPServerContent(r)
	registerConsultingAgentGenerators(r)
	registerConsultingWorkflowGenerators(r)
}

func registerMCPServerContent(r *toolreg.Registry) {
	cat := catalog.MustDefault()
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
		entry, ok := knownMCPServers[serverName]
		if !ok {
			return nil, fmt.Errorf("unknown MCP server %q", serverName)
		}
		return json.Marshal(entry)
	}
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
