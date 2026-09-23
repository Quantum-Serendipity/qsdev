package config

import (
	"cmp"
	"fmt"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ComplianceLevel represents an ordered security compliance tier.
// Higher values indicate stricter security requirements. The ordinal of a
// level is its catalog `order`; the constants name the built-in levels.
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

// complianceLevelUnknown is the ordinal assigned to any unrecognized
// compliance/security level. It sorts strictly below every known level
// (including baseline) so a garbage or typo'd level can never silently
// satisfy a security floor: enforceSecurityFloor treats it as below the
// floor and raises it, rather than passing it through as if it were baseline.
const complianceLevelUnknown ComplianceLevel = -1

// complianceLevelOrdinal returns the catalog-defined ordinal for a level
// name, or complianceLevelUnknown (below baseline) when the name is not a
// catalog compliance level. Deriving the ordinal from the catalog's `order`
// means an org-defined level sorts where its definition places it.
func complianceLevelOrdinal(name string) ComplianceLevel {
	cat, err := catalog.Default()
	if err != nil {
		return complianceLevelUnknown
	}
	def, ok := cat.ComplianceLevel(name)
	if !ok {
		return complianceLevelUnknown
	}
	return ComplianceLevel(def.Order)
}

// GetComplianceLevels returns compliance level profiles.
// Backed by the compliance section of internal/catalog/defaults.yaml.
func GetComplianceLevels() map[string]ComplianceProfile {
	return buildComplianceLevels()
}

func buildComplianceLevels() map[string]ComplianceProfile {
	cat, err := catalog.Default()
	if err != nil {
		return nil
	}
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

// complianceLevelNames returns the catalog's compliance level names in
// ascending order of strictness.
func complianceLevelNames() []string {
	cat, err := catalog.Default()
	if err != nil {
		return nil
	}
	defs := cat.ComplianceLevels()
	names := make([]string, 0, len(defs))
	for name := range defs {
		names = append(names, name)
	}
	slices.SortFunc(names, func(a, b string) int {
		return cmp.Or(cmp.Compare(defs[a].Order, defs[b].Order), strings.Compare(a, b))
	})
	return names
}

// ParseComplianceLevel converts a string to a ComplianceLevel ordinal.
func ParseComplianceLevel(s string) (ComplianceLevel, error) {
	level := complianceLevelOrdinal(s)
	if level == complianceLevelUnknown {
		return 0, fmt.Errorf("unknown compliance level %q; valid values: %s",
			s, strings.Join(complianceLevelNames(), ", "))
	}
	return level, nil
}

// CompareComplianceLevels compares two compliance level strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Unknown/typo'd levels sort strictly below baseline, so an invalid resolved
// security level always compares as below any recognized floor (fail-closed).
func CompareComplianceLevels(a, b string) int {
	return cmp.Compare(complianceLevelOrdinal(a), complianceLevelOrdinal(b))
}

// ComplianceLevelToConfig converts a compliance level name to a QsdevConfig
// overlay suitable for merging into the resolution chain.
func ComplianceLevelToConfig(level string) *types.QsdevConfig {
	profile, ok := GetComplianceLevels()[level]
	if !ok {
		return nil
	}

	t := true
	return &types.QsdevConfig{
		Security: types.SecurityConfig{
			Level:           level,
			AgeGating:       &t,
			ScriptBlocking:  boolPtr(profile.ScriptBlocking),
			LockEnforcement: &t,
			VulnScanning:    &t,
		},
		Tools: types.ToolsConfig{
			Enabled: requiredHookTools(profile.RequiredPreCommitHooks),
		},
		ClaudeCode: types.ClaudeCodeConfig{
			PermissionLevel: profile.ClaudePermissionLevel,
		},
	}
}

// requiredHookTools maps compliance-required pre-commit hook IDs to the
// catalog tools that provide them. Hook IDs that are not catalog tools (for
// example ripsecrets, which is part of the always-on security hook set) are
// not tools and so must not be listed in tools.enabled.
func requiredHookTools(hookIDs []string) []string {
	cat, err := catalog.Default()
	if err != nil {
		return nil
	}
	var tools []string
	for _, id := range hookIDs {
		if _, ok := cat.Tool(id); ok {
			tools = append(tools, id)
		}
	}
	return tools
}

// boolPtr returns a pointer to a bool value.
func boolPtr(v bool) *bool {
	return &v
}
