package claudecode

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/tmpl"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
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

// CommandSummary describes a gdev CLI command for inclusion in CLAUDE.md.
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

	// Skills from gdev-ops manifest.
	opsManifest, _ := loadGdevOpsManifest()
	if opsManifest != nil {
		for _, s := range opsManifest.Skills {
			if answers.EnabledTools == nil || answers.EnabledTools[s.Name] {
				data.AvailableSkills = append(data.AvailableSkills, SkillSummary{
					Name: "/" + s.Name, Description: s.Description, Category: "gdev-operations",
				})
			}
		}
	}

	// Skills from consulting manifest.
	consultingManifest, _ := loadConsultingSkillManifest()
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
	agentManifest, _ := loadAgentManifest()
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

	// Static gdev commands.
	data.GdevCommands = []CommandSummary{
		{Command: "gdev init", Description: "Initialize or re-initialize project"},
		{Command: "gdev devenv doctor", Description: "Check system and project health"},
		{Command: "gdev devenv setup", Description: "Install missing prerequisites"},
		{Command: "gdev enable <tool>", Description: "Enable a tool"},
		{Command: "gdev disable <tool>", Description: "Disable a tool"},
		{Command: "gdev status", Description: "Show configuration state"},
		{Command: "gdev list", Description: "Show available tools"},
		{Command: "gdev check", Description: "Validate configuration for CI"},
	}

	// Devenv tasks from verification commands.
	if len(buildCmds) > 0 {
		data.DevenvTasks = append(data.DevenvTasks, TaskSummary{Name: "build", Commands: dedup(buildCmds)})
	}
	if len(testCmds) > 0 {
		data.DevenvTasks = append(data.DevenvTasks, TaskSummary{Name: "test", Commands: dedup(testCmds)})
	}
	if len(lintCmds) > 0 {
		data.DevenvTasks = append(data.DevenvTasks, TaskSummary{Name: "lint", Commands: dedup(lintCmds)})
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
