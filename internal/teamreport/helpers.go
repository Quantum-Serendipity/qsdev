package teamreport

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/posture"
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

// relativeTime formats a time.Time as a human-readable relative duration
// string such as "1h ago", "3d ago", "2m ago" (months), etc.
func relativeTime(t time.Time) string {
	now := time.Now().UTC()
	d := now.Sub(t)

	if d < 0 {
		return "in the future"
	}

	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d min ago", mins)
	case d < 24*time.Hour:
		hours := int(d.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	case d < 30*24*time.Hour:
		days := int(d.Hours() / 24)
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	default:
		months := int(d.Hours() / (24 * 30))
		if months == 0 {
			months = 1
		}
		if months == 1 {
			return "1mo ago"
		}
		return fmt.Sprintf("%dmo ago", months)
	}
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
