package teamreport

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
	"github.com/Quantum-Serendipity/qsdev/internal/timeutil"
)

// medianFloat64 returns the median of a sorted slice of float64 values.
// The input slice MUST be sorted in ascending order.
// Returns 0 for an empty slice.
func medianFloat64(sorted []float64) float64 {
	n := len(sorted)
	if n == 0 {
		return 0
	}
	if n%2 == 1 {
		return sorted[n/2]
	}
	return (sorted[n/2-1] + sorted[n/2]) / 2.0
}

// relativeTime formats a time.Time as a compact human-readable relative
// duration string such as "1h ago", "3d ago", "1mo ago".
func relativeTime(t time.Time) string {
	return timeutil.RelativeTimeShort(t)
}

// lastScanText renders when a project's dependencies were last scanned,
// or "never" when they have not been scanned.
func lastScanText(p ProjectSummary) string {
	if p.LastScan == nil {
		return "never"
	}
	return relativeTime(*p.LastScan)
}

// depHealthText renders a dependency health sub-score, or "n/a (not scanned)"
// when it is nil because the project's dependencies were never scanned.
func depHealthText(score *float64) string {
	if score == nil {
		return "n/a (not scanned)"
	}
	return fmt.Sprintf("%.1f", *score)
}

// vulnCountsText renders a project's critical/high counts, or "n/a" when
// its dependencies were not scanned and the counts are unknown.
func vulnCountsText(p ProjectSummary) string {
	if !p.Scanned {
		return "n/a"
	}
	return fmt.Sprintf("%d/%d", p.VulnTotals.Critical, p.VulnTotals.High)
}

// scoreToGrade delegates to posture.ScoreToGrade for consistent grading.
func scoreToGrade(score float64) string {
	return posture.ScoreToGrade(score)
}

// sortProjectsByScoreDesc sorts a slice of ProjectSummary by score descending.
func sortProjectsByScoreDesc(projects []ProjectSummary) {
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Score.Total > projects[j].Score.Total
	})
}

// roundTo1 rounds a float64 to one decimal place.
func roundTo1(v float64) float64 {
	return math.Round(v*10) / 10
}
