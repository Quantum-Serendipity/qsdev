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
