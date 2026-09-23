package config

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
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
	if p.ClaudePermissionLevel != "minimal" {
		t.Errorf("expected minimal, got %q", p.ClaudePermissionLevel)
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

// Regression: a client compliance level resolved to a config its own
// validator rejects — strict set the non-existent preset "restricted" and
// every level listed the pre-commit hook ID "ripsecrets" as a tool.
func TestResolveConfig_ClientComplianceLevelValidates(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	var toolNames []string
	for name := range cat.Tools() {
		toolNames = append(toolNames, name)
	}

	for _, level := range []string{"baseline", "enhanced", "strict"} {
		t.Run(level, func(t *testing.T) {
			t.Parallel()
			project := &types.QsdevConfig{
				Version: types.ConfigVersionCurrent,
				Client:  &types.ClientConfig{Name: "acme", SecurityLevel: level},
			}
			result, err := ResolveConfig(DefaultQsdevConfig(), nil, project, nil, false)
			if err != nil {
				t.Fatal(err)
			}
			errs := ValidateQsdevConfig(result.Config, ValidateOptions{ToolNames: toolNames})
			for _, e := range errs {
				t.Errorf("resolved %s config is invalid: %s %q: %s", level, e.Field, e.Value, e.Message)
			}
		})
	}
}

func TestComplianceLevelToConfig_EnablesOnlyCatalogTools(t *testing.T) {
	t.Parallel()
	cfg := ComplianceLevelToConfig("strict")
	if cfg == nil {
		t.Fatal("ComplianceLevelToConfig(strict) returned nil")
	}
	if slices.Contains(cfg.Tools.Enabled, "ripsecrets") {
		t.Errorf("tools.enabled contains the hook ID ripsecrets: %v", cfg.Tools.Enabled)
	}
	for _, want := range []string{"gitleaks", "semgrep", "license-compliance"} {
		if !slices.Contains(cfg.Tools.Enabled, want) {
			t.Errorf("tools.enabled = %v, want it to contain %q", cfg.Tools.Enabled, want)
		}
	}
}

// Regression: level ordering came from a hardcoded map, so an org-defined
// level sorted below baseline and never raised the floor. It must follow the
// catalog's `order`.
func TestCompareComplianceLevels_UsesCatalogOrder(t *testing.T) {
	catalog.ResetDefault()
	t.Cleanup(catalog.ResetDefault)

	overlay := `compliance:
    regulated:
        order: 3
        age_gating_threshold_hours: 720
        script_blocking: true
        required_pre_commit_hooks: [gitleaks]
        mcp_server_policy: explicit-only
        claude_permission_level: minimal
        claude_audit_log: true
        sbom_policy: every-build
        license_scanning: true
`
	orgFile := filepath.Join(t.TempDir(), "defaults.yaml")
	if err := os.WriteFile(orgFile, []byte(overlay), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(branding.Get().EnvPrefix+"ORG_CONFIG", orgFile)

	if got := CompareComplianceLevels("regulated", "strict"); got != 1 {
		t.Errorf("CompareComplianceLevels(regulated, strict) = %d, want 1", got)
	}
	if got := CompareComplianceLevels("baseline", "regulated"); got != -1 {
		t.Errorf("CompareComplianceLevels(baseline, regulated) = %d, want -1", got)
	}
	level, err := ParseComplianceLevel("regulated")
	if err != nil {
		t.Fatalf("ParseComplianceLevel(regulated): %v", err)
	}
	if level != 3 {
		t.Errorf("ParseComplianceLevel(regulated) = %d, want 3", level)
	}
	if err := catalog.OrgOverlayError(); err != nil {
		t.Fatalf("overlay rejected: %v", err)
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
