package devenv

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/tmpl"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// NixosPodmanTemplateData holds data for rendering the NixOS Podman rootless guide.
type NixosPodmanTemplateData struct {
	Username string
}

// GenerateNixosPodmanGuide produces docs/nixos-podman-rootless.md when the
// detected OS is NixOS and a Podman container runtime is present.
// Returns nil when the conditions are not met.
func GenerateNixosPodmanGuide(answers types.WizardAnswers) (*types.GeneratedFile, error) {
	rt := answers.Detected.ContainerRuntime
	isPodman := rt == "podman-rootless" || rt == "podman-rootful"
	if answers.Detected.OSFamily != "nixos" || !isPodman {
		return nil, nil
	}

	username := answers.Detected.Username
	if username == "" {
		username = "youruser"
	}

	data := &NixosPodmanTemplateData{Username: username}

	renderer, err := tmpl.NewNixRenderer(templateFS, "templates")
	if err != nil {
		return nil, fmt.Errorf("creating renderer: %w", err)
	}

	content, err := renderer.Render("nixos-podman-rootless.md", data)
	if err != nil {
		return nil, fmt.Errorf("rendering nixos-podman-rootless guide: %w", err)
	}

	return &types.GeneratedFile{
		Path:     "docs/nixos-podman-rootless.md",
		Content:  content,
		Mode:     0o644,
		Strategy: types.Overwrite,
	}, nil
}
