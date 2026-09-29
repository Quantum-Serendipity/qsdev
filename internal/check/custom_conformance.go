package check

import "fmt"

// CheckCustomConformance reports each requirement of the project's custom
// conformance policy as a check. A failing requirement is high severity, so
// it fails `qsdev check` at the default audit level, the same threshold at
// which `qsdev status` gates on conformance. A project without a custom
// policy yields no results.
func CheckCustomConformance(ctx CheckContext) []CheckResult {
	custom := ctx.CustomConformance
	if custom == nil {
		return nil
	}
	results := make([]CheckResult, 0, len(custom.Requirements))
	for _, req := range custom.Requirements {
		r := CheckResult{
			Category: CategoryCustomConformance,
			Name:     req.Name,
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  req.Reason,
			FilePath: custom.PolicyFile,
		}
		if !req.Pass {
			r.Status = StatusFail
			r.Severity = SeverityHigh
			r.Remediation = fmt.Sprintf("Bring the project in line with requirement %q, or change it in %s",
				req.Name, custom.PolicyFile)
		}
		results = append(results, r)
	}
	return results
}
