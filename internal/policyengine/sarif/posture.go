package sarif

type PolicyPosture struct {
	RulesActive       int             `json:"rulesActive"`
	RulesTotal        int             `json:"rulesTotal"`
	MonitorModeCount  int             `json:"monitorModeCount"`
	BypassTierSummary map[string]int  `json:"bypassTierSummary"`
	CategoryCoverage  map[string]bool `json:"categoryCoverage"`
}

type PackageRiskPosture struct {
	TotalPackages     int                  `json:"totalPackages"`
	GradeDistribution map[string]int       `json:"gradeDistribution"`
	CriticalFindings  []PackageRiskFinding `json:"criticalFindings,omitempty"`
	AggregateScore    int                  `json:"aggregateScore"`
	AggregateGrade    string               `json:"aggregateGrade"`
}

type PackageRiskFinding struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Score   int    `json:"score"`
	Grade   string `json:"grade"`
	Ceiling string `json:"ceiling,omitempty"`
}

type McpTrustPosture struct {
	Tier1Count             int  `json:"tier1Count"`
	Tier2Count             int  `json:"tier2Count"`
	Tier3Count             int  `json:"tier3Count"`
	ConfusedDeputyActive   bool `json:"confusedDeputyActive"`
	ProjectedDenyRuleCount int  `json:"projectedDenyRuleCount"`
}
