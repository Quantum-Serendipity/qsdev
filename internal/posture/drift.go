package posture

import (
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// DetectDrift runs all drift detection categories and aggregates the results
// into a single DriftReport.
func DetectDrift(projectDir string, genState types.GeneratedState, enabledTools map[string]bool) *DriftReport {
	categories := []DriftCategory{
		detectFileModification(projectDir, genState),
		detectVersionDrift(genState),
		detectToolAvailability(enabledTools),
		detectMarkerIntegrity(projectDir, enabledTools),
		detectLockfileDrift(projectDir),
		detectHookDrift(projectDir, enabledTools),
	}

	report := &DriftReport{
		Categories: categories,
		BySeverity: make(map[DriftSeverity]int),
	}

	for _, cat := range categories {
		report.TotalFindings += len(cat.Findings)
		for _, f := range cat.Findings {
			report.BySeverity[f.Severity]++
		}
	}

	return report
}
