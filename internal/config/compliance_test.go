package config

import (
	"testing"
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
