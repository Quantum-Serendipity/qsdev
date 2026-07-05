package config

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestComplianceLevel_BaselineMappings(t *testing.T) {
	p := GetComplianceLevels()["baseline"]
	if p.AgeGatingThresholdHours != 72 {
		t.Errorf("expected 72h age-gating, got %d", p.AgeGatingThresholdHours)
	}
	if len(p.RequiredPreCommitHooks) != 2 {
		t.Errorf("expected 2 hooks, got %d", len(p.RequiredPreCommitHooks))
	}
	if p.MCPServerPolicy != "allow-list" {
		t.Errorf("expected allow-list, got %q", p.MCPServerPolicy)
	}
	if p.ClaudePermissionLevel != "standard" {
		t.Errorf("expected standard, got %q", p.ClaudePermissionLevel)
	}
	if p.SBOMPolicy != "off" {
		t.Errorf("expected off, got %q", p.SBOMPolicy)
	}
}

func TestComplianceLevel_EnhancedMappings(t *testing.T) {
	p := GetComplianceLevels()["enhanced"]
	if p.AgeGatingThresholdHours != 168 {
		t.Errorf("expected 168h age-gating, got %d", p.AgeGatingThresholdHours)
	}
	if len(p.RequiredPreCommitHooks) != 3 {
		t.Errorf("expected 3 hooks, got %d: %v", len(p.RequiredPreCommitHooks), p.RequiredPreCommitHooks)
	}
	if p.SBOMPolicy != "on-release" {
		t.Errorf("expected on-release, got %q", p.SBOMPolicy)
	}
}

func TestComplianceLevel_StrictMappings(t *testing.T) {
	p := GetComplianceLevels()["strict"]
	if p.AgeGatingThresholdHours != 336 {
		t.Errorf("expected 336h age-gating, got %d", p.AgeGatingThresholdHours)
	}
	if len(p.RequiredPreCommitHooks) != 4 {
		t.Errorf("expected 4 hooks, got %d: %v", len(p.RequiredPreCommitHooks), p.RequiredPreCommitHooks)
	}
	if p.MCPServerPolicy != "explicit-only" {
		t.Errorf("expected explicit-only, got %q", p.MCPServerPolicy)
	}
	if p.ClaudePermissionLevel != "restricted" {
		t.Errorf("expected restricted, got %q", p.ClaudePermissionLevel)
	}
	if !p.ClaudeAuditLog {
		t.Error("expected ClaudeAuditLog to be true")
	}
	if p.SBOMPolicy != "every-build" {
		t.Errorf("expected every-build, got %q", p.SBOMPolicy)
	}
	if !p.LicenseScanning {
		t.Error("expected LicenseScanning to be true")
	}
}

func TestComplianceLevelToConfig_ProducesValidQsdevConfig(t *testing.T) {
	for _, level := range []string{"baseline", "enhanced", "strict"} {
		cfg := ComplianceLevelToConfig(level)
		if cfg == nil {
			t.Errorf("ComplianceLevelToConfig(%q) returned nil", level)
			continue
		}
		if cfg.Security.Level != level {
			t.Errorf("expected security level %q, got %q", level, cfg.Security.Level)
		}
		if cfg.Security.AgeGating == nil || !*cfg.Security.AgeGating {
			t.Errorf("expected age_gating=true for level %q", level)
		}
	}

	// Unknown level returns nil.
	if cfg := ComplianceLevelToConfig("unknown"); cfg != nil {
		t.Error("expected nil for unknown level")
	}
}

func TestParseComplianceLevel_Ordering(t *testing.T) {
	baseline, err := ParseComplianceLevel("baseline")
	if err != nil {
		t.Fatal(err)
	}
	enhanced, err := ParseComplianceLevel("enhanced")
	if err != nil {
		t.Fatal(err)
	}
	strict, err := ParseComplianceLevel("strict")
	if err != nil {
		t.Fatal(err)
	}

	if baseline >= enhanced {
		t.Error("expected baseline < enhanced")
	}
	if enhanced >= strict {
		t.Error("expected enhanced < strict")
	}
}

func TestParseComplianceLevel_Unknown(t *testing.T) {
	_, err := ParseComplianceLevel("ultra-secure")
	if err == nil {
		t.Error("expected error for unknown compliance level")
	}
}

func TestCompareComplianceLevels_AllPairs(t *testing.T) {
	t.Parallel()
	tests := []struct {
		a, b string
		want int
	}{
		{"baseline", "baseline", 0},
		{"enhanced", "enhanced", 0},
		{"strict", "strict", 0},
		{"baseline", "enhanced", -1},
		{"baseline", "strict", -1},
		{"enhanced", "strict", -1},
		{"enhanced", "baseline", 1},
		{"strict", "baseline", 1},
		{"strict", "enhanced", 1},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			t.Parallel()
			got := CompareComplianceLevels(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("CompareComplianceLevels(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

// TestCompareComplianceLevels_UnknownSortsBelowBaseline pins F-CAP-4.2-3: an
// unknown/typo'd level must sort strictly below baseline so it can never
// silently satisfy a security floor.
func TestCompareComplianceLevels_UnknownSortsBelowBaseline(t *testing.T) {
	t.Parallel()
	tests := []struct {
		a, b string
		want int
	}{
		{"strikt", "baseline", -1}, // typo'd "strict"
		{"", "baseline", -1},       // empty
		{"garbage", "enhanced", -1},
		{"garbage", "strict", -1},
		{"baseline", "strikt", 1}, // known outranks unknown
		{"unknown-a", "unknown-b", 0},
	}
	for _, tt := range tests {
		if got := CompareComplianceLevels(tt.a, tt.b); got != tt.want {
			t.Errorf("CompareComplianceLevels(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

// TestResolveConfig_FloorRaisesTypodSecurityLevel is the end-to-end guard for
// F-CAP-4.2-3: a garbage/typo'd resolved security level with a known floor must
// trigger a floor violation and be raised to the floor, not silently pass
// through as if it were baseline.
func TestResolveConfig_FloorRaisesTypodSecurityLevel(t *testing.T) {
	org := DefaultQsdevConfig()
	project := &types.QsdevConfig{
		Security: types.SecurityConfig{Level: "baseline"}, // floor
	}
	// A local layer supplies a misspelled level; last-wins merge would leave it
	// in place, so the floor check is the only line of defense.
	local := &LocalConfig{
		Security: types.SecurityConfig{Level: "strikt"},
	}
	result, err := ResolveConfig(org, nil, project, local, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Config.Security.Level != "baseline" {
		t.Errorf("expected typo'd level raised to floor %q, got %q", "baseline", result.Config.Security.Level)
	}
	if len(result.Violations) == 0 {
		t.Error("expected a floor violation for the invalid security level, got none")
	}
}
