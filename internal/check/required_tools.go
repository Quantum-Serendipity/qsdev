package check

import (
	"maps"
	"os"
	"path/filepath"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/internal/surgery"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// MCPConfigRelPath is the project-relative path of the Claude Code MCP server
// configuration.
const MCPConfigRelPath = ".mcp.json"

// CheckRequiredTools verifies that every always-on tool is recorded in
// tools.enabled and not in tools.disabled. A tool in neither list was dropped
// without the explicit `disable --force` opt-out. It also verifies that the
// on-disk .mcp.json configures the server of every always-on MCP tool the
// project's expected generation configures (CheckContext.RequiredMCPServers),
// so a tool recorded as enabled is never one whose server is gone.
func CheckRequiredTools(ctx CheckContext) []CheckResult {
	if ctx.QsdevConfig == nil {
		return []CheckResult{
			{
				Category: CategoryRequiredTools,
				Name:     "required_tools",
				Status:   StatusSkip,
				Severity: SeverityInfo,
				Message:  configUnavailableMessage(ctx, "required tools check"),
			},
		}
	}

	if len(ctx.AlwaysOnToolNames) == 0 {
		return []CheckResult{
			{
				Category: CategoryRequiredTools,
				Name:     "required_tools",
				Status:   StatusSkip,
				Severity: SeverityInfo,
				Message:  "No always-on tools registered; skipping required tools check",
			},
		}
	}

	tools := ctx.QsdevConfig.Tools
	cfgFile := branding.Get().ConfigFile
	var results []CheckResult
	// Only always-on tools are required; opt-in and detected tools may be
	// legitimately listed in tools.disabled or absent from both lists.
	for _, toolName := range ctx.AlwaysOnToolNames {
		switch {
		case slices.Contains(tools.Disabled, toolName):
			results = append(results, CheckResult{
				Category:    CategoryRequiredTools,
				Name:        "tool_not_disabled_" + toolName,
				Status:      StatusFail,
				Severity:    SeverityHigh,
				Message:     "Required tool " + toolName + " is in the disabled list",
				Remediation: "Remove " + toolName + " from tools.disabled in " + cfgFile,
			})
		case !slices.Contains(tools.Enabled, toolName):
			results = append(results, CheckResult{
				Category: CategoryRequiredTools,
				Name:     "tool_missing_" + toolName,
				Status:   StatusFail,
				Severity: SeverityHigh,
				Message:  "Always-on tool " + toolName + " is in neither tools.enabled nor tools.disabled in " + cfgFile,
				Remediation: "Run `qsdev update` to restore " + toolName +
					", or `qsdev disable " + toolName + " --force` to opt out explicitly",
			})
		}
	}

	results = append(results, missingMCPServers(ctx)...)

	if len(results) == 0 {
		results = append(results, CheckResult{
			Category: CategoryRequiredTools,
			Name:     "required_tools",
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  "All always-on tools are enabled",
		})
	}

	return results
}

// missingMCPServers returns a failure for each always-on tool whose server
// ctx.RequiredMCPServers requires and the project's .mcp.json lacks.
func missingMCPServers(ctx CheckContext) []CheckResult {
	if len(ctx.RequiredMCPServers) == 0 {
		return nil
	}
	// A missing or unreadable file configures no server; each required one
	// is reported below.
	content, _ := os.ReadFile(filepath.Join(ctx.ProjectRoot, MCPConfigRelPath))
	var results []CheckResult
	for _, tool := range slices.Sorted(maps.Keys(ctx.RequiredMCPServers)) {
		server := ctx.RequiredMCPServers[tool]
		if surgery.JSONHasMCPServer(content, server) {
			continue
		}
		results = append(results, CheckResult{
			Category: CategoryRequiredTools,
			Name:     "tool_mcp_server_missing_" + tool,
			Status:   StatusFail,
			Severity: SeverityHigh,
			Message:  "Always-on tool " + tool + " is enabled but its MCP server " + server + " is missing from " + MCPConfigRelPath,
			Remediation: "Run `qsdev init --update` to restore the " + server + " server, or `qsdev disable " + tool +
				" --force` to opt out explicitly",
		})
	}
	return results
}
