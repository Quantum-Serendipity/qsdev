package policy

import (
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox"
)

// TestToSandboxConfig_DenyIsNotAMount is the policy-side regression for the
// inverted deny: filesystem.deny entries must travel as explicit deny intent,
// never as mounts (which the backend would bind and so EXPOSE).
func TestToSandboxConfig_DenyIsNotAMount(t *testing.T) {
	t.Parallel()

	spec := DefaultPolicy()
	spec.Filesystem.Deny = append(spec.Filesystem.Deny, "/opt/company-secrets")

	cfg := ToSandboxConfig(spec, sandbox.CategoryLinter, "")

	if !slices.Contains(cfg.Deny, "/opt/company-secrets") {
		t.Errorf("custom deny entry missing from cfg.Deny: %v", cfg.Deny)
	}
	if len(cfg.Deny) != len(spec.Filesystem.Deny) {
		t.Errorf("cfg.Deny has %d entries, want %d", len(cfg.Deny), len(spec.Filesystem.Deny))
	}
	for _, m := range cfg.Mounts {
		if slices.Contains(spec.Filesystem.Deny, m.Source) {
			t.Errorf("deny entry %q was turned into a mount: %+v", m.Source, m)
		}
	}
}

func TestToSandboxConfig_NetworkMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		baseMode     string
		categoryMode string
		override     string
		category     sandbox.HookCategory
		wantMode     string
		wantIsolated bool
	}{
		{
			name:         "explicit category deny isolates a network category",
			baseMode:     "deny",
			categoryMode: "deny",
			category:     sandbox.CategoryTestRunner,
			wantMode:     "deny",
			wantIsolated: true,
		},
		{
			name:         "hook override deny isolates a network category",
			baseMode:     "allow",
			categoryMode: "filtered",
			override:     "deny",
			category:     sandbox.CategoryNetworkLinter,
			wantMode:     "deny",
			wantIsolated: true,
		},
		{
			name:         "empty category mode inherits the base mode",
			baseMode:     "deny",
			categoryMode: "",
			category:     sandbox.CategoryTestRunner,
			wantMode:     "deny",
			wantIsolated: true,
		},
		{
			name:         "empty everywhere falls back to the category default",
			category:     sandbox.CategoryTestRunner,
			wantMode:     "filtered",
			wantIsolated: false,
		},
		{
			name:         "empty everywhere keeps a linter isolated",
			category:     sandbox.CategoryLinter,
			wantMode:     "deny",
			wantIsolated: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			spec := DefaultPolicy()
			spec.Network.Mode = tt.baseMode
			catName := tt.category.String()
			cat := spec.HookCategories[catName]
			cat.Network = tt.categoryMode
			spec.HookCategories[catName] = cat
			if tt.override != "" {
				spec.HookOverrides = map[string]HookOverride{"h": {NetworkOverride: tt.override}}
			}

			cfg := ToSandboxConfig(spec, tt.category, "h")

			if got := cfg.EffectiveNetworkMode(); got != tt.wantMode {
				t.Errorf("EffectiveNetworkMode() = %q, want %q", got, tt.wantMode)
			}
			if got := cfg.NetworkIsolated(); got != tt.wantIsolated {
				t.Errorf("NetworkIsolated() = %v, want %v", got, tt.wantIsolated)
			}
		})
	}
}
