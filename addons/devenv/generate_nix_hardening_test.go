package devenv_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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
		return
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

// TestGenerateNixHardeningGuide_Accuracy covers W173 and W174: evaluator
// settings are described as client-side, untrusted users' keys as ignored,
// and the standalone config does not lock macOS/Debian users out.
func TestGenerateNixHardeningGuide_Accuracy(t *testing.T) {
	t.Parallel()
	got, err := devenv.GenerateNixHardeningGuide(types.WizardAnswers{NixHardeningGuide: true})
	if err != nil {
		t.Fatalf("GenerateNixHardeningGuide: %v", err)
	}
	content := string(got.Content)
	ignored := section(content, "**Settings that are IGNORED per-user", "**Evaluator settings")
	perUser := section(content, "**Settings that work per-user:**", "**Settings that are IGNORED")

	tests := []struct {
		name    string
		in      string
		sub     string
		present bool
	}{
		{"restrict-eval is not called daemon-enforced", ignored, "restrict-eval", false},
		{"evaluator settings are client-side", content, "**Evaluator settings (client-side, NOT enforced by the daemon):**", true},
		{"restrict-eval can be overridden", content, "--option restrict-eval false", true},
		{"untrusted public keys are ignored", ignored, "extra-trusted-public-keys", true},
		{"public keys are not a per-user setting", perUser, "trusted-public-keys", false},
		{"no wheel-only allowed-users in standalone config", content, "allowed-users = root @nixbld @wheel", false},
		{"standalone config names the user", content, "allowed-users = root <your-username>", true},
		{"macOS daemon restart", content, "sudo launchctl kickstart -k system/org.nixos.nix-daemon", true},
		{"no invalid sandbox check", content, "nix build --sandbox false", false},
		{"valid sandbox check", content, "nix build --option sandbox false", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := strings.Contains(tt.in, tt.sub); got != tt.present {
				t.Errorf("contains %q = %v, want %v", tt.sub, got, tt.present)
			}
		})
	}
}

// section returns the text of content between the start and end markers.
func section(content, start, end string) string {
	_, after, ok := strings.Cut(content, start)
	if !ok {
		return ""
	}
	before, _, _ := strings.Cut(after, end)
	return before
}
