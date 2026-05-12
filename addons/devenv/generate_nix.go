package devenv

import (
	"fmt"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/tmpl"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// GenerateDevenvNix produces a GeneratedFile containing the rendered devenv.nix
// from wizard answers and ecosystem module registry.
func GenerateDevenvNix(answers types.WizardAnswers, registry *ecosystem.Registry) (*types.GeneratedFile, error) {
	data, err := BuildDevenvNixData(answers, registry)
	if err != nil {
		return nil, fmt.Errorf("building devenv.nix template data: %w", err)
	}

	renderer, err := tmpl.NewNixRenderer(templateFS, "templates")
	if err != nil {
		return nil, fmt.Errorf("creating Nix renderer: %w", err)
	}

	content, err := renderer.Render("devenv.nix", data)
	if err != nil {
		return nil, fmt.Errorf("rendering devenv.nix template: %w", err)
	}

	return &types.GeneratedFile{
		Path:           "devenv.nix",
		Content:        content,
		Mode:           0o644,
		Strategy:       types.ManualMerge,
		SkipValidation: true,
	}, nil
}
