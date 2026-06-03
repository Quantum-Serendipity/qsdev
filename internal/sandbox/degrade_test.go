package sandbox

import "testing"

func TestDetermineTier(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		caps *SystemCapabilities
		want DegradationTier
	}{
		{
			name: "all features",
			caps: &SystemCapabilities{
				HasBwrap:    true,
				HasUserNS:   true,
				LandlockABI: 4,
				HasSeccomp:  true,
				HasCgroupV2: true,
			},
			want: TierFull,
		},
		{
			name: "no landlock",
			caps: &SystemCapabilities{
				HasBwrap:   true,
				HasUserNS:  true,
				HasSeccomp: true,
			},
			want: TierBwrapWithoutLandlock,
		},
		{
			name: "no seccomp",
			caps: &SystemCapabilities{
				HasBwrap:    true,
				HasUserNS:   true,
				LandlockABI: 1,
			},
			want: TierBwrapWithoutSeccomp,
		},
		{
			name: "bwrap but no userns",
			caps: &SystemCapabilities{
				HasBwrap:      true,
				HasSystemdRun: true,
			},
			want: TierSystemdRun,
		},
		{
			name: "systemd-run only",
			caps: &SystemCapabilities{
				HasSystemdRun: true,
			},
			want: TierSystemdRun,
		},
		{
			name: "nothing available",
			caps: &SystemCapabilities{},
			want: TierUnsandboxed,
		},
		{
			name: "nil capabilities",
			caps: nil,
			want: TierUnsandboxed,
		},
		{
			name: "bwrap without userns no systemd",
			caps: &SystemCapabilities{
				HasBwrap: true,
			},
			want: TierUnsandboxed,
		},
		{
			name: "full features except cgroup",
			caps: &SystemCapabilities{
				HasBwrap:    true,
				HasUserNS:   true,
				LandlockABI: 2,
				HasSeccomp:  true,
			},
			want: TierFull,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := DetermineTier(tt.caps)
			if got != tt.want {
				t.Errorf("DetermineTier() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTierMessage(t *testing.T) {
	t.Parallel()
	tests := []struct {
		tier     DegradationTier
		wantMsg  bool
		contains string
	}{
		{TierFull, false, ""},
		{TierBwrapWithoutLandlock, true, "Landlock"},
		{TierBwrapWithoutSeccomp, true, "Seccomp"},
		{TierSystemdRun, true, "systemd-run"},
		{TierUnsandboxed, true, "qsdev doctor"},
	}
	for _, tt := range tests {
		t.Run(tt.tier.String(), func(t *testing.T) {
			t.Parallel()
			msg := TierMessage(tt.tier)
			if tt.wantMsg && msg == "" {
				t.Error("expected non-empty message")
			}
			if !tt.wantMsg && msg != "" {
				t.Errorf("expected empty message, got: %s", msg)
			}
			if tt.contains != "" {
				if len(msg) == 0 || !containsSubstr(msg, tt.contains) {
					t.Errorf("message should contain %q, got: %s", tt.contains, msg)
				}
			}
		})
	}
}

func TestTierSecurityLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		tier DegradationTier
		want string
	}{
		{TierFull, "strong"},
		{TierBwrapWithoutLandlock, "moderate"},
		{TierBwrapWithoutSeccomp, "moderate"},
		{TierSystemdRun, "minimal"},
		{TierUnsandboxed, "none"},
	}
	for _, tt := range tests {
		t.Run(tt.tier.String(), func(t *testing.T) {
			t.Parallel()
			if got := TierSecurityLevel(tt.tier); got != tt.want {
				t.Errorf("TierSecurityLevel(%v) = %q, want %q", tt.tier, got, tt.want)
			}
		})
	}
}

func containsSubstr(s, substr string) bool {
	return len(s) >= len(substr) && searchSubstr(s, substr)
}

func searchSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
