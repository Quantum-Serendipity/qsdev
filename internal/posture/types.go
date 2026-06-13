package posture

import (
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/sarif"
	"github.com/Quantum-Serendipity/qsdev/internal/posture/drift"
	"github.com/Quantum-Serendipity/qsdev/internal/tier"
)

// SchemaVersion is the current version of the PostureReport schema.
const SchemaVersion = "1.1.0"

// LayerStatus enumerates defense layer states.
type LayerStatus string

const (
	LayerEnabled       LayerStatus = "enabled"
	LayerPartial       LayerStatus = "partial"
	LayerDisabled      LayerStatus = "disabled"
	LayerNotApplicable LayerStatus = "not-applicable"
)

// LayerWeight categorizes defense layer importance.
type LayerWeight string

const (
	WeightCritical LayerWeight = "critical"
	WeightHigh     LayerWeight = "high"
	WeightMedium   LayerWeight = "medium"
	WeightLow      LayerWeight = "low"
)

// TierInfo describes the project's position in the progressive onboarding
// tier system: supply-chain-only (T1), standard (T2), full (T3).
type ReportTierInfo struct {
	Current  string `json:"current"`
	Position int    `json:"position"`
	Total    int    `json:"total"`
	NextTier string `json:"nextTier,omitempty"`
}

// PostureReport is the top-level structure containing the complete security
// posture assessment of a project.
type PostureReport struct {
	SchemaVersion      string                    `json:"schemaVersion"`
	GeneratedAt        time.Time                 `json:"generatedAt"`
	QsdevVersion       string                    `json:"qsdevVersion"`
	ProjectPath        string                    `json:"projectPath"`
	ProjectName        string                    `json:"projectName"`
	Tier               ReportTierInfo            `json:"tier"`
	Score              AggregateScore            `json:"score"`
	Conformance        ConformanceResult         `json:"conformance"`
	Defense            DefenseCoverage           `json:"defense"`
	Config             ConfigHealth              `json:"config"`
	Dependencies       DependencyHealth          `json:"dependencies"`
	Drift              drift.Report              `json:"drift"`
	Tools              []ToolStatus              `json:"tools"`
	Ecosystems         []EcosystemStatus         `json:"ecosystems"`
	PolicyPosture      *sarif.PolicyPosture      `json:"policyPosture,omitempty"`
	PackageRiskPosture *sarif.PackageRiskPosture `json:"packageRiskPosture,omitempty"`
	McpTrustPosture    *sarif.McpTrustPosture    `json:"mcpTrustPosture,omitempty"`
}

// AggregateScore holds the overall security posture grade and sub-scores.
type AggregateScore struct {
	Total     float64 `json:"total"`
	Grade     string  `json:"grade"`
	Defense   float64 `json:"defense"`
	Config    float64 `json:"config"`
	DepHealth float64 `json:"depHealth"`
}

// ConformanceResult holds pass/fail results for baseline and enhanced
// conformance levels, plus an optional custom level.
type ConformanceResult struct {
	Baseline ConformanceLevel  `json:"baseline"`
	Enhanced ConformanceLevel  `json:"enhanced"`
	Custom   *ConformanceLevel `json:"custom,omitempty"`
}

// ConformanceLevel holds a pass/fail verdict and individual checks.
type ConformanceLevel struct {
	Pass   bool               `json:"pass"`
	Checks []ConformanceCheck `json:"checks"`
}

// ConformanceCheck represents a single conformance check with its result.
type ConformanceCheck struct {
	Name   CheckName `json:"name"`
	Pass   bool      `json:"pass"`
	Reason string    `json:"reason,omitempty"`
}

// DefenseCoverage summarizes which defense layers are active and the
// overall defense-in-depth score.
type DefenseCoverage struct {
	Score   float64        `json:"score"`
	Enabled int            `json:"enabled"`
	Total   int            `json:"total"`
	Layers  []DefenseLayer `json:"layers"`
}

// DefenseLayer describes a single defense-in-depth layer and its current status.
type DefenseLayer struct {
	Name    string      `json:"name"`
	Status  LayerStatus `json:"status"`
	Weight  LayerWeight `json:"weight"`
	Score   int         `json:"score"` // 0-10
	MinTier int         `json:"minTier"`
	Details string      `json:"details,omitempty"`
	Reason  string      `json:"reason,omitempty"`
}

// ConfigHealth tracks the state of managed configuration files.
type ConfigHealth struct {
	Score    float64          `json:"score"`
	Total    int              `json:"total"`
	Current  int              `json:"current"`
	Modified int              `json:"modified"`
	Outdated int              `json:"outdated"`
	Missing  int              `json:"missing"`
	Files    []ConfigFileInfo `json:"files"`
}

// ConfigFileInfo describes the state of a single managed configuration file.
type ConfigFileInfo struct {
	Path        string `json:"path"`
	State       string `json:"state"`    // "current"|"modified"|"outdated"|"missing"|"corrupt"
	Category    string `json:"category"` // "machine-owned"|"human-edited"
	HashMatch   bool   `json:"hashMatch"`
	StoredHash  string `json:"storedHash,omitempty"`
	CurrentHash string `json:"currentHash,omitempty"`
}

// DependencyHealth tracks vulnerability counts across ecosystems.
type DependencyHealth struct {
	Score      float64            `json:"score"`
	Ecosystems []EcosystemStatus  `json:"ecosystems"`
	Totals     VulnSeverityCounts `json:"totals"`
	LastScan   *time.Time         `json:"lastScan,omitempty"`
	Stale      bool               `json:"stale"`
}

// VulnSeverityCounts holds vulnerability counts broken down by severity.
type VulnSeverityCounts struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Moderate int `json:"moderate"`
	Low      int `json:"low"`
	Info     int `json:"info"`
}

// Total returns the sum of all vulnerability counts.
func (v VulnSeverityCounts) Total() int {
	return v.Critical + v.High + v.Moderate + v.Low + v.Info
}

// EcosystemStatus tracks the dependency health of a single ecosystem.
type EcosystemStatus struct {
	Name       string             `json:"name"`
	Detected   bool               `json:"detected"`
	LockFile   string             `json:"lockFile"`
	VulnCounts VulnSeverityCounts `json:"vulnCounts"`
	AgeGate    string             `json:"ageGate,omitempty"`
	LastScan   *time.Time         `json:"lastScan,omitempty"`
}

// TierDescription returns a short description for a tier name.
func TierDescription(name string) string {
	for _, t := range tier.AllTiers() {
		if t.Name == name {
			return t.Description
		}
	}
	return ""
}

// ToolStatus describes the availability and configuration of a security tool.
type ToolStatus struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Category    string `json:"category"`
	Enabled     bool   `json:"enabled"`
	Available   bool   `json:"available"`
	ConfigFile  string `json:"configFile,omitempty"`
	Description string `json:"description"`
}

// AssessOptions configures the behavior of the Assess function.
type AssessOptions struct {
	FreshScan  bool
	AuditLevel string
	PolicyFile string
	CacheDir   string
	CacheTTL   time.Duration
}
