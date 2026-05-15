package check

// CheckRequiredTools verifies that always-on tools are not in the disabled list.
func CheckRequiredTools(ctx CheckContext) []CheckResult {
	if ctx.QsdevConfig == nil {
		return []CheckResult{
			{
				Category: CategoryRequiredTools,
				Name:     "required_tools",
				Status:   StatusSkip,
				Severity: SeverityInfo,
				Message:  "No .qsdev.yaml found; skipping required tools check",
			},
		}
	}

	if len(ctx.ToolNames) == 0 {
		return []CheckResult{
			{
				Category: CategoryRequiredTools,
				Name:     "required_tools",
				Status:   StatusSkip,
				Severity: SeverityInfo,
				Message:  "No tools registered; skipping required tools check",
			},
		}
	}

	disabled := make(map[string]bool, len(ctx.QsdevConfig.Tools.Disabled))
	for _, t := range ctx.QsdevConfig.Tools.Disabled {
		disabled[t] = true
	}

	var results []CheckResult
	for _, toolName := range ctx.ToolNames {
		if disabled[toolName] {
			results = append(results, CheckResult{
				Category:    CategoryRequiredTools,
				Name:        "tool_not_disabled_" + toolName,
				Status:      StatusFail,
				Severity:    SeverityHigh,
				Message:     "Required tool " + toolName + " is in the disabled list",
				Remediation: "Remove " + toolName + " from tools.disabled in .qsdev.yaml",
			})
		}
	}

	if len(results) == 0 {
		results = append(results, CheckResult{
			Category: CategoryRequiredTools,
			Name:     "required_tools",
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  "No required tools are disabled",
		})
	}

	return results
}
