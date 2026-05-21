package config

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ComplianceLevel represents an ordered security compliance tier.
// Higher values indicate stricter security requirements.
type ComplianceLevel int

const (
	// ComplianceLevelBaseline is the minimum security posture.
	ComplianceLevelBaseline ComplianceLevel = iota
	// ComplianceLevelEnhanced adds additional scanning and longer age-gating.
	ComplianceLevelEnhanced
	// ComplianceLevelStrict enforces maximum security with audit logging.
	ComplianceLevelStrict
)

// ComplianceProfile describes the concrete security settings for a compliance level.
type ComplianceProfile struct {
	AgeGatingThresholdHours int
	ScriptBlocking          bool
	RequiredPreCommitHooks  []string
	MCPServerPolicy         string
	ClaudePermissionLevel   string
	ClaudeAuditLog          bool
	SBOMPolicy              string
	LicenseScanning         bool
}

// complianceLevelOrder maps level name to ordinal for comparison.
var complianceLevelOrder = map[string]ComplianceLevel{
	"baseline": ComplianceLevelBaseline,
	"enhanced": ComplianceLevelEnhanced,
	"strict":   ComplianceLevelStrict,
}

// ComplianceLevels maps compliance level names to their profiles.
// Backed by internal/catalog/defaults/compliance.yaml.
var ComplianceLevels = buildComplianceLevels()

func buildComplianceLevels() map[string]ComplianceProfile {
	cat := catalog.Default()
	defs := cat.ComplianceLevels()

	result := make(map[string]ComplianceProfile, len(defs))
	for name, def := range defs {
		result[name] = ComplianceProfile{
			AgeGatingThresholdHours: def.AgeGatingThresholdHours,
			ScriptBlocking:          def.ScriptBlocking,
			RequiredPreCommitHooks:  def.RequiredPreCommitHooks,
			MCPServerPolicy:         def.MCPServerPolicy,
			ClaudePermissionLevel:   def.ClaudePermissionLevel,
			ClaudeAuditLog:          def.ClaudeAuditLog,
			SBOMPolicy:              def.SBOMPolicy,
			LicenseScanning:         def.LicenseScanning,
		}
	}
	return result
}

// ParseComplianceLevel converts a string to a ComplianceLevel ordinal.
func ParseComplianceLevel(s string) (ComplianceLevel, error) {
	level, ok := complianceLevelOrder[s]
	if !ok {
		return 0, fmt.Errorf("unknown compliance level %q; valid values: baseline, enhanced, strict", s)
	}
	return level, nil
}

// CompareComplianceLevels compares two compliance level strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Unknown levels are treated as below baseline.
func CompareComplianceLevels(a, b string) int {
	aLevel := complianceLevelOrder[a]
	bLevel := complianceLevelOrder[b]

	if aLevel < bLevel {
		return -1
	}
	if aLevel > bLevel {
		return 1
	}
	return 0
}

// ComplianceLevelToConfig converts a compliance level name to a QsdevConfig
// overlay suitable for merging into the resolution chain.
func ComplianceLevelToConfig(level string) *types.QsdevConfig {
	profile, ok := ComplianceLevels[level]
	if !ok {
		return nil
	}

	t := true
	return &types.QsdevConfig{
		Security: types.SecurityConfig{
			Level:          level,
			AgeGating:      &t,
			ScriptBlocking: boolPtr(profile.ScriptBlocking),
			LockEnforce:    &t,
			VulnScanning:   &t,
		},
		Tools: types.ToolsConfig{
			Enabled: profile.RequiredPreCommitHooks,
		},
		ClaudeCode: types.ClaudeCodeConfig{
			PermissionLevel: profile.ClaudePermissionLevel,
		},
	}
}

// boolPtr returns a pointer to a bool value.
func boolPtr(v bool) *bool {
	return &v
}
