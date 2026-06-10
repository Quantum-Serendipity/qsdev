package risk

func ScoreAll(packages []PackageInfo) DependencyHealth {
	health := DependencyHealth{
		TotalPackages:     len(packages),
		GradeDistribution: make(map[RiskGrade]int),
	}

	if len(packages) == 0 {
		return health
	}

	weightedSum := 0.0
	totalWeight := 0.0

	for i := range packages {
		score := ScorePackage(&packages[i])

		health.GradeDistribution[score.Grade]++

		if score.Grade == GradeF {
			health.CriticalFindings = append(health.CriticalFindings, score)
		}

		weight := 1.0
		if packages[i].IsDirect {
			weight = 2.0
		}
		weightedSum += float64(score.Score) * weight
		totalWeight += weight
	}

	if totalWeight > 0 {
		health.AggregateScore = int(weightedSum / totalWeight)
	}
	health.AggregateGrade = gradeFromScore(health.AggregateScore)

	return health
}
