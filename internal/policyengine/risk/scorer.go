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

	// Data-freshness floor. The probe set treats the *absence* of a negative
	// signal (no CVEs, no KEV listing, no install scripts) as a passing score,
	// which is indistinguishable from "the package was never enriched". Dividing
	// the weighted sum by activeWeightSum then normalizes the missing categories
	// away, so an all-unknown package scores ~87 (grade B). Rather than fail open
	// on a package with no telemetry, quarantine it at grade F.
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
// registry (its provenance cannot be established), and a package whose available
// probes cover less than minActiveWeight of the weighted signal is effectively
// unenriched. Either condition fails closed to grade F rather than failing open.
func insufficientData(info *PackageInfo, activeWeightSum float64) bool {
	if info.FirstPublishedAt == nil && info.PublishedAt == nil {
		return true
	}
	return activeWeightSum < minActiveWeight
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
