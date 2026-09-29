package defaults

import (
	"reflect"
	"slices"
	"strings"
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
		{name: "mcp_servers", section: "mcp_servers", wantNil: true},
		{name: "docs_corpus", section: "docs_corpus", wantNil: true},
		{name: "permission_deny_rules", section: "permission_deny_rules", wantNil: true},
		{name: "permission_preset_defs", section: "permission_preset_defs", wantNil: true},
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

	names := catalog.SectionNames()
	if len(names) == 0 {
		t.Fatal("catalog.SectionNames returned empty slice")
	}

	// Every advertised section must resolve, including the security-relevant
	// mcp_servers, docs_corpus and permission_* sections.
	u := &catalog.UnifiedDefaults{}
	for _, name := range names {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := sectionFromUnified(u, name); err != nil {
				t.Errorf("catalog.SectionNames includes %q but sectionFromUnified rejects it: %v", name, err)
			}
		})
	}
}

func TestSectionNamesCoverUnifiedDefaults(t *testing.T) {
	t.Parallel()

	known := make(map[string]bool)
	for _, n := range catalog.SectionNames() {
		known[n] = true
	}
	typ := reflect.TypeOf(catalog.UnifiedDefaults{})
	for i := range typ.NumField() {
		tag, _, _ := strings.Cut(typ.Field(i).Tag.Get("yaml"), ",")
		if !known[tag] {
			t.Errorf("UnifiedDefaults section %q (field %s) is missing from catalog.SectionNames", tag, typ.Field(i).Name)
		}
	}
}

func TestSectionFromUnifiedReturnsPermissionRules(t *testing.T) {
	t.Parallel()

	u := &catalog.UnifiedDefaults{
		PermissionDenyRules: map[string][]string{"destructive_ops": {"Bash(rm -rf /)"}},
	}
	got, err := sectionFromUnified(u, "permission_deny_rules")
	if err != nil {
		t.Fatalf("sectionFromUnified: %v", err)
	}
	rules, ok := got.(map[string][]string)
	if !ok || len(rules["destructive_ops"]) != 1 {
		t.Errorf("sectionFromUnified(permission_deny_rules) = %#v, want the deny rules map", got)
	}
}

func TestEditorCommand(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		env      string
		wantBin  string
		wantArgs []string
	}{
		{name: "unset", env: "", wantBin: "vi", wantArgs: []string{"/f.yaml"}},
		{name: "whitespace only", env: " \t ", wantBin: "vi", wantArgs: []string{"/f.yaml"}},
		{name: "plain", env: "nano", wantBin: "nano", wantArgs: []string{"/f.yaml"}},
		{name: "with args", env: "code --wait", wantBin: "code", wantArgs: []string{"--wait", "/f.yaml"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			bin, args := editorCommand(tt.env, "/f.yaml")
			if bin != tt.wantBin || !slices.Equal(args, tt.wantArgs) {
				t.Errorf("editorCommand(%q) = %q %v, want %q %v", tt.env, bin, args, tt.wantBin, tt.wantArgs)
			}
		})
	}
}
