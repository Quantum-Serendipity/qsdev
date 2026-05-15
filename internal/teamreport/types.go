package teamreport

import (
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/posture"
)

// TeamReport is the top-level aggregation of security posture across
// multiple projects. It powers the team dashboard and alerting pipeline.
type TeamReport struct {
	SchemaVersion string           `json:"schemaVersion"`
	GeneratedAt   time.Time        `json:"generatedAt"`
	Summary       TeamSummary      `json:"summary"`
	Projects      []ProjectSummary `json:"projects"`
	Trends        []ProjectTrend   `json:"trends,omitempty"`
	Alerts        []PostureAlert   `json:"alerts"`
}

// TeamSummary provides aggregate metrics across all projects.
type TeamSummary struct {
	ProjectCount       int     `json:"projectCount"`
	AverageScore       float64 `json:"averageScore"`
	MedianScore        float64 `json:"medianScore"`
	BaselinePassRate   float64 `json:"baselinePassRate"`
	EnhancedPassRate   float64 `json:"enhancedPassRate"`
	TotalCriticalVulns int     `json:"totalCriticalVulns"`
	TotalHighVulns     int     `json:"totalHighVulns"`
	ProjectsNeedUpdate int     `json:"projectsNeedingUpdate"`
}

// ProjectSummary condenses a single project's posture report into the
// fields relevant for team-level aggregation and display.
type ProjectSummary struct {
	Name        string                     `json:"name"`
	Repo        string                     `json:"repo,omitempty"`
	Score       posture.AggregateScore     `json:"score"`
	Conformance posture.ConformanceResult  `json:"conformance"`
	VulnTotals  posture.VulnSeverityCounts `json:"vulnTotals"`
	QsdevVersion string                     `json:"qsdevVersion"`
	LastScan    time.Time                  `json:"lastScan"`
	Stale       bool                       `json:"stale,omitempty"`
}

// ProjectTrend tracks score history for a single project over time.
type ProjectTrend struct {
	Project    string       `json:"project"`
	DataPoints []TrendPoint `json:"dataPoints"`
}

// TrendPoint is a single data point in a project's score history.
type TrendPoint struct {
	Date  string  `json:"date"`
	Score float64 `json:"score"`
}

// PostureAlert represents a security concern that requires human attention.
type PostureAlert struct {
	Project  string `json:"project"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Action   string `json:"action"`
}

// IssueSpec describes a GitHub issue to be created for a project with
// degraded security posture.
type IssueSpec struct {
	Title  string   `json:"title"`
	Body   string   `json:"body"`
	Repo   string   `json:"repo"`
	Labels []string `json:"labels"`
}

// ScopeFile defines the set of projects to include in a team report
// when using the scope-based collection method.
type ScopeFile struct {
	Projects []ScopeProject `json:"projects" yaml:"projects"`
}

// ScopeProject identifies a single project within a scope file.
type ScopeProject struct {
	Repo   string `json:"repo" yaml:"repo"`
	Branch string `json:"branch,omitempty" yaml:"branch,omitempty"`
}

// AggregateOptions configures the behavior of the Aggregate function.
type AggregateOptions struct {
	HistoryFile   string  `json:"historyFile,omitempty"`
	IncludeTrends bool    `json:"includeTrends,omitempty"`
	Threshold     float64 `json:"threshold,omitempty"`
	QsdevVersion   string  `json:"qsdevVersion,omitempty"`
}
