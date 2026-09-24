package devinit

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// buildPlanPreview generates a formatted preview of what files will be
// generated for the current form state. It renders the answers mapFormToAnswers
// produces, so the preview matches what is generated on the quick path as well
// as the customize path.
func buildPlanPreview(fs *formState) string {
	answers := mapFormToAnswers(fs, fs.partial.ProjectRoot, fs.partial.ProjectName, fs.partial.Detected)
	return renderPlanPreview(answers)
}

// renderPlanPreview lists the files the given answers generate.
func renderPlanPreview(a types.WizardAnswers) string {
	var b strings.Builder
	fmt.Fprintln(&b, "Will generate:")

	// Always include devenv files.
	fmt.Fprintln(&b, "  devenv.yaml           Nix inputs configuration")
	fmt.Fprintf(&b, "  devenv.nix            %s\n", devenvSummary(a))

	if a.Direnv {
		fmt.Fprintln(&b, "  .envrc                direnv auto-activation")
	}

	if a.ClaudeCode {
		fmt.Fprintln(&b, "  CLAUDE.md             project documentation")
		fmt.Fprintf(&b, "  .claude/settings.json %s\n", permissionSummary(a))
		fmt.Fprintln(&b, "  .claude/rules/        security + language conventions")
		if len(a.Skills) > 0 {
			fmt.Fprintf(&b, "  .claude/skills/       %s\n", strings.Join(a.Skills, ", "))
		}
		if hooks := hookNames(a); len(hooks) > 0 {
			fmt.Fprintf(&b, "  .claude/hooks/        %s\n", strings.Join(hooks, ", "))
		}
		if len(a.MCPServers) > 0 {
			fmt.Fprintf(&b, "  .mcp.json             %s\n", strings.Join(a.MCPServers, ", "))
		}
		if tools := agentToolNames(a.AgentTools); len(tools) > 0 {
			fmt.Fprintf(&b, "  AI agent tools        %s\n", strings.Join(tools, ", "))
		}
	}

	if a.NixHardeningGuide {
		fmt.Fprintln(&b, "  nix-hardening.md    Nix security hardening guide")
	}

	return b.String()
}

// buildDetailedDefaults generates a human-readable summary of what the
// quick-path defaults will configure, so the user can review before accepting.
// defaults must be the answers the quick path actually produces.
func buildDetailedDefaults(defaults types.WizardAnswers) string {
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
	fmt.Fprintf(&b, "  direnv: %s\n", enabledWord(defaults.Direnv))

	if !defaults.ClaudeCode {
		fmt.Fprintln(&b, "  Claude Code: disabled")
		return b.String()
	}
	fmt.Fprintf(&b, "  Claude Code: enabled (%s)\n", permissionSummary(defaults))
	fmt.Fprintf(&b, "  Hooks: %s\n", joinOrNone(hookNames(defaults)))
	fmt.Fprintf(&b, "  MCP servers: %s\n", joinOrNone(defaults.MCPServers))
	fmt.Fprintf(&b, "  Agent tools: %s\n", joinOrNone(agentToolNames(defaults.AgentTools)))

	return b.String()
}

// devenvSummary describes the devenv.nix contents from the selected
// languages and services.
func devenvSummary(a types.WizardAnswers) string {
	var parts []string
	for _, lang := range a.Languages {
		parts = append(parts, languageDisplayName(lang.Name))
	}
	for _, svc := range a.Services {
		parts = append(parts, serviceLabel(svc.Name))
	}
	if len(parts) == 0 {
		return "development environment"
	}
	return strings.Join(parts, ", ")
}

// permissionSummary describes the permission preset settings.json will use,
// resolved exactly as settings generation resolves it.
func permissionSummary(a types.WizardAnswers) string {
	return effectivePermissionPreset(a, nil) + " permissions"
}

// hookNames lists the enabled Claude Code hooks by their preset names, marking
// each one that has no policy to enforce.
func hookNames(a types.WizardAnswers) []string {
	h := a.Hooks
	candidates := []struct {
		name    string
		enabled bool
	}{
		{"self-protection", h.SelfProtection},
		{"auto-format", h.AutoFormat},
		{"safety-block", h.SafetyBlock},
		{"pre-commit", h.PreCommit},
		{"audit-log", h.AuditLog},
		{"credential-scan", h.CredentialScan},
		{"destructive-prevention", h.DestructivePrevention},
		{"soc2-audit", h.SOC2Audit},
		{"file-boundary", h.FileBoundary},
		{"tool-gates", h.ToolGates},
	}
	unenforced := make(map[string]bool)
	for _, u := range claudecode.HooksWithoutPolicy(a) {
		unenforced[u.Name] = true
	}
	var names []string
	for _, c := range candidates {
		switch {
		case !c.enabled:
		case unenforced[c.name]:
			names = append(names, c.name+" (no policy)")
		default:
			names = append(names, c.name)
		}
	}
	return names
}

// agentToolNames lists the enabled AI agent tools.
func agentToolNames(t types.AgentToolsAnswers) []string {
	var names []string
	if t.PostmortemEnabled {
		names = append(names, "postmortem")
	}
	if t.VersionSentinel {
		names = append(names, "version-sentinel")
	}
	if t.SembleEnabled {
		names = append(names, "semble ("+t.SembleMode+")")
	}
	return names
}

func enabledWord(on bool) string {
	if on {
		return "enabled"
	}
	return "disabled"
}

func joinOrNone(items []string) string {
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}
