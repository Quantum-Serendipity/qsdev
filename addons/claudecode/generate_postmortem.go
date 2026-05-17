package claudecode

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
	"github.com/Quantum-Serendipity/qsdev/internal/tmpl"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

type postmortemTemplateData struct {
	VerificationCommands []string
}

func generatePostmortemSkill(answers types.WizardAnswers, registry *ecosystem.Registry) (*types.GeneratedFile, error) {
	if !answers.AgentTools.PostmortemEnabled {
		return nil, nil
	}

	cmds := collectVerificationCommands(answers, registry)

	renderer, err := tmpl.NewMarkdownRenderer(templateFS, "templates")
	if err != nil {
		return nil, fmt.Errorf("creating renderer: %w", err)
	}

	data := postmortemTemplateData{
		VerificationCommands: cmds,
	}

	content, err := renderer.Render("skills/agent-postmortem.md", data)
	if err != nil {
		return nil, fmt.Errorf("rendering postmortem skill: %w", err)
	}

	return &types.GeneratedFile{
		Path:     ".claude/skills/agent-postmortem/SKILL.md",
		Content:  content,
		Mode:     0o644,
		Strategy: types.LibraryManaged,
	}, nil
}

func collectVerificationCommands(answers types.WizardAnswers, registry *ecosystem.Registry) []string {
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
		if mod, ok := registry.ByName(lang.Name); ok {
			modules = append(modules, mod)
		}
	}

	agg := ecosystem.AggregateVerificationCommands(modules, configFor)
	return sliceutil.Dedup(agg.All())
}
