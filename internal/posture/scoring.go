package posture

import "math"

// WeightMultiplier returns the numeric weight for a LayerWeight.
func WeightMultiplier(w LayerWeight) float64 {
	switch w {
	case WeightCritical:
		return 10.0
	case WeightHigh:
		return 7.5
	case WeightMedium:
		return 5.0
	case WeightLow:
		return 2.5
	default:
		return 0.0
	}
}

// ComputeDefenseScore computes weighted defense coverage (0-100).
// Not-applicable layers are excluded from both numerator and denominator.
func ComputeDefenseScore(layers []DefenseLayer) float64 {
	var totalWeight, earnedWeight float64
	for _, l := range layers {
		if l.Status == LayerNotApplicable {
			continue
		}
		w := WeightMultiplier(l.Weight)
		totalWeight += w
		switch l.Status {
		case LayerEnabled:
			earnedWeight += w
		case LayerPartial:
			earnedWeight += w * float64(l.Score) / 10.0
		}
	}
	if totalWeight == 0 {
		return 100.0
	}
	return (earnedWeight / totalWeight) * 100.0
}

// ComputeTierRelativeDefenseScore computes weighted defense coverage (0-100)
// considering only layers whose MinTier <= currentTier. Layers above the
// project's current tier are excluded from both numerator and denominator,
// so a T1 project is not penalized for missing T3-only layers.
// Not-applicable layers are also excluded.
func ComputeTierRelativeDefenseScore(layers []DefenseLayer, currentTier int) float64 {
	var totalWeight, earnedWeight float64
	for _, l := range layers {
		if l.MinTier > currentTier {
			continue
		}
		if l.Status == LayerNotApplicable {
			continue
		}
		w := WeightMultiplier(l.Weight)
		totalWeight += w
		switch l.Status {
		case LayerEnabled:
			earnedWeight += w
		case LayerPartial:
			earnedWeight += w * float64(l.Score) / 10.0
		}
	}
	if totalWeight == 0 {
		return 100.0
	}
	return (earnedWeight / totalWeight) * 100.0
}

// ComputeAggregateScore combines three layer scores with 40/30/30 weighting.
func ComputeAggregateScore(defense, config, deps float64) AggregateScore {
	total := defense*0.40 + config*0.30 + deps*0.30
	return AggregateScore{
		Total:     math.Round(total*10) / 10,
		Grade:     ScoreToGrade(total),
		Defense:   math.Round(defense*10) / 10,
		Config:    math.Round(config*10) / 10,
		DepHealth: math.Round(deps*10) / 10,
	}
}

// ScoreToGrade converts 0-100 score to letter grade using INTEGER rounding.
// 89.5 rounds to 90 = "A-", 89.4 rounds to 89 = "B+"
func ScoreToGrade(score float64) string {
	rounded := int(math.Round(score))
	switch {
	case rounded >= 97:
		return "A+"
	case rounded >= 93:
		return "A"
	case rounded >= 90:
		return "A-"
	case rounded >= 87:
		return "B+"
	case rounded >= 83:
		return "B"
	case rounded >= 80:
		return "B-"
	case rounded >= 77:
		return "C+"
	case rounded >= 73:
		return "C"
	case rounded >= 70:
		return "C-"
	case rounded >= 67:
		return "D+"
	case rounded >= 63:
		return "D"
	case rounded >= 60:
		return "D-"
	default:
		return "F"
	}
}
