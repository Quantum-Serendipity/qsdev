package posture

// ShouldExitNonZero determines whether the posture assessment warrants a
// non-zero exit code, based on the configured audit level.
//
// Audit levels (from strictest to most permissive):
//   - "info" / "any": any findings of any kind
//   - "low": any vulnerabilities (Critical+High+Moderate+Low > 0)
//   - "moderate": Critical+High+Moderate > 0
//   - "high": Critical+High > 0 OR baseline conformance FAIL
//   - "critical": Critical > 0
//   - "none": always false (never exit non-zero)
//
// Independent of the level (except "none"), a dependency result that cannot be
// certified clean — a failed scan (ScanFailed) or any unresolved-severity
// vulnerability (Totals.Unknown > 0) — fails closed. Both are captured by the
// single Certifiable predicate, so the gate and every other consumer agree.
func ShouldExitNonZero(report *PostureReport, auditLevel string) bool {
	if auditLevel == "none" {
		return false
	}
	// A failed scan leaves vulnerability status unknown, and a vulnerability whose
	// severity could not be resolved could be anything up to critical. Neither can
	// be certified clean, so fail closed rather than certify on the strength of
	// zero Totals that only reflect a check that never completed conclusively.
	if !report.Dependencies.Certifiable() {
		return true
	}
	switch auditLevel {
	case "critical":
		return report.Dependencies.Totals.Critical > 0
	case "high":
		if report.Dependencies.Totals.Critical > 0 || report.Dependencies.Totals.High > 0 {
			return true
		}
		return !report.Conformance.Baseline.Pass
	case "moderate":
		return report.Dependencies.Totals.Critical > 0 ||
			report.Dependencies.Totals.High > 0 ||
			report.Dependencies.Totals.Moderate > 0
	case "low":
		totals := report.Dependencies.Totals
		return totals.Critical > 0 || totals.High > 0 ||
			totals.Moderate > 0 || totals.Low > 0
	case "info", "any":
		return hasAnyFindings(report)
	default:
		// Unknown levels default to "high" behavior.
		if report.Dependencies.Totals.Critical > 0 || report.Dependencies.Totals.High > 0 {
			return true
		}
		return !report.Conformance.Baseline.Pass
	}
}

// hasAnyFindings returns true if the report contains any findings at all:
// vulnerabilities, drift findings, failing conformance checks, or disabled layers.
func hasAnyFindings(report *PostureReport) bool {
	if report.Dependencies.Totals.Total() > 0 {
		return true
	}
	if report.Drift.TotalFindings > 0 {
		return true
	}
	if !report.Conformance.Baseline.Pass {
		return true
	}
	for _, l := range report.Defense.Layers {
		if l.Status == LayerDisabled || l.Status == LayerPartial {
			return true
		}
	}
	for _, f := range report.Config.Files {
		if f.State != "current" {
			return true
		}
	}
	return false
}
