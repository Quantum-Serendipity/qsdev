package posture

import (
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/version"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

const categoryVersionDrift = "Version Drift"

// detectVersionDrift compares the gdev version recorded in the generation state
// against the currently running gdev version.
func detectVersionDrift(genState types.GeneratedState) DriftCategory {
	cat := DriftCategory{Name: categoryVersionDrift}

	currentVersion := version.Info().Version

	if genState.GdevVersion == "" {
		cat.Findings = append(cat.Findings, DriftFinding{
			Category:    categoryVersionDrift,
			Severity:    DriftInfo,
			Subject:     "gdev version",
			Description: "State was generated before version tracking was added",
			Actual:      currentVersion,
			Remediation: "Run gdev update to record the current version",
		})
		return cat
	}

	if genState.GdevVersion != currentVersion {
		cat.Findings = append(cat.Findings, DriftFinding{
			Category:    categoryVersionDrift,
			Severity:    DriftInfo,
			Subject:     "gdev version",
			Description: "Configuration was generated with a different gdev version",
			Expected:    genState.GdevVersion,
			Actual:      currentVersion,
			Remediation: "Run gdev update to regenerate configuration with the current version",
		})
	}

	return cat
}
