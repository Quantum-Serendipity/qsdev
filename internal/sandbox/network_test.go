package sandbox

import (
	"strings"
	"testing"
)

func TestSandboxConfig_NetworkIsolated(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		category HookCategory
		mode     string
		want     bool
	}{
		{"explicit deny beats network category", CategoryTestRunner, NetworkModeDeny, true},
		{"explicit deny on network-linter", CategoryNetworkLinter, NetworkModeDeny, true},
		{"allow on linter shares network", CategoryLinter, NetworkModeAllow, false},
		{"filtered shares network", CategoryNetworkLinter, NetworkModeFiltered, false},
		{"empty mode uses network category default", CategoryTestRunner, "", false},
		{"empty mode uses isolated category default", CategoryFormatter, "", true},
		{"unknown mode fails closed", CategoryTestRunner, "denied", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := &SandboxConfig{HookCategory: tt.category, Network: NetworkPolicy{Mode: tt.mode}}
			if got := cfg.NetworkIsolated(); got != tt.want {
				t.Errorf("NetworkIsolated() = %v, want %v (effective mode %q)", got, tt.want, cfg.EffectiveNetworkMode())
			}
		})
	}
}

func TestSandboxConfig_UnenforcedNetworkControls(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		cfg     SandboxConfig
		wantLen int
		wantSub []string
	}{
		{
			name:    "deny enforces everything",
			cfg:     SandboxConfig{Network: NetworkPolicy{Mode: NetworkModeDeny, DenyLAN: true}},
			wantLen: 0,
		},
		{
			name:    "allow is not a filter",
			cfg:     SandboxConfig{Network: NetworkPolicy{Mode: NetworkModeAllow}},
			wantLen: 0,
		},
		{
			name: "filtered reports mode, egress and LAN",
			cfg: SandboxConfig{Network: NetworkPolicy{
				Mode:        NetworkModeFiltered,
				EgressRules: []EgressRule{{Host: "proxy.golang.org", Port: 443}},
				DenyLAN:     true,
			}},
			wantLen: 3,
			wantSub: []string{"filtered", "egress", "denyLAN"},
		},
		{
			name:    "category default filtered is reported",
			cfg:     SandboxConfig{HookCategory: CategoryTestRunner},
			wantLen: 1,
			wantSub: []string{"filtered"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.cfg.UnenforcedNetworkControls()
			if len(got) != tt.wantLen {
				t.Fatalf("UnenforcedNetworkControls() = %v, want %d entries", got, tt.wantLen)
			}
			joined := strings.Join(got, "\n")
			for _, sub := range tt.wantSub {
				if !strings.Contains(joined, sub) {
					t.Errorf("UnenforcedNetworkControls() = %v, missing %q", got, sub)
				}
			}
		})
	}
}
