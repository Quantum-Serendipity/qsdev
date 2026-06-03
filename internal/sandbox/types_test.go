package sandbox

import "testing"

func TestDegradationTier_String(t *testing.T) {
	t.Parallel()
	tests := []struct {
		tier DegradationTier
		want string
	}{
		{TierFull, "full"},
		{TierBwrapWithoutLandlock, "bwrap-without-landlock"},
		{TierBwrapWithoutSeccomp, "bwrap-without-seccomp"},
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

func TestParseHookCategory(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  HookCategory
	}{
		{"linter", CategoryLinter},
		{"formatter", CategoryFormatter},
		{"network-linter", CategoryNetworkLinter},
		{"generator", CategoryGenerator},
		{"test-runner", CategoryTestRunner},
		{"unknown", CategoryLinter},
		{"", CategoryLinter},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			if got := ParseHookCategory(tt.input); got != tt.want {
				t.Errorf("ParseHookCategory(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
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

func TestParseHookCategory_RoundTrip(t *testing.T) {
	t.Parallel()
	categories := []HookCategory{
		CategoryLinter, CategoryFormatter, CategoryNetworkLinter,
		CategoryGenerator, CategoryTestRunner,
	}
	for _, cat := range categories {
		t.Run(cat.String(), func(t *testing.T) {
			t.Parallel()
			parsed := ParseHookCategory(cat.String())
			if parsed != cat {
				t.Errorf("ParseHookCategory(%q) = %v, want %v", cat.String(), parsed, cat)
			}
		})
	}
}
