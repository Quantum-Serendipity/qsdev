package check

import (
	"slices"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// CheckRequiredTools verifies that every always-on tool is recorded in
// tools.enabled and not in tools.disabled. A tool in neither list was dropped
// without the explicit `disable --force` opt-out.
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
