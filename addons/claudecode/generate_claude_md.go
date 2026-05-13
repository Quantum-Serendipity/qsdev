package claudecode

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/tmpl"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// ClaudeMdTemplateData holds all data required to render the CLAUDE.md template.
type ClaudeMdTemplateData struct {
	ProjectName        string
	ProjectDescription string
	ArchitectureNotes  string
	Languages          []string // Display names: "Go", "JavaScript/TypeScript", etc.
	BuildCommands      []string
	TestCommands       []string
	LintCommands       []string
	HasSecurityHooks   bool
	PackageManagers    []string // "npm", "pip", "cargo" for security section specifics

	PostmortemEnabled      bool
	VersionSentinelEnabled bool
	VersionSentinelCovered []string
	VersionSentinelUncovered []string
	SembleEnabled          bool
	SembleMode             string
	TrailOfBitsEnabled     bool
}

// BuildClaudeMdData assembles all template data from wizard answers and ecosystem
// modules. It maps language choices to display names, derives commands from
// ecosystem module VerificationCommands, and collects package manager metadata.
func BuildClaudeMdData(answers types.WizardAnswers, registry *ecosystem.Registry) *ClaudeMdTemplateData {
	data := &ClaudeMdTemplateData{
		ProjectName:      answers.ProjectName,
		HasSecurityHooks: answers.Hooks.SafetyBlock,
	}

	var buildCmds, testCmds, lintCmds []string

	for _, lang := range answers.Languages {
		mod, ok := registry.ByName(lang.Name)
		if !ok {
			continue
		}
		data.Languages = append(data.Languages, mod.DisplayName())
		for _, pm := range mod.PackageManagers() {
			data.PackageManagers = append(data.PackageManagers, pm.Name)
		}

		config := toModuleConfig(lang)
		vc := mod.VerificationCommands(config)
		buildCmds = append(buildCmds, vc.Build...)
		testCmds = append(testCmds, vc.Test...)
		lintCmds = append(lintCmds, vc.Lint...)
	}

	data.BuildCommands = dedup(buildCmds)
	data.TestCommands = dedup(testCmds)
	data.LintCommands = dedup(lintCmds)
	data.PackageManagers = dedup(data.PackageManagers)

	// Agent tools metadata for CLAUDE.md.
	data.TrailOfBitsEnabled = contains(answers.Skills, "security-review")
	data.PostmortemEnabled = answers.AgentTools.PostmortemEnabled
	data.SembleEnabled = answers.AgentTools.SembleEnabled
	data.SembleMode = answers.AgentTools.SembleMode
	if answers.AgentTools.VersionSentinel {
		data.VersionSentinelEnabled = true
		report := collectManifestCoverage(answers, registry)
		for _, m := range report.Covered {
			data.VersionSentinelCovered = append(data.VersionSentinelCovered, m.Path)
		}
		for _, m := range report.Uncovered {
			data.VersionSentinelUncovered = append(data.VersionSentinelUncovered, m.Path)
		}
	}

	// Generate a default description if none provided.
	if data.ProjectName != "" && len(data.Languages) > 0 {
		data.ProjectDescription = fmt.Sprintf("%s — a %s project.", data.ProjectName, strings.Join(data.Languages, ", "))
	} else if data.ProjectName != "" {
		data.ProjectDescription = fmt.Sprintf("%s development environment.", data.ProjectName)
	} else {
		data.ProjectDescription = "Development environment managed by gdev."
	}

	return data
}

// GenerateClaudeMd produces a GeneratedFile containing the rendered CLAUDE.md
// from wizard answers and ecosystem module registry.
func GenerateClaudeMd(answers types.WizardAnswers, registry *ecosystem.Registry) (*types.GeneratedFile, error) {
	data := BuildClaudeMdData(answers, registry)

	renderer, err := tmpl.NewMarkdownRenderer(templateFS, "templates")
	if err != nil {
		return nil, fmt.Errorf("creating Markdown renderer: %w", err)
	}

	content, err := renderer.Render("claude-md", data)
	if err != nil {
		return nil, fmt.Errorf("rendering CLAUDE.md template: %w", err)
	}

	return &types.GeneratedFile{
		Path:     "CLAUDE.md",
		Content:  content,
		Mode:     0o644,
		Strategy: types.SectionMarker,
	}, nil
}
