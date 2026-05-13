package devenv_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/devenv"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestGenerateNixHardeningGuide_Disabled(t *testing.T) {
	answers := types.WizardAnswers{
		NixHardeningGuide: false,
	}

	got, err := devenv.GenerateNixHardeningGuide(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil when NixHardeningGuide=false, got %+v", got)
	}
}

func TestGenerateNixHardeningGuide_Enabled(t *testing.T) {
	answers := types.WizardAnswers{
		NixHardeningGuide: true,
	}

	got, err := devenv.GenerateNixHardeningGuide(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil GeneratedFile, got nil")
	}

	if got.Path != "docs/nix-conf-hardening.md" {
		t.Errorf("Path = %q, want %q", got.Path, "docs/nix-conf-hardening.md")
	}
	if got.Mode != 0o644 {
		t.Errorf("Mode = %#o, want %#o", got.Mode, 0o644)
	}
	if got.Strategy != types.Overwrite {
		t.Errorf("Strategy = %v, want Overwrite", got.Strategy)
	}
	if len(got.Content) == 0 {
		t.Error("Content is empty")
	}
}

func TestGenerateNixHardeningGuide_ContainsAllSettings(t *testing.T) {
	answers := types.WizardAnswers{
		NixHardeningGuide: true,
	}

	got, err := devenv.GenerateNixHardeningGuide(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	settings := []string{
		"sandbox",
		"sandbox-fallback",
		"require-sigs",
		"trusted-users",
		"accept-flake-config",
		"filter-syscalls",
		"restrict-eval",
	}

	for _, setting := range settings {
		if !strings.Contains(content, setting) {
			t.Errorf("content does not contain setting %q", setting)
		}
	}
}

func TestGenerateNixHardeningGuide_TrustedUsersWarning(t *testing.T) {
	answers := types.WizardAnswers{
		NixHardeningGuide: true,
	}

	got, err := devenv.GenerateNixHardeningGuide(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	if !strings.Contains(content, "DANGER") {
		t.Error("content does not contain 'DANGER' warning")
	}
	if !strings.Contains(content, "root access") {
		t.Error("content does not contain 'root access' warning")
	}
}

func TestGenerateNixHardeningGuide_DefaultCaches(t *testing.T) {
	answers := types.WizardAnswers{
		NixHardeningGuide: true,
	}

	got, err := devenv.GenerateNixHardeningGuide(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	cacheURLs := []string{
		"https://cache.nixos.org",
		"https://devenv.cachix.org",
		"https://cachix.cachix.org",
	}

	for _, url := range cacheURLs {
		if !strings.Contains(content, url) {
			t.Errorf("content does not contain cache URL %q", url)
		}
	}
}

func TestGenerateNixHardeningGuide_NixOSModule(t *testing.T) {
	answers := types.WizardAnswers{
		NixHardeningGuide: true,
	}

	got, err := devenv.GenerateNixHardeningGuide(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	if !strings.Contains(content, "nix.settings") {
		t.Error("content does not contain NixOS module 'nix.settings' block")
	}
}

func TestGenerateNixHardeningGuide_StandaloneNixConf(t *testing.T) {
	answers := types.WizardAnswers{
		NixHardeningGuide: true,
	}

	got, err := devenv.GenerateNixHardeningGuide(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)

	if !strings.Contains(content, "Standalone nix.conf") {
		t.Error("content does not contain 'Standalone nix.conf' section")
	}
	if !strings.Contains(content, "/etc/nix/nix.conf") {
		t.Error("content does not contain '/etc/nix/nix.conf' path reference")
	}
}
