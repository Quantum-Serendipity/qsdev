package devenv

import (
	"strings"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// toModuleConfig converts a LanguageChoice into an ecosystem.ModuleConfig.
func toModuleConfig(lang types.LanguageChoice) ecosystem.ModuleConfig {
	return ecosystem.ToModuleConfig(lang)
}

// extrasMap converts extras to a map[string]string via the shared helper.
func extrasMap(extras []string) map[string]string {
	return ecosystem.ExtrasMap(extras)
}

// inputKeyFromURL derives an input name from a Nix flake URL by extracting
// the repository name. For example:
//   - "github:NixOS/nixpkgs/nixos-25.11" → "nixpkgs"
//   - "github:cachix/nixpkgs-python"      → "nixpkgs-python"
//   - "github:cachix/git-hooks.nix"       → "git-hooks.nix"
func inputKeyFromURL(url string) string {
	parts := strings.Split(url, "/")
	if len(parts) >= 2 {
		return parts[1]
	}
	return url
}
