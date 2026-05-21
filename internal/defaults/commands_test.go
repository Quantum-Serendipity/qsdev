package defaults

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
)

func TestSectionFromUnified(t *testing.T) {
	t.Parallel()

	u := &catalog.UnifiedDefaults{
		Tiers:      map[string]catalog.TierDef{"starter": {Order: 1, Description: "test"}},
		Compliance: map[string]catalog.ComplianceLevelDef{"basic": {Order: 1}},
		Tools:      map[string]catalog.ToolDef{"ripsecrets": {DisplayName: "ripsecrets"}},
		Services:   []string{"postgres"},
	}

	tests := []struct {
		name      string
		section   string
		wantNil   bool
		wantError bool
	}{
		{name: "tiers", section: "tiers"},
		{name: "compliance", section: "compliance"},
		{name: "profiles", section: "profiles", wantNil: true},
		{name: "profile_aliases", section: "profile_aliases", wantNil: true},
		{name: "project_profiles", section: "project_profiles", wantNil: true},
		{name: "tools", section: "tools"},
		{name: "security_hooks", section: "security_hooks", wantNil: true},
		{name: "base_packages", section: "base_packages", wantNil: true},
		{name: "unset_vars", section: "unset_vars", wantNil: true},
		{name: "keep_vars", section: "keep_vars", wantNil: true},
		{name: "custom_hooks", section: "custom_hooks", wantNil: true},
		{name: "hook_tier_order", section: "hook_tier_order", wantNil: true},
		{name: "hook_tiers", section: "hook_tiers", wantNil: true},
		{name: "tier_to_compliance", section: "tier_to_compliance", wantNil: true},
		{name: "tier_to_enabled_tools", section: "tier_to_enabled_tools", wantNil: true},
		{name: "default_mcp_servers", section: "default_mcp_servers", wantNil: true},
		{name: "default_agent_tools", section: "default_agent_tools", wantNil: true},
		{name: "languages", section: "languages", wantNil: true},
		{name: "services", section: "services"},
		{name: "permission_presets", section: "permission_presets", wantNil: true},
		{name: "hook_presets", section: "hook_presets", wantNil: true},
		{name: "security_levels", section: "security_levels", wantNil: true},
		{name: "data_classifications", section: "data_classifications", wantNil: true},
		{name: "package_managers", section: "package_managers", wantNil: true},
		{name: "tool_categories", section: "tool_categories", wantNil: true},
		{name: "case insensitive", section: "TIERS"},
		{name: "mixed case", section: "Tools"},

		{name: "unknown section", section: "nonexistent", wantError: true},
		{name: "empty section", section: "", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result, err := sectionFromUnified(u, tt.section)
			if tt.wantError {
				if err == nil {
					t.Fatalf("expected error for section %q, got nil", tt.section)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error for section %q: %v", tt.section, err)
			}
			if !tt.wantNil && result == nil {
				t.Errorf("expected non-nil result for section %q", tt.section)
			}
		})
	}
}

func TestSectionNames(t *testing.T) {
	t.Parallel()

	names := sectionNames()
	if len(names) == 0 {
		t.Fatal("sectionNames returned empty slice")
	}

	// Every name should be a valid section.
	u := &catalog.UnifiedDefaults{}
	for _, name := range names {
		_, err := sectionFromUnified(u, name)
		if err != nil {
			t.Errorf("sectionNames includes %q but sectionFromUnified rejects it: %v", name, err)
		}
	}
}
