package posture

import (
	"errors"
	"fmt"
	"strings"
)

// ErrUnknownAuditLevel is returned by ParseAuditLevel for a level it does not
// recognize.
var ErrUnknownAuditLevel = errors.New("unknown audit level")

// auditLevels lists the canonical audit levels, most permissive first.
var auditLevels = []string{"none", "critical", "high", "moderate", "low", "info"}

// auditLevelAliases maps accepted spellings onto canonical levels. "medium" is
// the name `check --audit-level` uses for the same threshold, and "any" is a
// long-standing synonym for "info".
var auditLevelAliases = map[string]string{
	"medium": "moderate",
	"any":    "info",
}

// ParseAuditLevel validates an audit level and returns its canonical form. It
// accepts the canonical levels and their aliases in any case, and rejects
// everything else so a typo cannot silently select a different threshold.
func ParseAuditLevel(level string) (string, error) {
	l := strings.ToLower(strings.TrimSpace(level))
	if canonical, ok := auditLevelAliases[l]; ok {
		return canonical, nil
	}
	for _, known := range auditLevels {
		if l == known {
			return known, nil
		}
	}
	return "", fmt.Errorf("%w %q: must be one of %s (or medium, any)",
		ErrUnknownAuditLevel, level, strings.Join(auditLevels, ", "))
}

// ShouldExitNonZero determines whether the posture assessment warrants a
// non-zero exit code, based on the configured audit level.
//
// Audit levels (from strictest to most permissive):
//   - "info" / "any": any findings of any kind
//   - "low": any vulnerabilities (Critical+High+Moderate+Low > 0) OR "high"
//   - "moderate": Critical+High+Moderate > 0 OR "high"
//   - "high": Critical+High > 0 OR baseline or custom conformance FAIL
//   - "critical": Critical > 0
//   - "none": always false (never exit non-zero)
//
// Independent of the level (except "none"), a dependency result that cannot be
// certified clean — a failed scan (ScanFailed) or any unresolved-severity
// vulnerability (Totals.Unknown > 0) — fails closed. Both are captured by the
// single Certifiable predicate, so the gate and every other consumer agree.
//
// Levels are resolved with ParseAuditLevel, so aliases such as "medium" select
// the same threshold as their canonical level. A level that cannot be resolved
// fails closed: a gate that cannot tell what it is enforcing must not pass.
// Callers should validate user input with ParseAuditLevel up front to report
// the mistake instead.
func ShouldExitNonZero(report *PostureReport, auditLevel string) bool {
	level, err := ParseAuditLevel(auditLevel)
	if err != nil {
		return true
	}
	if level == "none" {
		return false
	}
	// A failed scan leaves vulnerability status unknown, and a vulnerability whose
	// severity could not be resolved could be anything up to critical. Neither can
	// be certified clean, so fail closed rather than certify on the strength of
	// zero Totals that only reflect a check that never completed conclusively.
	if !report.Dependencies.Certifiable() {
		return true
	}
	switch level {
	case "critical":
		return report.Dependencies.Totals.Critical > 0
	case "high":
		if report.Dependencies.Totals.Critical > 0 || report.Dependencies.Totals.High > 0 {
			return true
		}
		return conformanceFails(report)
	case "moderate":
		return report.Dependencies.Totals.Critical > 0 ||
			report.Dependencies.Totals.High > 0 ||
			report.Dependencies.Totals.Moderate > 0 ||
			conformanceFails(report)
	case "low":
		totals := report.Dependencies.Totals
		return totals.Critical > 0 || totals.High > 0 ||
			totals.Moderate > 0 || totals.Low > 0 ||
			conformanceFails(report)
	default: // "info"
		return hasAnyFindings(report)
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
	if conformanceFails(report) {
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

// conformanceFails reports whether baseline conformance or the project's own
// conformance policy (.qsdev-policy.yaml, evaluated into Conformance.Custom)
// failed. A project without a custom policy has no custom level to fail. It
// gates "high" and, because each level includes every check of the more
// permissive levels, "moderate", "low" and "info" too.
//
// An unknown baseline (dependencies not scanned) is not a failure: it is
// reported as unknown, never as a pass, and the vulnerability gate it cannot
// evaluate is flagged by the caller (status warns that --scan is needed).
func conformanceFails(report *PostureReport) bool {
	if report.Conformance.Baseline.Verdict() == CheckFail {
		return true
	}
	return report.Conformance.Custom != nil && report.Conformance.Custom.Verdict() == CheckFail
}
