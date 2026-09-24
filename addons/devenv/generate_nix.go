package devenv

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/tmpl"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// GenerateDevenvNix produces a GeneratedFile containing the rendered devenv.nix
// from wizard answers and ecosystem module registry.
func GenerateDevenvNix(answers types.WizardAnswers, registry *ecosystem.Registry) (*types.GeneratedFile, error) {
	content, err := renderDevenvNix(answers, registry)
	if err != nil {
		return nil, err
	}

	// The pieces above each write their own attribute paths; define every
	// key once so the file parses and passes the always-on statix hook.
	normalized, err := normalizeNixModule(string(content))
	if err != nil {
		return nil, fmt.Errorf("normalizing devenv.nix attributes: %w", err)
	}

	return &types.GeneratedFile{
		Path:     "devenv.nix",
		Content:  []byte(normalized),
		Mode:     fileutil.ModeReadWrite,
		Strategy: types.ManualMerge,
	}, nil
}

// renderDevenvNix renders the devenv.nix template as assembled from its
// pieces, before normalizeNixModule groups repeated keys.
func renderDevenvNix(answers types.WizardAnswers, registry *ecosystem.Registry) ([]byte, error) {
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
	return content, nil
}
