package drift

import (
	"cmp"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// severityRank orders severities from most to least severe.
var severityRank = map[Severity]int{Critical: 0, Error: 1, Warning: 2, Info: 3}

// Severities lists every severity from most to least severe, for rendering
// severity summaries in a fixed order.
func Severities() []Severity {
	return []Severity{Critical, Error, Warning, Info}
}

// sortFindings orders findings deterministically: by subject, then by
// severity (most severe first), then by description.
func sortFindings(findings []Finding) {
	slices.SortStableFunc(findings, func(a, b Finding) int {
		return cmp.Or(
			cmp.Compare(a.Subject, b.Subject),
			cmp.Compare(severityRank[a.Severity], severityRank[b.Severity]),
			cmp.Compare(a.Description, b.Description),
		)
	})
}

// Detect runs all drift detection categories and aggregates the results
// into a single Report.
func Detect(projectDir string, genState types.GeneratedState, enabledTools map[string]bool) *Report {
	categories := []Category{
		detectFileModification(projectDir, genState),
		detectVersionDrift(genState),
		detectToolAvailability(enabledTools),
		detectMarkerIntegrity(projectDir, enabledTools),
		detectLockfileDrift(projectDir),
		detectHookDrift(projectDir, enabledTools),
	}

	report := &Report{
		Categories: categories,
		BySeverity: make(map[Severity]int),
	}

	for _, cat := range categories {
		// Several detectors range over maps, so their findings arrive in a
		// random order. Sort them so identical projects produce identical
		// reports (stable diffs, golden tests, SARIF result ordering).
		sortFindings(cat.Findings)
		report.TotalFindings += len(cat.Findings)
		for _, f := range cat.Findings {
			report.BySeverity[f.Severity]++
		}
	}

	return report
}
