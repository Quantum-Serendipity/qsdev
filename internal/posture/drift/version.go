package drift

import (
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

const categoryVersionDrift = "Version Drift"

// detectVersionDrift compares the qsdev version recorded in the generation state
// against the currently running qsdev version.
func detectVersionDrift(genState types.GeneratedState) Category {
	cat := Category{Name: categoryVersionDrift}
	app := branding.Get().AppName

	currentVersion := version.Info().Version

	if genState.QsdevVersion == "" {
		cat.Findings = append(cat.Findings, Finding{
			Category:    categoryVersionDrift,
			Severity:    Info,
			Subject:     app + " version",
			Description: "State was generated before version tracking was added",
			Actual:      currentVersion,
			Remediation: "Run " + app + " update to record the current version",
		})
		return cat
	}

	if genState.QsdevVersion != currentVersion {
		cat.Findings = append(cat.Findings, Finding{
			Category:    categoryVersionDrift,
			Severity:    Info,
			Subject:     app + " version",
			Description: "Configuration was generated with a different " + app + " version",
			Expected:    genState.QsdevVersion,
			Actual:      currentVersion,
			Remediation: "Run " + app + " update to regenerate configuration with the current version",
		})
	}

	return cat
}
