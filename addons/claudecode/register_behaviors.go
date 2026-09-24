package claudecode

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/tier"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// init only records the provider; the catalog and tool registry are built on
// first use (see toolreg.Default), after main has configured the process. The
// manifests the provider reads are embedded at build time, so a load failure
// there is a broken build and panics rather than leaving tools silently
// without their GenerateFuncs (which would surface later as "no files
// needed").
func init() {
	toolreg.RegisterBehaviors(registerToolBehaviors)
}

func registerToolBehaviors(r *toolreg.Registry) {
	registerOperationSkillTools(r)
	registerMCPServerContent(r)
	registerAgentToolGenerators(r)
	registerConsultingAgentGenerators(r)
	registerConsultingWorkflowGenerators(r)
	registerDocToolBehaviors(r)
}

// registerOperationSkillTools registers one always-on tool per entry of the
// operation-skill manifest, so the manifest deployOperationSkills reads is the
// single source of truth for which operation skills exist and are enabled.
func registerOperationSkillTools(r *toolreg.Registry) {
	manifest, err := loadQsdevOpsManifest()
	if err != nil {
		slog.Warn("failed to load ops manifest; operation skills not registered", "error", err)
		return
	}
	for _, s := range manifest.Skills {
		if err := r.Register(toolreg.NewOperationSkillTool(s.Name, s.Description)); err != nil {
			slog.Warn("registering operation skill tool", "skill", s.Name, "error", err)
		}
	}
}

func registerMCPServerContent(r *toolreg.Registry) {
	cat, err := catalog.Default()
	if err != nil {
		panic(fmt.Sprintf("claudecode: loading embedded catalog: %v", err))
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
			SharedContent: map[toolreg.SharedSection]toolreg.SharedContentFunc{
				{Path: ".mcp.json", SectionID: mcpSectionID}: mcpServerContentFunc(serverName),
			},
		})
	}
}

func mcpServerContentFunc(serverName string) toolreg.SharedContentFunc {
	return func(answers types.WizardAnswers) ([]byte, error) {
		cat, err := catalog.Default()
		if err != nil {
			return nil, fmt.Errorf("loading catalog for MCP server %q: %w", serverName, err)
		}
		def, ok := cat.MCPServer(serverName)
		if !ok {
			return nil, fmt.Errorf("unknown MCP server %q", serverName)
		}
		entry := catalogServerEntry(serverName, def, installedMCPServers(answers.ProjectRoot))
		if err := requirePinnedLaunch(serverName, entry); err != nil {
			return nil, err
		}
		return json.Marshal(entry)
	}
}

func registerAgentToolGenerators(r *toolreg.Registry) {
	// agent-postmortem and version-sentinel are catalog default_policy: always-on
	// (see internal/catalog/defaults.yaml). Always-on tools are emitted regardless
	// of tier by the generator's AlwaysOn loop (generator.go, gated on
	// tool.Default == toolreg.AlwaysOn) and by the enable path. These GenerateFuncs
	// therefore must NOT self-gate on tier — doing so caused CLAUDE.md to advertise
	// the skill at Standard while its SKILL.md was silently omitted (BL-P1-9).
	r.AttachBehavior("agent-postmortem", toolreg.ToolBehavior{
		GenerateFunc: func(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
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
			registry := ecosystem.DefaultRegistry()
			return generateVersionSentinelFiles(answers, registry)
		},
	})
}

func registerConsultingAgentGenerators(r *toolreg.Registry) {
	manifest, err := loadAgentManifest()
	if err != nil {
		panic(fmt.Sprintf("claudecode: loading embedded agent manifest: %v", err))
	}

	for _, a := range manifest.Agents {
		agentName := a.Name
		r.AttachBehavior("consulting-agent-"+agentName, toolreg.ToolBehavior{
			GenerateFunc: fullTierFile(func() (types.GeneratedFile, error) {
				return consultingAgentFile(agentName)
			}),
		})
	}
}

// fullTierFile adapts a single-file builder into a tool GenerateFunc that
// applies the same Full-tier policy as Generate's deployAgents and
// deployWorkflowSkills. Below Full it returns no files, so `enable` reports
// that the tool needs a higher tier instead of writing a file init/update
// would never regenerate.
func fullTierFile(build func() (types.GeneratedFile, error)) toolreg.GenerateFunc {
	return func(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
		if resolveTier(answers) < tier.Full {
			return nil, nil
		}
		f, err := build()
		if err != nil {
			return nil, err
		}
		return []types.GeneratedFile{f}, nil
	}
}

func registerConsultingWorkflowGenerators(r *toolreg.Registry) {
	manifest, err := loadConsultingSkillManifest()
	if err != nil {
		panic(fmt.Sprintf("claudecode: loading embedded consulting skill manifest: %v", err))
	}

	for _, skill := range manifest.Skills {
		skillName := skill.Name
		r.AttachBehavior("consulting-workflow-"+skillName, toolreg.ToolBehavior{
			GenerateFunc: fullTierFile(func() (types.GeneratedFile, error) {
				return consultingWorkflowFile(skillName)
			}),
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
