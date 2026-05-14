package teamreport

import (
	"time"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/posture"
)

// Test helpers shared across test files.

func projectSummaryHelper(name string, score float64, baselinePass, enhancedPass bool, critVulns, highVulns int, gdevVersion string, lastScan time.Time) ProjectSummary {
	return ProjectSummary{
		Name:        name,
		Score:       makeScore(score),
		Conformance: makeConformance(baselinePass, enhancedPass),
		VulnTotals:  makeVulns(critVulns, highVulns),
		GdevVersion: gdevVersion,
		LastScan:    lastScan,
	}
}

func makeScore(total float64) posture.AggregateScore {
	return posture.AggregateScore{
		Total:     total,
		Grade:     posture.ScoreToGrade(total),
		Defense:   total,
		Config:    total,
		DepHealth: total,
	}
}

func makeConformance(baselinePass, enhancedPass bool) posture.ConformanceResult {
	return posture.ConformanceResult{
		Baseline: posture.ConformanceLevel{
			Pass:   baselinePass,
			Checks: []posture.ConformanceCheck{},
		},
		Enhanced: posture.ConformanceLevel{
			Pass:   enhancedPass,
			Checks: []posture.ConformanceCheck{},
		},
	}
}

func makeVulns(critical, high int) posture.VulnSeverityCounts {
	return posture.VulnSeverityCounts{
		Critical: critical,
		High:     high,
	}
}
