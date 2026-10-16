package check

import (
	"maps"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
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
// ctx.RequiredMCPServers requires and the project's .mcp.json lacks, or
// configures as a definition ctx.TrustedMCP does not vouch for.
func missingMCPServers(ctx CheckContext) []CheckResult {
	if len(ctx.RequiredMCPServers) == 0 {
		return nil
	}
	// A missing, unreadable or unparseable file (mcpconfig also refuses keys
	// that Claude Code and qsdev would read differently) configures no
	// server; each required one is reported below.
	servers, readErr := mcpregistry.ConfiguredServers(ctx.ProjectRoot, nil)
	configured := make(map[string]mcphealth.ServerConfig, len(servers))
	for _, s := range servers {
		configured[s.Name] = s
	}
	var results []CheckResult
	for _, tool := range slices.Sorted(maps.Keys(ctx.RequiredMCPServers)) {
		server := ctx.RequiredMCPServers[tool]
		cfg, ok := configured[server]
		if ok {
			if r, bad := untrustedMCPServer(ctx, tool, cfg); bad {
				results = append(results, r)
			}
			continue
		}
		results = append(results, missingMCPServer(tool, server, readErr))
	}
	return results
}

// missingMCPServer returns the failure for tool's required server, absent
// from .mcp.json or, when readErr is set, in a file that could not be read.
// `qsdev init --update` merges onto the file and keeps what it cannot read,
// so an unreadable file is restored by overwriting it.
func missingMCPServer(tool, server string, readErr error) CheckResult {
	r := CheckResult{
		Category: CategoryRequiredTools,
		Name:     "tool_mcp_server_missing_" + tool,
		Status:   StatusFail,
		Severity: SeverityHigh,
		Message:  "Always-on tool " + tool + " is enabled but its MCP server " + server + " is missing from " + MCPConfigRelPath,
		Remediation: "Run `qsdev init --update` to restore the " + server + " server, or `qsdev disable " + tool +
			" --force` to opt out explicitly",
	}
	if readErr != nil {
		r.Message = "Always-on tool " + tool + " is enabled but its MCP server " + server + " cannot be read from " +
			MCPConfigRelPath + ": " + readErr.Error()
		r.Remediation = restoreMCPConfigRemediation(tool)
	}
	return r
}

// restoreMCPConfigRemediation is the remediation for a required server whose
// .mcp.json entry must be replaced by the generated one.
func restoreMCPConfigRemediation(tool string) string {
	return "Run `qsdev update --configs-only --overwrite-modified` to restore the generated " + MCPConfigRelPath +
		" (this discards other edits to managed files), or `qsdev disable " + tool + " --force` to opt out explicitly"
}

// untrustedMCPServer returns the failure for tool's required server when its
// configured entry cfg is not a definition ctx.TrustedMCP vouches for, and
// reports whether there is one. With ctx.TrustedMCP unknown there is none.
func untrustedMCPServer(ctx CheckContext, tool string, cfg mcphealth.ServerConfig) (CheckResult, bool) {
	if ctx.TrustedMCP == nil {
		return CheckResult{}, false
	}
	diff := mcpregistry.TrustMismatch(cfg, ctx.TrustedMCP[cfg.Name])
	if diff == "" {
		return CheckResult{}, false
	}
	return CheckResult{
		Category: CategoryRequiredTools,
		Name:     "tool_mcp_server_untrusted_" + tool,
		Status:   StatusFail,
		Severity: SeverityHigh,
		Message: "Always-on tool " + tool + " is enabled but its MCP server " + cfg.Name + " in " + MCPConfigRelPath +
			" is not a definition qsdev trusts: " + diff,
		Remediation: restoreMCPConfigRemediation(tool),
	}, true
}
