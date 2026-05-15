package posture

import (
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

const categoryVersionDrift = "Version Drift"

// detectVersionDrift compares the qsdev version recorded in the generation state
// against the currently running qsdev version.
func detectVersionDrift(genState types.GeneratedState) DriftCategory {
	cat := DriftCategory{Name: categoryVersionDrift}

	currentVersion := version.Info().Version

	if genState.QsdevVersion == "" {
		cat.Findings = append(cat.Findings, DriftFinding{
			Category:    categoryVersionDrift,
			Severity:    DriftInfo,
			Subject:     "qsdev version",
			Description: "State was generated before version tracking was added",
			Actual:      currentVersion,
			Remediation: "Run qsdev update to record the current version",
		})
		return cat
	}

	if genState.QsdevVersion != currentVersion {
		cat.Findings = append(cat.Findings, DriftFinding{
			Category:    categoryVersionDrift,
			Severity:    DriftInfo,
			Subject:     "qsdev version",
			Description: "Configuration was generated with a different qsdev version",
			Expected:    genState.QsdevVersion,
			Actual:      currentVersion,
			Remediation: "Run qsdev update to regenerate configuration with the current version",
		})
	}

	return cat
}
