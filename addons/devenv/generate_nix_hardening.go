package devenv

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/tmpl"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// NixHardeningTemplateData holds data for rendering the nix-conf hardening guide.
type NixHardeningTemplateData struct {
	DefaultCaches []CacheEntry
}

// CacheEntry represents a binary cache with its URL and signing key.
type CacheEntry struct {
	Name      string
	URL       string
	PublicKey string
	Purpose   string
}

var defaultDevenvCaches = []CacheEntry{
	{
		Name:      "cache.nixos.org",
		URL:       "https://cache.nixos.org",
		PublicKey: "cache.nixos.org-1:6NCHdD59X431o0gWypbMrAURkbJ16ZPMQFGspcDShjY=",
		Purpose:   "Official NixOS binary cache",
	},
	{
		Name:      "devenv.cachix.org",
		URL:       "https://devenv.cachix.org",
		PublicKey: "devenv.cachix.org-1:w1cLUi8dv3hnoSPGAuibQv+f9TZLr6cv/Hm9XgU50cw=",
		Purpose:   "devenv.sh project cache",
	},
	{
		Name:      "cachix.cachix.org",
		URL:       "https://cachix.cachix.org",
		PublicKey: "cachix.cachix.org-1:eWNHQldwUO7G2VkjpnjDbWwy4KQ/HNxht7H4SSoMckM=",
		Purpose:   "Cachix service cache",
	},
}

// GenerateNixHardeningGuide produces docs/nix-conf-hardening.md.
// Returns nil when NixHardeningGuide is false in answers.
func GenerateNixHardeningGuide(answers types.WizardAnswers) (*types.GeneratedFile, error) {
	if !answers.NixHardeningGuide {
		return nil, nil
	}

	data := &NixHardeningTemplateData{
		DefaultCaches: defaultDevenvCaches,
	}

	// Use the Nix renderer because all templates share a single embedded FS directory.
	// The Nix func map is a superset of the Markdown func map, so markdown templates
	// parse correctly. Using NewMarkdownRenderer would fail because it cannot parse
	// sibling .tmpl files that reference Nix-specific functions.
	renderer, err := tmpl.NewNixRenderer(templateFS, "templates")
	if err != nil {
		return nil, fmt.Errorf("creating renderer: %w", err)
	}

	content, err := renderer.Render("nix-conf-hardening.md", data)
	if err != nil {
		return nil, fmt.Errorf("rendering nix-conf-hardening guide: %w", err)
	}

	return &types.GeneratedFile{
		Path:     "docs/nix-conf-hardening.md",
		Content:  content,
		Mode:     0o644,
		Strategy: types.Overwrite,
	}, nil
}
