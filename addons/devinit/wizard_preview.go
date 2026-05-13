package devinit

import (
	"fmt"
	"strings"
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
