package teamreport

import (
	"fmt"
	"sort"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// Alert severity constants.
const (
	SeverityCritical = "critical"
	SeverityHigh     = "high"
	SeverityMedium   = "medium"
)

// severityRank returns a numeric rank for sorting alerts by severity.
// Lower rank = higher severity.
func severityRank(sev string) int {
	switch sev {
	case SeverityCritical:
		return 0
	case SeverityHigh:
		return 1
	case SeverityMedium:
		return 2
	default:
		return 3
	}
}

// generateAlerts produces alerts for projects that need attention.
// Alerts are sorted by severity (critical first), then by score ascending
// within each severity level.
func generateAlerts(projects []ProjectSummary, opts AggregateOptions, history *HistoryStore) []PostureAlert {
	var alerts []PostureAlert

	for _, p := range projects {
		alerts = append(alerts, alertsForProject(p, opts, history)...)
	}

	// Sort: severity descending (critical first), then score ascending.
	sort.SliceStable(alerts, func(i, j int) bool {
		ri := severityRank(alerts[i].Severity)
		rj := severityRank(alerts[j].Severity)
		if ri != rj {
			return ri < rj
		}
		// Within same severity, sort by project name for determinism.
		return alerts[i].Project < alerts[j].Project
	})

	return alerts
}

// alertsForProject generates alerts for a single project.
func alertsForProject(p ProjectSummary, opts AggregateOptions, history *HistoryStore) []PostureAlert {
	var alerts []PostureAlert

	// CRITICAL: project has critical vulnerabilities.
	if p.VulnTotals.Critical > 0 {
		alerts = append(alerts, PostureAlert{
			Project:  p.Name,
			Severity: SeverityCritical,
			Message:  fmt.Sprintf("%d critical vulnerabilities found", p.VulnTotals.Critical),
			Action:   "Run vulnerability scan and apply patches immediately",
		})
	}

	// HIGH: dependency scan inconclusive (failed ecosystem scan or unresolved
	// severities). A member whose scan cannot be certified clean must not read
	// clean fleet-wide on the strength of zero VulnTotals.
	if !p.Certifiable {
		alerts = append(alerts, PostureAlert{
			Project:  p.Name,
			Severity: SeverityHigh,
			Message:  "Dependency scan inconclusive: failed or unresolved-severity vulnerabilities (not confirmed clean)",
			Action:   "Re-run 'qsdev status --scan' and resolve failed or unresolved-severity findings",
		})
	} else if !p.Scanned {
		// HIGH: dependencies never scanned. Zero VulnTotals then mean
		// "unknown", so the project must not read clean fleet-wide.
		alerts = append(alerts, PostureAlert{
			Project:  p.Name,
			Severity: SeverityHigh,
			Message:  "Dependencies not scanned for vulnerabilities (vulnerability status unknown)",
			Action:   "Generate the posture report with 'qsdev status --scan --json'",
		})
	}

	// HIGH: baseline conformance FAIL. An unknown baseline (dependencies not
	// scanned) is covered by the not-scanned alert above.
	if p.Conformance.Baseline.Verdict() == posture.CheckFail {
		alerts = append(alerts, PostureAlert{
			Project:  p.Name,
			Severity: SeverityHigh,
			Message:  "Baseline conformance check failed",
			Action:   "Run 'qsdev status' and fix baseline conformance issues",
		})
	}

	// MEDIUM: score below the team threshold.
	if opts.Threshold > 0 && p.Score.Total < opts.Threshold {
		alerts = append(alerts, PostureAlert{
			Project:  p.Name,
			Severity: SeverityMedium,
			Message:  fmt.Sprintf("Score %.1f is below the team threshold of %.1f", p.Score.Total, opts.Threshold),
			Action:   "Run 'qsdev status' and address the lowest-scoring posture areas",
		})
	}

	// Score drop alerts (require history).
	if history != nil {
		prevScore, ok := history.PreviousScore(p.Name)
		if ok {
			delta := prevScore - p.Score.Total

			// HIGH: score drop > 10 points.
			if delta > 10 {
				alerts = append(alerts, PostureAlert{
					Project:  p.Name,
					Severity: SeverityHigh,
					Message:  fmt.Sprintf("Score dropped %.1f points (%.1f -> %.1f)", delta, prevScore, p.Score.Total),
					Action:   "Investigate recent changes and run 'qsdev status' for details",
				})
			} else if delta > 5 {
				// MEDIUM: score drop > 5 points.
				alerts = append(alerts, PostureAlert{
					Project:  p.Name,
					Severity: SeverityMedium,
					Message:  fmt.Sprintf("Score dropped %.1f points (%.1f -> %.1f)", delta, prevScore, p.Score.Total),
					Action:   "Review recent changes that may have affected security posture",
				})
			}
		}
	}

	// MEDIUM: qsdev version outdated (>2 minor versions behind).
	if isOutdatedGdev(p.QsdevVersion, opts.QsdevVersion) {
		alerts = append(alerts, PostureAlert{
			Project:  p.Name,
			Severity: SeverityMedium,
			Message:  fmt.Sprintf("%s version %s is outdated (current: %s)", branding.Get().AppName, p.QsdevVersion, opts.QsdevVersion),
			Action:   "Update " + branding.Get().AppName + " to the latest version",
		})
	}

	// MEDIUM: stale scan (>7 days).
	if p.Stale {
		alerts = append(alerts, PostureAlert{
			Project:  p.Name,
			Severity: SeverityMedium,
			Message:  fmt.Sprintf("Last scan is stale (%s)", lastScanText(p)),
			Action:   "Re-run 'qsdev status --scan' to refresh posture data",
		})
	}

	return alerts
}
