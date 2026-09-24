// Package check implements the qsdev check command for CI enforcement.
// It verifies binary compatibility, config integrity, required tools,
// generated file state, and security hardening.
package check

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// CheckCategory groups related checks for reporting.
type CheckCategory string

const (
	CategoryBinaryCompat      CheckCategory = "binary_compatibility"
	CategoryConfigIntegrity   CheckCategory = "config_integrity"
	CategoryRequiredTools     CheckCategory = "required_tools"
	CategoryFileState         CheckCategory = "generated_file_state"
	CategorySecurityHarden    CheckCategory = "security_hardening"
	CategoryDenyConflicts     CheckCategory = "deny_rule_conflicts"
	CategoryCustomConformance CheckCategory = "custom_conformance"
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
	case CategoryCustomConformance:
		return "Custom Conformance"
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
	Category    CheckCategory     `json:"category"`
	Name        string            `json:"name"`
	Status      CheckStatus       `json:"status"`
	Severity    CheckSeverity     `json:"severity"`
	Message     string            `json:"message"`
	Remediation string            `json:"remediation,omitempty"`
	FilePath    string            `json:"file_path,omitempty"`
	AutoFixable bool              `json:"auto_fixable"`
	Metadata    map[string]string `json:"metadata,omitempty"`
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
	ProjectRoot          string
	BinaryVersion        string
	QsdevConfig          *types.QsdevConfig
	ConfigErr            error    // why QsdevConfig is nil: not found vs. failed to parse
	ToolNames            []string // every registered tool, for config name validation
	AlwaysOnToolNames    []string // tools that must never appear in tools.disabled
	ProfileNames         []string // project-type profiles, for validating `profile`
	RequiredDenyRules    []string
	StateFile            string
	DenyRules            []string
	SkillOps             []SkillOps
	ExpectedConflictKeys map[string]string
	// ExpectedClaudeSettings is the .claude/settings.json the generator
	// produces for the project's saved answers (nil when unknown); its hook
	// registrations and bypass setting must still be in force on disk.
	ExpectedClaudeSettings []byte
	// ManifestFile is the committed manifest of machine-owned generated files
	// (state.ManifestFile under the project root). Unlike StateFile it exists
	// on a clean CI checkout, so it is what CI verifies generated files
	// against. Empty disables the manifest check.
	ManifestFile string
	// CustomConformance is the project's own conformance policy
	// (.qsdev-policy.yaml) as evaluated against a posture assessment by the
	// command layer; nil when the project has no custom policy.
	CustomConformance *CustomConformance
	// DeclaredEnv holds the environment variables the project's devenv
	// modules (devenv.nix, devenv.local.nix) declare, read by the command
	// layer; the cloud isolation check judges environment separation from
	// it. DeclaredEnvErr records a module that could not be read or parsed.
	DeclaredEnv    map[string]string
	DeclaredEnvErr error
	// ProbeTool runs a tool's version probe for the toolchain requirement
	// checks; nil skips them.
	ProbeTool ToolProber
}

// CustomConformance carries the evaluated requirements of a project's custom
// conformance policy. It mirrors posture's custom conformance level so this
// package does not depend on the posture assessment.
type CustomConformance struct {
	// PolicyFile is the policy file's path relative to the project root.
	PolicyFile   string
	Requirements []PolicyRequirement
}

// PolicyRequirement is the outcome of one custom conformance requirement.
type PolicyRequirement struct {
	Name   string
	Pass   bool
	Reason string
}

// CheckFailedError signals that checks failed at the given audit level.
type CheckFailedError struct {
	FailCount int
	Level     AuditLevel
}

func (e *CheckFailedError) Error() string {
	return fmt.Sprintf("%d check(s) failed at audit level %s", e.FailCount, e.Level)
}

func (e *CheckFailedError) ExitCode() int {
	return 1
}
