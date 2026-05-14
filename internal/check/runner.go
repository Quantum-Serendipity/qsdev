package check

import (
	"path/filepath"
	"time"
)

// RunAllChecks executes all check categories and builds a report.
func RunAllChecks(ctx CheckContext) *CheckReport {
	var results []CheckResult

	results = append(results, CheckBinaryCompatibility(ctx)...)
	results = append(results, CheckConfigIntegrity(ctx)...)
	results = append(results, CheckRequiredTools(ctx)...)
	results = append(results, CheckFileState(ctx)...)
	results = append(results, CheckSecurityHardening(ctx)...)
	results = append(results, CheckDenyRuleConflicts(ctx)...)

	return BuildReport(results, ctx.BinaryVersion, projectName(ctx))
}

// BuildReport constructs a CheckReport from results.
func BuildReport(results []CheckResult, binaryVersion, projectName string) *CheckReport {
	summary := CheckSummary{Total: len(results)}
	for _, r := range results {
		switch r.Status {
		case StatusPass:
			summary.Pass++
		case StatusFail:
			summary.Fail++
		case StatusWarn:
			summary.Warn++
		case StatusSkip:
			summary.Skip++
		}
	}

	return &CheckReport{
		Version:   binaryVersion,
		Project:   projectName,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Checks:    results,
		Summary:   summary,
	}
}

// ShouldFail returns true if any failed check meets or exceeds the severity
// threshold for the given audit level.
func ShouldFail(results []CheckResult, level AuditLevel) bool {
	if level == AuditLevelNone {
		return false
	}

	threshold := auditLevelRank(level)
	for _, r := range results {
		if r.Status == StatusFail && severityRank(r.Severity) >= threshold {
			return true
		}
	}
	return false
}

// FailCount returns the number of failed checks that meet or exceed the
// severity threshold for the given audit level.
func FailCount(results []CheckResult, level AuditLevel) int {
	if level == AuditLevelNone {
		return 0
	}

	threshold := auditLevelRank(level)
	count := 0
	for _, r := range results {
		if r.Status == StatusFail && severityRank(r.Severity) >= threshold {
			count++
		}
	}
	return count
}

func projectName(ctx CheckContext) string {
	return filepath.Base(ctx.ProjectRoot)
}
