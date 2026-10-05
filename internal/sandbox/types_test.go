package sandbox

import (
	"slices"
	"testing"
)

func TestDegradationTier_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		tier DegradationTier
		want string
	}{
		{TierFull, "full"},
		{TierBwrapWithoutLandlock, "bwrap-without-landlock"},
		{TierBwrapWithoutSeccomp, "bwrap-without-seccomp"},
		{TierBwrapOnly, "bwrap-only"},
		{TierSystemdRun, "systemd-run"},
		{TierUnsandboxed, "unsandboxed"},
		{DegradationTier(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.tier.String(); got != tt.want {
				t.Errorf("DegradationTier(%d).String() = %q, want %q", tt.tier, got, tt.want)
			}
		})
	}
}

func TestHookCategory_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cat  HookCategory
		want string
	}{
		{CategoryLinter, "linter"},
		{CategoryFormatter, "formatter"},
		{CategoryNetworkLinter, "network-linter"},
		{CategoryGenerator, "generator"},
		{CategoryTestRunner, "test-runner"},
		{HookCategory(99), "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			t.Parallel()
			if got := tt.cat.String(); got != tt.want {
				t.Errorf("HookCategory(%d).String() = %q, want %q", tt.cat, got, tt.want)
			}
		})
	}
}

func TestParseHookCategoryStrict(t *testing.T) {
	t.Parallel()
	const valid = "(valid: linter, formatter, network-linter, generator, test-runner)"
	tests := []struct {
		input   string
		want    HookCategory
		wantErr string
	}{
		{input: "linter", want: CategoryLinter},
		{input: "formatter", want: CategoryFormatter},
		{input: "network-linter", want: CategoryNetworkLinter},
		{input: "generator", want: CategoryGenerator},
		{input: "test-runner", want: CategoryTestRunner},
		{input: "test-runer", wantErr: `unknown category "test-runer" ` + valid},
		{input: "unknown", wantErr: `unknown category "unknown" ` + valid},
		{input: "Linter", wantErr: `unknown category "Linter" ` + valid},
		{input: " linter", wantErr: `unknown category " linter" ` + valid},
		{input: "", wantErr: `unknown category "" ` + valid},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got, err := ParseHookCategoryStrict(tt.input)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("ParseHookCategoryStrict(%q) = %v, %v; want error %q", tt.input, got, err, tt.wantErr)
				}
				return
			}
			if err != nil || got != tt.want {
				t.Errorf("ParseHookCategoryStrict(%q) = %v, %v; want %v", tt.input, got, err, tt.want)
			}
		})
	}
}

// TestParseHookCategoryStrict_RoundTrip pins that every name HookCategoryNames
// lists parses back to the category whose String() it is.
func TestParseHookCategoryStrict_RoundTrip(t *testing.T) {
	t.Parallel()
	names := HookCategoryNames()
	if want := []string{"linter", "formatter", "network-linter", "generator", "test-runner"}; !slices.Equal(names, want) {
		t.Fatalf("HookCategoryNames() = %v, want %v", names, want)
	}
	for _, name := range names {
		cat, err := ParseHookCategoryStrict(name)
		if err != nil {
			t.Errorf("ParseHookCategoryStrict(%q): %v", name, err)
			continue
		}
		if cat.String() != name {
			t.Errorf("ParseHookCategoryStrict(%q) = %v, which does not round-trip", name, cat)
		}
	}
}

func TestHookCategory_WorktreeReadOnly(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cat  HookCategory
		want bool
	}{
		{CategoryLinter, true},
		{CategoryFormatter, false},
		{CategoryNetworkLinter, true},
		{CategoryGenerator, false},
		{CategoryTestRunner, false},
	}
	for _, tt := range tests {
		t.Run(tt.cat.String(), func(t *testing.T) {
			t.Parallel()
			if got := tt.cat.WorktreeReadOnly(); got != tt.want {
				t.Errorf("%s.WorktreeReadOnly() = %v, want %v", tt.cat, got, tt.want)
			}
		})
	}
}

func TestHookCategory_NetworkAllowed(t *testing.T) {
	t.Parallel()
	tests := []struct {
		cat  HookCategory
		want bool
	}{
		{CategoryLinter, false},
		{CategoryFormatter, false},
		{CategoryNetworkLinter, true},
		{CategoryGenerator, false},
		{CategoryTestRunner, true},
	}
	for _, tt := range tests {
		t.Run(tt.cat.String(), func(t *testing.T) {
			t.Parallel()
			if got := tt.cat.NetworkAllowed(); got != tt.want {
				t.Errorf("%s.NetworkAllowed() = %v, want %v", tt.cat, got, tt.want)
			}
		})
	}
}

func TestDefaultResourceLimits(t *testing.T) {
	t.Parallel()
	limits := DefaultResourceLimits()
	if limits.MemoryBytes != 2*1024*1024*1024 {
		t.Errorf("MemoryBytes = %d, want 2GB", limits.MemoryBytes)
	}
	if limits.MaxPIDs != 4096 {
		t.Errorf("MaxPIDs = %d, want 4096", limits.MaxPIDs)
	}
	if limits.CPUQuotaPercent != 200 {
		t.Errorf("CPUQuotaPercent = %d, want 200", limits.CPUQuotaPercent)
	}
}
