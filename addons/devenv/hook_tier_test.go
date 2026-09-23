package devenv_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
)

func TestFilterHooksByTier_Baseline(t *testing.T) {
	hooks := []string{
		"ripsecrets", "gitleaks", "check-added-large-files",
		"no-commit-to-branch", "check-merge-conflicts",
		"semgrep", "shellcheck", "formatters",
		"lock-file-audit", "nix-secrets-check", "statix",
	}

	result := mustFilterHooksByTier(t, hooks, "baseline")

	expected := map[string]bool{
		"ripsecrets":              true,
		"gitleaks":                true,
		"check-added-large-files": true,
		"no-commit-to-branch":     true,
		"check-merge-conflicts":   true,
	}

	if len(result) != len(expected) {
		t.Errorf("baseline tier: expected %d hooks, got %d: %v", len(expected), len(result), result)
	}

	for _, h := range result {
		if !expected[h] {
			t.Errorf("baseline tier: unexpected hook %q", h)
		}
	}
}

func TestFilterHooksByTier_Enhanced(t *testing.T) {
	hooks := []string{
		"ripsecrets", "gitleaks", "check-added-large-files",
		"no-commit-to-branch", "check-merge-conflicts",
		"semgrep", "shellcheck", "formatters",
		"lock-file-audit", "nix-secrets-check", "statix",
	}

	result := mustFilterHooksByTier(t, hooks, "enhanced")

	expected := map[string]bool{
		"ripsecrets":              true,
		"gitleaks":                true,
		"check-added-large-files": true,
		"no-commit-to-branch":     true,
		"check-merge-conflicts":   true,
		"semgrep":                 true,
		"shellcheck":              true,
		"formatters":              true,
	}

	if len(result) != len(expected) {
		t.Errorf("enhanced tier: expected %d hooks, got %d: %v", len(expected), len(result), result)
	}

	for _, h := range result {
		if !expected[h] {
			t.Errorf("enhanced tier: unexpected hook %q", h)
		}
	}
}

func TestFilterHooksByTier_Specialized(t *testing.T) {
	hooks := []string{
		"ripsecrets", "gitleaks", "check-added-large-files",
		"no-commit-to-branch", "check-merge-conflicts",
		"semgrep", "shellcheck", "formatters",
		"lock-file-audit", "nix-secrets-check", "statix",
	}

	result := mustFilterHooksByTier(t, hooks, "specialized")

	// Specialized includes baseline + enhanced + specialized = all 11.
	if len(result) != 11 {
		t.Errorf("specialized tier: expected 11 hooks, got %d: %v", len(result), result)
	}
}

func TestFilterHooksByTier_Full(t *testing.T) {
	hooks := []string{
		"ripsecrets", "gitleaks", "check-added-large-files",
		"no-commit-to-branch", "check-merge-conflicts",
		"semgrep", "shellcheck", "formatters",
		"lock-file-audit", "nix-secrets-check", "statix",
		"custom-hook-1", "custom-hook-2",
	}

	result := mustFilterHooksByTier(t, hooks, "full")

	// Full tier returns all hooks, including unknown ones.
	if len(result) != len(hooks) {
		t.Errorf("full tier: expected %d hooks, got %d", len(hooks), len(result))
	}
}

func TestFilterHooksByTier_EmptyTier(t *testing.T) {
	hooks := []string{"ripsecrets", "semgrep", "custom-hook"}

	result := mustFilterHooksByTier(t, hooks, "")

	// Empty tier treated as full.
	if len(result) != len(hooks) {
		t.Errorf("empty tier: expected %d hooks, got %d", len(hooks), len(result))
	}
}

func TestFilterHooksByTier_EmptyHooks(t *testing.T) {
	result := mustFilterHooksByTier(t, nil, "baseline")
	if result != nil {
		t.Errorf("expected nil for empty hooks, got %v", result)
	}
}

func TestFilterHooksByTier_PreservesOrder(t *testing.T) {
	hooks := []string{"check-merge-conflicts", "ripsecrets", "gitleaks"}

	result := mustFilterHooksByTier(t, hooks, "baseline")

	if len(result) != 3 {
		t.Fatalf("expected 3 hooks, got %d", len(result))
	}
	if result[0] != "check-merge-conflicts" {
		t.Errorf("expected first hook to be check-merge-conflict, got %q", result[0])
	}
	if result[1] != "ripsecrets" {
		t.Errorf("expected second hook to be ripsecrets, got %q", result[1])
	}
	if result[2] != "gitleaks" {
		t.Errorf("expected third hook to be gitleaks, got %q", result[2])
	}
}

func TestFilterHooksByTier_BaselineExcludesHigherTiers(t *testing.T) {
	hooks := []string{"semgrep", "lock-file-audit", "statix"}

	result := mustFilterHooksByTier(t, hooks, "baseline")

	if len(result) != 0 {
		t.Errorf("baseline should exclude enhanced/specialized hooks, got %v", result)
	}
}

// mustFilterHooksByTier calls FilterHooksByTier and fails the test on error.
func mustFilterHooksByTier(t *testing.T, hooks []string, tier string) []string {
	t.Helper()
	result, err := devenv.FilterHooksByTier(hooks, tier)
	if err != nil {
		t.Fatalf("FilterHooksByTier(%v, %q): %v", hooks, tier, err)
	}
	return result
}

func TestFilterHooksByTier_EnhancedExcludesSpecialized(t *testing.T) {
	hooks := []string{"lock-file-audit", "nix-secrets-check", "statix"}

	result := mustFilterHooksByTier(t, hooks, "enhanced")

	if len(result) != 0 {
		t.Errorf("enhanced should exclude specialized hooks, got %v", result)
	}
}
