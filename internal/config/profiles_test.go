package config

import (
	"strings"
	"testing"
)

func TestGetBuiltInProfile_SupplyChainOnly(t *testing.T) {
	cfg, err := GetBuiltInProfile("supply-chain-only")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Security.Level != "baseline" {
		t.Errorf("expected baseline security, got %q", cfg.Security.Level)
	}
	if cfg.ClaudeCode.PermissionLevel != "supply-chain-only" {
		t.Errorf("expected supply-chain-only permissions, got %q", cfg.ClaudeCode.PermissionLevel)
	}
	if cfg.Tier != "supply-chain-only" {
		t.Errorf("expected tier supply-chain-only, got %q", cfg.Tier)
	}
	if len(cfg.ClaudeCode.MCPServers) != 0 {
		t.Errorf("expected no MCP servers, got %v", cfg.ClaudeCode.MCPServers)
	}
	if len(cfg.Tools.Enabled) != 0 {
		t.Errorf("expected no tools, got %v", cfg.Tools.Enabled)
	}
}

func TestGetBuiltInProfile_Standard(t *testing.T) {
	cfg, err := GetBuiltInProfile("standard")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Security.Level != "baseline" {
		t.Errorf("expected baseline security, got %q", cfg.Security.Level)
	}
	if cfg.ClaudeCode.PermissionLevel != "standard" {
		t.Errorf("expected standard permissions, got %q", cfg.ClaudeCode.PermissionLevel)
	}
	if cfg.Tier != "standard" {
		t.Errorf("expected tier standard, got %q", cfg.Tier)
	}
	if len(cfg.Tools.Enabled) != 1 || cfg.Tools.Enabled[0] != "gitleaks" {
		t.Errorf("expected [gitleaks], got %v", cfg.Tools.Enabled)
	}
}

func TestGetBuiltInProfile_Full(t *testing.T) {
	cfg, err := GetBuiltInProfile("full")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Security.Level != "enhanced" {
		t.Errorf("expected enhanced security, got %q", cfg.Security.Level)
	}
	if cfg.Tier != "full" {
		t.Errorf("expected tier full, got %q", cfg.Tier)
	}
	toolSet := make(map[string]bool)
	for _, tool := range cfg.Tools.Enabled {
		toolSet[tool] = true
	}
	for _, expected := range []string{"semgrep", "gitleaks", "secretspec"} {
		if !toolSet[expected] {
			t.Errorf("expected tool %q in full profile", expected)
		}
	}
}

func TestGetBuiltInProfile_LegacyAliases(t *testing.T) {
	tests := []struct {
		legacy   string
		resolved string
	}{
		{"startup-fast", "standard"},
		{"consulting-default", "full"},
	}
	for _, tc := range tests {
		cfg, err := GetBuiltInProfile(tc.legacy)
		if err != nil {
			t.Fatalf("GetBuiltInProfile(%q): %v", tc.legacy, err)
		}
		if cfg.Tier != tc.resolved {
			t.Errorf("GetBuiltInProfile(%q).Tier = %q, want %q", tc.legacy, cfg.Tier, tc.resolved)
		}
	}
}

func TestGetBuiltInProfile_Unknown(t *testing.T) {
	_, err := GetBuiltInProfile("nonexistent")
	if err == nil {
		t.Fatal("expected error for unknown profile")
	}
	errMsg := err.Error()
	if !strings.Contains(errMsg, "unknown profile") {
		t.Errorf("expected 'unknown profile' in error, got %q", errMsg)
	}
	for _, name := range []string{"supply-chain-only", "standard", "full"} {
		if !strings.Contains(errMsg, name) {
			t.Errorf("expected %q in error message, got %q", name, errMsg)
		}
	}
}

func TestListBuiltInProfiles_Sorted(t *testing.T) {
	profiles := ListBuiltInProfiles()
	if len(profiles) != 3 {
		t.Fatalf("expected 3 profiles, got %d", len(profiles))
	}

	for i := 1; i < len(profiles); i++ {
		if profiles[i].Name < profiles[i-1].Name {
			t.Errorf("profiles not sorted: %q comes after %q",
				profiles[i].Name, profiles[i-1].Name)
		}
	}

	expectedOrder := []string{"full", "standard", "supply-chain-only"}
	for i, expected := range expectedOrder {
		if profiles[i].Name != expected {
			t.Errorf("index %d: expected %q, got %q", i, expected, profiles[i].Name)
		}
	}

	for _, p := range profiles {
		if p.Description == "" {
			t.Errorf("profile %q has empty description", p.Name)
		}
	}
}

func TestResolveProfileAlias(t *testing.T) {
	tests := []struct {
		input     string
		want      string
		wantAlias bool
	}{
		{"startup-fast", "standard", true},
		{"consulting-default", "full", true},
		{"standard", "standard", false},
		{"full", "full", false},
		{"supply-chain-only", "supply-chain-only", false},
	}
	for _, tc := range tests {
		got, isAlias := ResolveProfileAlias(tc.input)
		if got != tc.want || isAlias != tc.wantAlias {
			t.Errorf("ResolveProfileAlias(%q) = (%q, %v), want (%q, %v)",
				tc.input, got, isAlias, tc.want, tc.wantAlias)
		}
	}
}
