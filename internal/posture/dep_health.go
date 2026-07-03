package posture

import "math"

const (
	DeductCritical    = 25.0
	DeductHigh        = 10.0
	DeductModerate    = 3.0
	DeductLow         = 1.0
	DeductMissingLock = 15.0
	// DeductUnknown penalizes a vulnerability whose severity could not be
	// resolved. Its true severity could be anything up to critical, so it is
	// deducted at the High rate — enough to visibly drop the score below a clean
	// 100 without overstating it as a confirmed critical.
	DeductUnknown = 10.0
)

// ComputeDepScore calculates dependency health score (0-100).
// Starts at 100, deducts per vulnerability and per missing lock file. Floor at 0.
func ComputeDepScore(ecosystems []EcosystemStatus) DependencyHealth {
	var totals VulnSeverityCounts
	score := 100.0

	detectedCount := 0
	for _, eco := range ecosystems {
		if !eco.Detected {
			continue
		}
		detectedCount++

		totals.Critical += eco.VulnCounts.Critical
		totals.High += eco.VulnCounts.High
		totals.Moderate += eco.VulnCounts.Moderate
		totals.Low += eco.VulnCounts.Low
		totals.Info += eco.VulnCounts.Info
		totals.Unknown += eco.VulnCounts.Unknown

		if eco.LockFile == "missing" {
			score -= DeductMissingLock
		}
	}

	score -= float64(totals.Critical) * DeductCritical
	score -= float64(totals.High) * DeductHigh
	score -= float64(totals.Moderate) * DeductModerate
	score -= float64(totals.Low) * DeductLow
	score -= float64(totals.Unknown) * DeductUnknown

	score = math.Max(0, score)

	if detectedCount == 0 {
		score = 100.0
	}

	return DependencyHealth{
		Ecosystems: ecosystems,
		Totals:     totals,
		Score:      score,
	}
}
