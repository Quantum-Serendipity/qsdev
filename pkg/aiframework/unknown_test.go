package aiframework

import (
	"encoding"
	"fmt"
	"testing"
)

// textEnum is the String/MarshalText surface every aiframework enum exposes.
type textEnum interface {
	fmt.Stringer
	encoding.TextMarshaler
}

// TestSecurityEnumsZeroValueIsUnknown pins that the zero value of every
// security/health enum is an explicit Unknown that never marshals or parses,
// so a forgotten field or partially built struct cannot read as the strongest
// tier, a healthy report, a passing check, or full content retention.
func TestSecurityEnumsZeroValueIsUnknown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		zero      textEnum
		unknown   textEnum
		unmarshal func([]byte) error
	}{
		{"EnforcementTier", EnforcementTier(0), TierUnknown, func(b []byte) error { var v EnforcementTier; return v.UnmarshalText(b) }},
		{"ContentTier", ContentTier(0), ContentUnknown, func(b []byte) error { var v ContentTier; return v.UnmarshalText(b) }},
		{"HealthStatus", HealthStatus(0), StatusUnknown, func(b []byte) error { var v HealthStatus; return v.UnmarshalText(b) }},
		{"CheckStatus", CheckStatus(0), CheckUnknown, func(b []byte) error { var v CheckStatus; return v.UnmarshalText(b) }},
		{"ValidationSeverity", ValidationSeverity(0), SeverityUnknown, func(b []byte) error { var v ValidationSeverity; return v.UnmarshalText(b) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.zero != tt.unknown {
				t.Fatalf("zero value = %v, want the Unknown constant", tt.zero)
			}
			if got := tt.zero.String(); got != "unknown" {
				t.Errorf("String() = %q, want %q", got, "unknown")
			}
			if b, err := tt.zero.MarshalText(); err == nil {
				t.Errorf("MarshalText() = %q, want error for the Unknown zero value", b)
			}
			for _, text := range []string{"", "unknown"} {
				if err := tt.unmarshal([]byte(text)); err == nil {
					t.Errorf("UnmarshalText(%q) succeeded, want error", text)
				}
			}
		})
	}
}

func TestZeroValueStructsFailClosed(t *testing.T) {
	t.Parallel()

	var arts PermissionArtifacts
	if s := arts.ActiveTier.Strength(); s != 0 {
		t.Errorf("zero PermissionArtifacts.ActiveTier.Strength() = %d, want 0 (weakest)", s)
	}
	for _, tier := range []EnforcementTier{TierKernel, TierHook, TierPolicy, TierAdvisory, TierExternal} {
		if TierUnknown.Strength() >= tier.Strength() {
			t.Errorf("TierUnknown.Strength() = %d, want weaker than %s (%d)", TierUnknown.Strength(), tier, tier.Strength())
		}
	}
	var report HealthReport
	if report.OverallStatus == StatusHealthy {
		t.Error("zero HealthReport reads as healthy")
	}
	var check HealthCheck
	if check.Status == CheckPass {
		t.Error("zero HealthCheck reads as passing")
	}
	var issue ValidationIssue
	if issue.Severity == SeverityWarning {
		t.Error("zero ValidationIssue reads as a warning")
	}
}
