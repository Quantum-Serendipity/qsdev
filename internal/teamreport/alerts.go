package teamreport

import (
	"fmt"
	"sort"
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

	// HIGH: baseline conformance FAIL.
	if !p.Conformance.Baseline.Pass {
		alerts = append(alerts, PostureAlert{
			Project:  p.Name,
			Severity: SeverityHigh,
			Message:  "Baseline conformance check failed",
			Action:   "Run 'gdev status' and fix baseline conformance issues",
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
					Action:   "Investigate recent changes and run 'gdev status' for details",
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

	// MEDIUM: gdev version outdated (>2 minor versions behind).
	if isOutdatedGdev(p.GdevVersion, opts.GdevVersion) {
		alerts = append(alerts, PostureAlert{
			Project:  p.Name,
			Severity: SeverityMedium,
			Message:  fmt.Sprintf("gdev version %s is outdated (current: %s)", p.GdevVersion, opts.GdevVersion),
			Action:   "Update gdev to the latest version",
		})
	}

	// MEDIUM: stale scan (>7 days).
	if p.Stale {
		alerts = append(alerts, PostureAlert{
			Project:  p.Name,
			Severity: SeverityMedium,
			Message:  fmt.Sprintf("Last scan is stale (%s)", relativeTime(p.LastScan)),
			Action:   "Re-run 'gdev status --scan' to refresh posture data",
		})
	}

	return alerts
}
