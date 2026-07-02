package claudecode

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
	"github.com/Quantum-Serendipity/qsdev/internal/tmpl"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
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

	// The template body has no YAML front-matter, but Claude Code only loads a
	// SKILL.md that begins with a name/description front-matter block. Synthesize
	// it (idempotent — a no-op should the template ever grow its own). Without
	// this the postmortem skill deploys to the right path but silently fails to
	// load (BL-P1-10 / DEFECT-8 class).
	content, err = prependSkillFrontMatter("agent-postmortem", postmortemSkillDescription(), content)
	if err != nil {
		return nil, err
	}

	return &types.GeneratedFile{
		Path:     ".claude/skills/agent-postmortem/SKILL.md",
		Content:  content,
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.LibraryManaged,
	}, nil
}

// postmortemSkillDescription returns the agent-postmortem skill description from
// the catalog (the single source of truth), falling back to a static string if
// the catalog cannot be loaded.
func postmortemSkillDescription() string {
	const fallback = "Verification skill requiring agents to validate changes against build/test/lint after implementation"
	cat, err := catalog.Default()
	if err != nil {
		return fallback
	}
	if def, ok := cat.Tool("agent-postmortem"); ok && def.Description != "" {
		return def.Description
	}
	return fallback
}

func collectVerificationCommands(answers types.WizardAnswers, registry *ecosystem.Registry) []string {
	modules, configFor := resolveLanguageModules(answers, registry)
	agg := ecosystem.AggregateVerificationCommands(modules, configFor)
	return sliceutil.Dedup(agg.All())
}
