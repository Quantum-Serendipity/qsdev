package devenv_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devenv"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestGenerateNixosPodmanGuide_NotNixOS(t *testing.T) {
	t.Parallel()

	answers := types.WizardAnswers{
		Detected: types.DetectedProject{
			OSFamily:         "debian",
			ContainerRuntime: "podman-rootless",
		},
	}

	got, err := devenv.GenerateNixosPodmanGuide(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil when OSFamily is not nixos, got %+v", got)
	}
}

func TestGenerateNixosPodmanGuide_NotPodman(t *testing.T) {
	t.Parallel()

	answers := types.WizardAnswers{
		Detected: types.DetectedProject{
			OSFamily:         "nixos",
			ContainerRuntime: "docker",
		},
	}

	got, err := devenv.GenerateNixosPodmanGuide(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil when ContainerRuntime is docker, got %+v", got)
	}
}

func TestGenerateNixosPodmanGuide_NoPodmanNoOS(t *testing.T) {
	t.Parallel()

	answers := types.WizardAnswers{
		Detected: types.DetectedProject{},
	}

	got, err := devenv.GenerateNixosPodmanGuide(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil when both fields are empty, got %+v", got)
	}
}

func TestGenerateNixosPodmanGuide_PodmanRootlessNixOS(t *testing.T) {
	t.Parallel()

	answers := types.WizardAnswers{
		Detected: types.DetectedProject{
			OSFamily:         "nixos",
			ContainerRuntime: "podman-rootless",
			Username:         "alice",
		},
	}

	got, err := devenv.GenerateNixosPodmanGuide(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil GeneratedFile, got nil")
		return
	}
	if got.Path != "docs/nixos-podman-rootless.md" {
		t.Errorf("Path = %q, want %q", got.Path, "docs/nixos-podman-rootless.md")
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

func TestGenerateNixosPodmanGuide_PodmanRootfulNixOS(t *testing.T) {
	t.Parallel()

	answers := types.WizardAnswers{
		Detected: types.DetectedProject{
			OSFamily:         "nixos",
			ContainerRuntime: "podman-rootful",
			Username:         "bob",
		},
	}

	got, err := devenv.GenerateNixosPodmanGuide(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil GeneratedFile for podman-rootful, got nil")
		return
	}
	if got.Path != "docs/nixos-podman-rootless.md" {
		t.Errorf("Path = %q, want %q", got.Path, "docs/nixos-podman-rootless.md")
	}
}

func TestGenerateNixosPodmanGuide_ContainsUsername(t *testing.T) {
	t.Parallel()

	answers := types.WizardAnswers{
		Detected: types.DetectedProject{
			OSFamily:         "nixos",
			ContainerRuntime: "podman-rootless",
			Username:         "testuser",
		},
	}

	got, err := devenv.GenerateNixosPodmanGuide(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil GeneratedFile, got nil")
		return
	}

	content := string(got.Content)
	if !strings.Contains(content, "testuser") {
		t.Error("content does not contain the provided username 'testuser'")
	}
}

func TestGenerateNixosPodmanGuide_ContainsAllSections(t *testing.T) {
	t.Parallel()

	answers := types.WizardAnswers{
		Detected: types.DetectedProject{
			OSFamily:         "nixos",
			ContainerRuntime: "podman-rootless",
			Username:         "alice",
		},
	}

	got, err := devenv.GenerateNixosPodmanGuide(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil GeneratedFile, got nil")
		return
	}

	content := string(got.Content)

	sections := []string{
		"virtualisation.podman",
		"subUidRanges",
		"/run/wrappers/bin",
		"linger",
		"quadlet-nix",
		"DOCKER_HOST",
		"overlay",
	}

	for _, section := range sections {
		if !strings.Contains(content, section) {
			t.Errorf("content does not contain expected section marker %q", section)
		}
	}
}

func TestGenerateNixosPodmanGuide_FallbackUsername(t *testing.T) {
	t.Parallel()

	answers := types.WizardAnswers{
		Detected: types.DetectedProject{
			OSFamily:         "nixos",
			ContainerRuntime: "podman-rootless",
			Username:         "",
		},
	}

	got, err := devenv.GenerateNixosPodmanGuide(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil GeneratedFile, got nil")
		return
	}

	content := string(got.Content)
	if !strings.Contains(content, "youruser") {
		t.Error("content does not contain fallback username 'youruser'")
	}
}
