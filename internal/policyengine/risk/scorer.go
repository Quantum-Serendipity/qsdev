package risk

import (
	"slices"
	"time"
)

var categoryWeights = map[string]float64{
	"publication":   0.15,
	"maintainer":    0.12,
	"behavioral":    0.20,
	"vulnerability": 0.35,
	"popularity":    0.08,
	"provenance":    0.10,
}

func ScorePackage(info *PackageInfo) PackageScore {
	var probes []ProbeResult

	for _, reg := range allProbes {
		if !probeApplies(reg, info.Ecosystem) {
			continue
		}
		result := reg.Fn(info)
		probes = append(probes, result)
	}

	categoryProbes := make(map[string][]ProbeResult)
	for _, p := range probes {
		categoryProbes[p.Category] = append(categoryProbes[p.Category], p)
	}

	var categories []CategoryScore
	weightedSum := 0.0
	activeWeightSum := 0.0

	for cat, weight := range categoryWeights {
		cp := categoryProbes[cat]
		if len(cp) == 0 {
			continue
		}

		scoreSum := 0.0
		available := 0
		for _, p := range cp {
			if p.Status != ProbeDataUnavailable {
				scoreSum += p.Score
				available++
			}
		}

		var rawScore float64
		if available > 0 {
			rawScore = scoreSum / float64(available)
			weightedSum += rawScore * weight
			activeWeightSum += weight
		}

		categories = append(categories, CategoryScore{
			Name:     cat,
			Weight:   weight,
			RawScore: rawScore,
			Probes:   cp,
		})
	}

	aggregate := 0
	if activeWeightSum > 0 {
		aggregate = int(weightedSum / activeWeightSum)
	}

	capped, ceiling := ApplyCeilings(aggregate, info)

	// Data-freshness floor. The vulnerability probes now report
	// ProbeDataUnavailable until an OSV/KEV lookup has actually run (see
	// PackageInfo.VulnDataAvailable), so an unenriched package no longer earns a
	// passing 100 for the absence of CVEs it was never checked for. Its
	// vulnerability category (weight 0.35) drops out of activeWeightSum, pushing
	// the collected signal below minActiveWeight. Rather than fail open on a
	// package whose security posture is unknown, quarantine it at grade F.
	if insufficientData(info, activeWeightSum) {
		capped = 0
		if ceiling == "" {
			ceiling = "insufficient-data"
		}
	}

	return PackageScore{
		PackageName:    info.Name,
		PackageVersion: info.Version,
		Ecosystem:      info.Ecosystem,
		Score:          capped,
		Grade:          gradeFromScore(capped),
		Categories:     categories,
		CeilingApplied: ceiling,
		Probes:         probes,
		DataFreshness:  time.Now(),
	}
}

// minActiveWeight is the minimum share of the total category weight that must
// be backed by real (available, non-stub) probe data for a package to be scored
// on the normal A–F scale. The category weights sum to 1.0, so this is an
// absolute floor on how much of the weighted signal was actually collected.
const minActiveWeight = 0.50

// insufficientData reports whether a package carries too little telemetry to be
// judged safe. A package with no publication timestamp was never located in a
// registry (its provenance cannot be established); a package whose available
// probes cover less than minActiveWeight of the weighted signal is effectively
// unenriched; and a package whose vulnerability status was never established has
// an unknown posture. Any condition fails closed to grade F rather than open.
func insufficientData(info *PackageInfo, activeWeightSum float64) bool {
	if info.FirstPublishedAt == nil && info.PublishedAt == nil {
		return true
	}
	if activeWeightSum < minActiveWeight {
		return true
	}
	// Backstop: even if future non-stub probes lift the remaining categories'
	// weight above the floor, a package whose vulnerability status was never
	// established (OSV/KEV lookup skipped, offline, or rate-limited) has an
	// unknown — not clean — posture and must not be judged safe.
	return !info.VulnDataAvailable
}

// Scorer is a stateless PackageRiskScorer implementation that delegates to the
// package-level scoring functions. It exists so the risk subsystem can be wired
// into the SecurityOrchestrator through the PackageRiskScorer interface.
type Scorer struct{}

// NewScorer returns a ready-to-use Scorer.
func NewScorer() Scorer { return Scorer{} }

// ScorePackage scores a single package.
func (Scorer) ScorePackage(info *PackageInfo) PackageScore { return ScorePackage(info) }

// ScoreAll scores a dependency set and rolls up aggregate health.
func (Scorer) ScoreAll(packages []PackageInfo) DependencyHealth { return ScoreAll(packages) }

func probeApplies(reg ProbeRegistration, eco Ecosystem) bool {
	if reg.Ecosystems == nil {
		return true
	}
	return slices.Contains(reg.Ecosystems, eco)
}

func gradeFromScore(score int) RiskGrade {
	switch {
	case score >= 90:
		return GradeA
	case score >= 80:
		return GradeB
	case score >= 70:
		return GradeC
	case score >= 50:
		return GradeD
	default:
		return GradeF
	}
}
