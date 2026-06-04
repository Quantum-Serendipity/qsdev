package devinit

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// buildPlanPreview generates a formatted preview of what files will be generated.
func buildPlanPreview(fs *formState) string {
	var b strings.Builder
	fmt.Fprintln(&b, "Will generate:")

	// Always include devenv files.
	fmt.Fprintln(&b, "  devenv.yaml           Nix inputs configuration")

	// Build the devenv.nix description from selected languages and services.
	devenvParts := buildDevenvParts(fs)
	fmt.Fprintf(&b, "  devenv.nix            %s\n", devenvParts)

	// .envrc when direnv is enabled.
	if fs.direnv {
		fmt.Fprintln(&b, "  .envrc                direnv auto-activation")
	}

	// Claude Code files when enabled.
	if fs.claudeCode {
		fmt.Fprintln(&b, "  CLAUDE.md             project documentation")
		fmt.Fprintf(&b, "  .claude/settings.json %s permissions\n", fs.permissionLevel)
		fmt.Fprintln(&b, "  .claude/rules/        security + language conventions")

		if len(fs.skills) > 0 {
			fmt.Fprintf(&b, "  .claude/skills/       %s\n", strings.Join(fs.skills, ", "))
		}

		if fs.autoFormat || fs.safetyBlock {
			var hookNames []string
			if fs.autoFormat {
				hookNames = append(hookNames, "auto-format")
			}
			if fs.safetyBlock {
				hookNames = append(hookNames, "safety-block")
			}
			fmt.Fprintf(&b, "  .claude/hooks/        %s\n", strings.Join(hookNames, ", "))
		}

		if len(fs.mcpServers) > 0 {
			fmt.Fprintf(&b, "  .mcp.json             %s\n", strings.Join(fs.mcpServers, ", "))
		}

		var agentTools []string
		if fs.agentPostmortem {
			agentTools = append(agentTools, "postmortem")
		}
		if fs.agentVersionSentinel {
			agentTools = append(agentTools, "version-sentinel")
		}
		if fs.agentSemble {
			agentTools = append(agentTools, "semble ("+fs.agentSembleMode+")")
		}
		if len(agentTools) > 0 {
			fmt.Fprintf(&b, "  AI agent tools        %s\n", strings.Join(agentTools, ", "))
		}
	}

	if fs.nixHardeningGuide {
		fmt.Fprintln(&b, "  nix-hardening.md    Nix security hardening guide")
	}

	return b.String()
}

// buildDetailedDefaults generates a human-readable summary of what the
// quick-path defaults will configure, so the user can review before accepting.
func buildDetailedDefaults(detected types.DetectedProject, defaults types.WizardAnswers) string {
	var b strings.Builder

	fmt.Fprintln(&b, "Languages:")
	if len(defaults.Languages) == 0 {
		fmt.Fprintln(&b, "  (none detected)")
	} else {
		for _, lang := range defaults.Languages {
			label := languageDisplayName(lang.Name)
			if lang.Version != "" {
				label += " " + lang.Version
			}
			if lang.PackageManager != "" {
				label += " (" + lang.PackageManager + ")"
			}
			fmt.Fprintf(&b, "  - %s\n", label)
		}
	}

	fmt.Fprintln(&b, "Services:")
	if len(defaults.Services) == 0 {
		fmt.Fprintln(&b, "  (none detected)")
	} else {
		for _, svc := range defaults.Services {
			fmt.Fprintf(&b, "  - %s\n", serviceLabel(svc.Name))
		}
	}

	fmt.Fprintln(&b, "Dev environment:")
	if defaults.Direnv {
		fmt.Fprintln(&b, "  direnv: enabled")
	} else {
		fmt.Fprintln(&b, "  direnv: disabled")
	}

	if defaults.ClaudeCode {
		fmt.Fprintln(&b, "  Claude Code: enabled (standard permissions)")
	} else {
		fmt.Fprintln(&b, "  Claude Code: disabled")
	}

	var tools []string
	if detected.HasGoMod || detected.HasPackageJSON || detected.HasPyProject ||
		detected.HasCargoToml || detected.HasCsproj {
		tools = append(tools, "version-sentinel")
	}
	tools = append(tools, "postmortem")
	fmt.Fprintf(&b, "  Agent tools: %s\n", strings.Join(tools, ", "))

	return b.String()
}

// buildDevenvParts constructs a description of the devenv.nix contents from formState.
func buildDevenvParts(fs *formState) string {
	var parts []string

	// On quick path, selectedLanguages might be pre-populated.
	for _, lang := range fs.selectedLanguages {
		parts = append(parts, languageLabel(lang))
	}

	for _, svc := range fs.selectedServices {
		parts = append(parts, serviceLabel(svc))
	}

	if len(parts) == 0 {
		return "development environment"
	}
	return strings.Join(parts, ", ")
}
