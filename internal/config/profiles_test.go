package config

import (
	"strings"
	"testing"
)

func TestGetBuiltInProfile_ConsultingDefault(t *testing.T) {
	cfg, err := GetBuiltInProfile("consulting-default")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Security.Level != "enhanced" {
		t.Errorf("expected enhanced security, got %q", cfg.Security.Level)
	}
	// Should have semgrep, gitleaks, secretspec in tools.
	toolSet := make(map[string]bool)
	for _, tool := range cfg.Tools.Enabled {
		toolSet[tool] = true
	}
	for _, expected := range []string{"semgrep", "gitleaks", "secretspec"} {
		if !toolSet[expected] {
			t.Errorf("expected tool %q in consulting-default profile", expected)
		}
	}
}

func TestGetBuiltInProfile_StartupFast(t *testing.T) {
	cfg, err := GetBuiltInProfile("startup-fast")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Security.Level != "baseline" {
		t.Errorf("expected baseline security, got %q", cfg.Security.Level)
	}
	if len(cfg.Tools.Enabled) != 1 || cfg.Tools.Enabled[0] != "gitleaks" {
		t.Errorf("expected [gitleaks], got %v", cfg.Tools.Enabled)
	}
}

func TestGetBuiltInProfile_Enterprise(t *testing.T) {
	cfg, err := GetBuiltInProfile("enterprise")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Security.Level != "strict" {
		t.Errorf("expected strict security, got %q", cfg.Security.Level)
	}
	if cfg.ClaudeCode.PermissionLevel != "restricted" {
		t.Errorf("expected restricted permissions, got %q", cfg.ClaudeCode.PermissionLevel)
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
	// Error should list available profiles.
	for _, name := range []string{"consulting-default", "startup-fast", "enterprise"} {
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

	// Verify sorted by name.
	for i := 1; i < len(profiles); i++ {
		if profiles[i].Name < profiles[i-1].Name {
			t.Errorf("profiles not sorted: %q comes after %q",
				profiles[i].Name, profiles[i-1].Name)
		}
	}

	// Expected order: consulting-default, enterprise, startup-fast.
	expectedOrder := []string{"consulting-default", "enterprise", "startup-fast"}
	for i, expected := range expectedOrder {
		if profiles[i].Name != expected {
			t.Errorf("index %d: expected %q, got %q", i, expected, profiles[i].Name)
		}
	}

	// Each should have a description.
	for _, p := range profiles {
		if p.Description == "" {
			t.Errorf("profile %q has empty description", p.Name)
		}
	}
}
