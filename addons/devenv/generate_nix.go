package devenv

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/internal/tmpl"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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
		Path:     "devenv.nix",
		Content:  content,
		Mode:     0o644,
		Strategy: types.ManualMerge,
	}, nil
}
