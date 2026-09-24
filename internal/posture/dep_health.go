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
	// DeductScanError penalizes an ecosystem whose vulnerability scan was
	// attempted but errored. Its zero counts mean "unknown", not "clean", so,
	// like an unresolved-severity vulnerability, it is deducted at the High
	// rate rather than scored as a clean 100.
	DeductScanError = DeductUnknown
)

// ComputeDepScore calculates dependency health from the per-ecosystem scan
// outcomes: the scan status (see depScanStatus), the vulnerability totals and
// the score (0-100). The score starts at 100 and deducts per vulnerability,
// per missing lock file and per ecosystem whose scan errored, floored at 0.
// Unscanned dependencies have unknown health, so their score is nil rather
// than a clean 100; a project with no detected ecosystem has nothing to be
// vulnerable and scores 100.
func ComputeDepScore(ecosystems []EcosystemStatus) DependencyHealth {
	status := depScanStatus(ecosystems)
	health := DependencyHealth{
		Ecosystems: ecosystems,
		Status:     status,
		Scanned:    status == DepScanned,
		ScanFailed: status == DepScanFailed,
	}

	var totals VulnSeverityCounts
	score := 100.0
	for _, eco := range ecosystems {
		if !eco.Detected {
			continue
		}

		totals.Critical += eco.VulnCounts.Critical
		totals.High += eco.VulnCounts.High
		totals.Moderate += eco.VulnCounts.Moderate
		totals.Low += eco.VulnCounts.Low
		totals.Info += eco.VulnCounts.Info
		totals.Unknown += eco.VulnCounts.Unknown

		if eco.LockFile == "missing" {
			score -= DeductMissingLock
		}
		if eco.ScanError {
			score -= DeductScanError
		}
	}

	score -= float64(totals.Critical) * DeductCritical
	score -= float64(totals.High) * DeductHigh
	score -= float64(totals.Moderate) * DeductModerate
	score -= float64(totals.Low) * DeductLow
	score -= float64(totals.Unknown) * DeductUnknown

	health.Totals = totals
	if status != DepUnscanned {
		score = math.Max(0, score)
		health.Score = &score
	}
	return health
}

// depScanStatus derives the aggregate scan status from the detected
// ecosystems. Any ecosystem whose scan errored makes it DepScanFailed. It is
// DepScanned only when at least one ecosystem was in fact scanned: a project
// whose lock files have no OSV coverage is unscanned, not scanned clean.
// Ecosystems are only marked Scanned or ScanError when a scan was requested,
// so without one the status is DepUnscanned.
func depScanStatus(ecosystems []EcosystemStatus) DepScanStatus {
	detected, scanned := false, false
	for _, eco := range ecosystems {
		if !eco.Detected {
			continue
		}
		if eco.ScanError {
			return DepScanFailed
		}
		detected = true
		scanned = scanned || eco.Scanned
	}
	switch {
	case !detected:
		return DepNotApplicable
	case scanned:
		return DepScanned
	default:
		return DepUnscanned
	}
}
