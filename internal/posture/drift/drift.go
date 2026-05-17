package drift

import (
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

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
		report.TotalFindings += len(cat.Findings)
		for _, f := range cat.Findings {
			report.BySeverity[f.Severity]++
		}
	}

	return report
}
