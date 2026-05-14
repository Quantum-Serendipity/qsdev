// Package check implements the gdev check command for CI enforcement.
// It verifies binary compatibility, config integrity, required tools,
// generated file state, and security hardening.
package check

import (
	"fmt"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// CheckCategory groups related checks for reporting.
type CheckCategory string

const (
	CategoryBinaryCompat   CheckCategory = "binary_compatibility"
	CategoryConfigIntegrity CheckCategory = "config_integrity"
	CategoryRequiredTools  CheckCategory = "required_tools"
	CategoryFileState      CheckCategory = "generated_file_state"
	CategorySecurityHarden  CheckCategory = "security_hardening"
	CategoryDenyConflicts   CheckCategory = "deny_rule_conflicts"
)

// categoryDisplayName returns a human-friendly label.
func categoryDisplayName(c CheckCategory) string {
	switch c {
	case CategoryBinaryCompat:
		return "Binary Compatibility"
	case CategoryConfigIntegrity:
		return "Config Integrity"
	case CategoryRequiredTools:
		return "Required Tools"
	case CategoryFileState:
		return "Generated File State"
	case CategorySecurityHarden:
		return "Security Hardening"
	case CategoryDenyConflicts:
		return "Deny Rule Conflicts"
	default:
		return string(c)
	}
}

// CheckSeverity indicates the importance of a check result.
type CheckSeverity string

const (
	SeverityCritical CheckSeverity = "critical"
	SeverityHigh     CheckSeverity = "high"
	SeverityMedium   CheckSeverity = "medium"
	SeverityLow      CheckSeverity = "low"
	SeverityInfo     CheckSeverity = "info"
)

// severityRank returns a numeric rank for severity comparison.
// Higher rank = more severe.
func severityRank(s CheckSeverity) int {
	switch s {
	case SeverityCritical:
		return 4
	case SeverityHigh:
		return 3
	case SeverityMedium:
		return 2
	case SeverityLow:
		return 1
	case SeverityInfo:
		return 0
	default:
		return -1
	}
}

// CheckStatus is the outcome of a single check.
type CheckStatus string

const (
	StatusPass CheckStatus = "pass"
	StatusFail CheckStatus = "fail"
	StatusWarn CheckStatus = "warn"
	StatusSkip CheckStatus = "skip"
)

// CheckResult represents the outcome of a single check.
type CheckResult struct {
	Category    CheckCategory `json:"category"`
	Name        string        `json:"name"`
	Status      CheckStatus   `json:"status"`
	Severity    CheckSeverity `json:"severity"`
	Message     string        `json:"message"`
	Remediation string        `json:"remediation,omitempty"`
	FilePath    string        `json:"file_path,omitempty"`
	AutoFixable bool          `json:"auto_fixable"`
}

// CheckReport is the complete output of a check run.
type CheckReport struct {
	Version   string        `json:"version"`
	Project   string        `json:"project"`
	Timestamp string        `json:"timestamp"`
	Checks    []CheckResult `json:"checks"`
	Summary   CheckSummary  `json:"summary"`
}

// CheckSummary tallies the outcomes of all checks.
type CheckSummary struct {
	Total int `json:"total"`
	Pass  int `json:"pass"`
	Fail  int `json:"fail"`
	Warn  int `json:"warn"`
	Skip  int `json:"skip"`
}

// AuditLevel controls the minimum severity that causes a non-zero exit.
type AuditLevel string

const (
	AuditLevelNone     AuditLevel = "none"
	AuditLevelLow      AuditLevel = "low"
	AuditLevelMedium   AuditLevel = "medium"
	AuditLevelHigh     AuditLevel = "high"
	AuditLevelCritical AuditLevel = "critical"
)

// auditLevelRank returns the minimum severity rank that should cause failure.
func auditLevelRank(level AuditLevel) int {
	switch level {
	case AuditLevelCritical:
		return severityRank(SeverityCritical)
	case AuditLevelHigh:
		return severityRank(SeverityHigh)
	case AuditLevelMedium:
		return severityRank(SeverityMedium)
	case AuditLevelLow:
		return severityRank(SeverityLow)
	case AuditLevelNone:
		return severityRank(SeverityCritical) + 1 // nothing fails
	default:
		return severityRank(SeverityMedium)
	}
}

// OutputFormat selects the report output format.
type OutputFormat string

const (
	FormatHuman OutputFormat = "human"
	FormatJSON  OutputFormat = "json"
	FormatSARIF OutputFormat = "sarif"
	FormatJUnit OutputFormat = "junit"
)

// CheckContext provides all dependencies for running checks.
// Constructed by the command layer to avoid circular imports.
type CheckContext struct {
	ProjectRoot       string
	BinaryVersion     string
	GdevConfig        *types.GdevConfig
	ToolNames         []string
	ProfileNames      []string
	RequiredDenyRules    []string
	StateFile            string
	DenyRules            []string
	SkillOps             []SkillOps
	ExpectedConflictKeys map[string]string
}

// CheckFailedError signals that checks failed at the given audit level.
type CheckFailedError struct {
	FailCount int
	Level     AuditLevel
}

func (e *CheckFailedError) Error() string {
	return fmt.Sprintf("%d check(s) failed at audit level %s", e.FailCount, e.Level)
}
