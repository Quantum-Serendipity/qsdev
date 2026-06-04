package aiframework

import (
	"context"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestEnforcementTierRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		tier EnforcementTier
		name string
	}{
		{TierKernel, "kernel"},
		{TierHook, "hook"},
		{TierPolicy, "policy"},
		{TierAdvisory, "advisory"},
		{TierExternal, "external"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := tt.tier.String()
			if s != tt.name {
				t.Fatalf("String() = %q, want %q", s, tt.name)
			}

			b, err := tt.tier.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText() error: %v", err)
			}
			if string(b) != tt.name {
				t.Fatalf("MarshalText() = %q, want %q", string(b), tt.name)
			}

			var got EnforcementTier
			if err := got.UnmarshalText(b); err != nil {
				t.Fatalf("UnmarshalText(%q) error: %v", string(b), err)
			}
			if got != tt.tier {
				t.Fatalf("UnmarshalText round-trip: got %d, want %d", got, tt.tier)
			}
		})
	}
}

func TestEnforcementTierUnknown(t *testing.T) {
	t.Parallel()

	unknown := EnforcementTier(99)

	if s := unknown.String(); s != "unknown" {
		t.Fatalf("String() = %q, want %q", s, "unknown")
	}

	if _, err := unknown.MarshalText(); err == nil {
		t.Fatal("MarshalText() should return error for unknown tier")
	}
}

func TestEnforcementTierUnmarshalInvalid(t *testing.T) {
	t.Parallel()

	var tier EnforcementTier
	if err := tier.UnmarshalText([]byte("bogus")); err == nil {
		t.Fatal("UnmarshalText(bogus) should return error")
	}
}

func TestEnforcementTierStrength(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		tier     EnforcementTier
		strength int
	}{
		{"kernel", TierKernel, 5},
		{"hook", TierHook, 4},
		{"policy", TierPolicy, 3},
		{"advisory", TierAdvisory, 2},
		{"external", TierExternal, 1},
		{"unknown", EnforcementTier(99), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.tier.Strength(); got != tt.strength {
				t.Fatalf("Strength() = %d, want %d", got, tt.strength)
			}
		})
	}

	// Verify ordering: Kernel > Hook > Policy > Advisory > External.
	tiers := []EnforcementTier{TierKernel, TierHook, TierPolicy, TierAdvisory, TierExternal}
	for i := 0; i < len(tiers)-1; i++ {
		if tiers[i].Strength() <= tiers[i+1].Strength() {
			t.Fatalf("%s.Strength() (%d) should be > %s.Strength() (%d)",
				tiers[i], tiers[i].Strength(), tiers[i+1], tiers[i+1].Strength())
		}
	}
}

func TestIgnoreCategoryRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		cat  IgnoreCategory
		name string
	}{
		{CategoryCredential, "credential"},
		{CategoryBinary, "binary"},
		{CategoryVendor, "vendor"},
		{CategoryInfrastructure, "infrastructure"},
		{CategoryQsdevInternal, "qsdev_internal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			s := tt.cat.String()
			if s != tt.name {
				t.Fatalf("String() = %q, want %q", s, tt.name)
			}

			b, err := tt.cat.MarshalText()
			if err != nil {
				t.Fatalf("MarshalText() error: %v", err)
			}
			if string(b) != tt.name {
				t.Fatalf("MarshalText() = %q, want %q", string(b), tt.name)
			}

			var got IgnoreCategory
			if err := got.UnmarshalText(b); err != nil {
				t.Fatalf("UnmarshalText(%q) error: %v", string(b), err)
			}
			if got != tt.cat {
				t.Fatalf("UnmarshalText round-trip: got %d, want %d", got, tt.cat)
			}
		})
	}
}

func TestIgnoreCategoryUnknown(t *testing.T) {
	t.Parallel()

	unknown := IgnoreCategory(99)

	if s := unknown.String(); s != "unknown" {
		t.Fatalf("String() = %q, want %q", s, "unknown")
	}

	if _, err := unknown.MarshalText(); err == nil {
		t.Fatal("MarshalText() should return error for unknown category")
	}
}

func TestIgnoreCategoryUnmarshalInvalid(t *testing.T) {
	t.Parallel()

	var cat IgnoreCategory
	if err := cat.UnmarshalText([]byte("bogus")); err == nil {
		t.Fatal("UnmarshalText(bogus) should return error")
	}
}

type mockToolAdapter struct{}

var _ ToolAdapter = (*mockToolAdapter)(nil)

func (m *mockToolAdapter) FrameworkID() FrameworkID         { return ClaudeCode }
func (m *mockToolAdapter) EnforcementTier() EnforcementTier { return TierHook }

func (m *mockToolAdapter) TranslatePermissions(_ context.Context, _ *PermissionPolicy) (*PermissionArtifacts, error) {
	return &PermissionArtifacts{}, nil
}

func (m *mockToolAdapter) TranslateIgnorePatterns(_ context.Context, _ []IgnorePattern) ([]types.GeneratedFile, error) {
	return nil, nil
}

func (m *mockToolAdapter) InjectCredentials(_ context.Context, _ *CredentialScope) (*CredentialArtifacts, error) {
	return &CredentialArtifacts{}, nil
}

func (m *mockToolAdapter) ReportGaps(_ context.Context, _ *PermissionPolicy) []EnforcementGap {
	return nil
}

func TestDefaultSandboxFilters(t *testing.T) {
	t.Parallel()

	filters := DefaultSandboxFilters()
	if len(filters) == 0 {
		t.Fatal("DefaultSandboxFilters() returned empty slice")
	}

	expected := map[string]bool{
		"*_SECRET*":    true,
		"*_TOKEN*":     true,
		"*_KEY*":       true,
		"*_PASSWORD*":  true,
		"AWS_*":        true,
		"GITHUB_TOKEN": true,
		"NPM_TOKEN":    true,
	}

	for _, f := range filters {
		if !expected[f] {
			t.Errorf("unexpected filter pattern: %q", f)
		}
	}

	for pattern := range expected {
		if !slices.Contains(filters, pattern) {
			t.Errorf("missing expected pattern: %q", pattern)
		}
	}
}

func TestEnforcementGapStrengthComparison(t *testing.T) {
	t.Parallel()

	gap := EnforcementGap{
		Rule:         PermissionRule{Pattern: "bash(*)", Reason: "shell access"},
		RequiredTier: TierKernel,
		ActualTier:   TierHook,
		Description:  "shell access requires kernel-level sandbox",
		Mitigation:   "enable container sandbox",
	}

	if gap.RequiredTier.Strength() <= gap.ActualTier.Strength() {
		t.Fatalf("RequiredTier.Strength() (%d) should be > ActualTier.Strength() (%d)",
			gap.RequiredTier.Strength(), gap.ActualTier.Strength())
	}
	if gap.Description == "" {
		t.Error("Description should be non-empty")
	}
	if gap.Mitigation == "" {
		t.Error("Mitigation should be non-empty")
	}
}
