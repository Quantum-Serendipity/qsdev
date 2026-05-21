package claudecode

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
	"github.com/Quantum-Serendipity/qsdev/internal/tmpl"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// SkillSummary describes a skill for inclusion in CLAUDE.md.
type SkillSummary struct {
	Name        string
	Description string
	Category    string
}

// AgentSummary describes a consulting agent for inclusion in CLAUDE.md.
type AgentSummary struct {
	Name        string
	Description string
}

// CommandSummary describes a qsdev CLI command for inclusion in CLAUDE.md.
type CommandSummary struct {
	Command     string
	Description string
}

// TaskSummary describes a devenv task for inclusion in CLAUDE.md.
type TaskSummary struct {
	Name     string
	Commands []string
}

// ClaudeMdTemplateData holds all data required to render the CLAUDE.md template.
type ClaudeMdTemplateData struct {
	ProjectName        string
	ProjectDescription string
	ArchitectureNotes  string
	Languages          []string
	BuildCommands      []string
	TestCommands       []string
	LintCommands       []string
	HasSecurityHooks   bool
	PackageManagers    []string

	PostmortemEnabled        bool
	VersionSentinelEnabled   bool
	VersionSentinelCovered   []string
	VersionSentinelUncovered []string
	SembleEnabled            bool
	SembleMode               string
	TrailOfBitsEnabled       bool

	AvailableSkills  []SkillSummary
	AvailableAgents  []AgentSummary
	GdevCommands     []CommandSummary
	DevenvTasks      []TaskSummary
	ModelSize        string
	HasGdevReference bool
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

		config := ecosystem.ToModuleConfig(lang)
		vc := mod.VerificationCommands(config)
		buildCmds = append(buildCmds, vc.Build...)
		testCmds = append(testCmds, vc.Test...)
		lintCmds = append(lintCmds, vc.Lint...)
	}

	data.BuildCommands = sliceutil.Dedup(buildCmds)
	data.TestCommands = sliceutil.Dedup(testCmds)
	data.LintCommands = sliceutil.Dedup(lintCmds)
	data.PackageManagers = sliceutil.Dedup(data.PackageManagers)

	// Agent tools metadata for CLAUDE.md.
	data.TrailOfBitsEnabled = sliceutil.Contains(answers.Skills, "security-review")
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
		data.ProjectDescription = "Development environment managed by " + branding.Get().AppName + "."
	}

	// Skills from qsdev-ops manifest.
	opsManifest, err := loadQsdevOpsManifest()
	if err != nil {
		slog.Warn("failed to load ops manifest", "error", err)
	}
	if opsManifest != nil {
		for _, s := range opsManifest.Skills {
			data.AvailableSkills = append(data.AvailableSkills, SkillSummary{
				Name: "/" + s.Name, Description: s.Description, Category: branding.Get().AppName + "-operations",
			})
		}
	}

	// Skills from consulting manifest.
	consultingManifest, err := loadConsultingSkillManifest()
	if err != nil {
		slog.Warn("failed to load consulting skill manifest", "error", err)
	}
	if consultingManifest != nil {
		for _, s := range consultingManifest.Skills {
			toolKey := "consulting-workflow-" + s.Name
			if answers.EnabledTools != nil && answers.EnabledTools[toolKey] {
				data.AvailableSkills = append(data.AvailableSkills, SkillSummary{
					Name: "/" + s.Name, Description: s.Description, Category: "consulting-workflows",
				})
			}
		}
	}

	// Agents from manifest.
	agentManifest, err := loadAgentManifest()
	if err != nil {
		slog.Warn("failed to load agent manifest", "error", err)
	}
	if agentManifest != nil {
		for _, a := range agentManifest.Agents {
			toolKey := "consulting-agent-" + a.Name
			if answers.EnabledTools != nil && answers.EnabledTools[toolKey] {
				data.AvailableAgents = append(data.AvailableAgents, AgentSummary{
					Name: "@" + a.Name, Description: a.Description,
				})
			}
		}
	}

	// Static qsdev commands.
	app := branding.Get().AppName
	data.GdevCommands = []CommandSummary{
		{Command: app + " init", Description: "Initialize or re-initialize project"},
		{Command: app + " devenv doctor", Description: "Check system and project health"},
		{Command: app + " devenv setup", Description: "Install missing prerequisites"},
		{Command: app + " enable <tool>", Description: "Enable a tool"},
		{Command: app + " disable <tool>", Description: "Disable a tool"},
		{Command: app + " status", Description: "Show configuration state"},
		{Command: app + " list", Description: "Show available tools"},
		{Command: app + " check", Description: "Validate configuration for CI"},
	}

	// Devenv tasks from verification commands.
	if len(buildCmds) > 0 {
		data.DevenvTasks = append(data.DevenvTasks, TaskSummary{Name: "build", Commands: sliceutil.Dedup(buildCmds)})
	}
	if len(testCmds) > 0 {
		data.DevenvTasks = append(data.DevenvTasks, TaskSummary{Name: "test", Commands: sliceutil.Dedup(testCmds)})
	}
	if len(lintCmds) > 0 {
		data.DevenvTasks = append(data.DevenvTasks, TaskSummary{Name: "lint", Commands: sliceutil.Dedup(lintCmds)})
	}

	// Model size for template rendering.
	data.ModelSize = ResolveModelSize(answers.ModelSize)
	if data.ModelSize == ModelSonnet && answers.ModelSize == "" {
		data.ModelSize = ModelOpus
	}
	data.HasGdevReference = true

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
