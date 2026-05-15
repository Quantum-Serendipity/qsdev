package devenv

import (
	"os"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// toModuleConfig converts a LanguageChoice into an ecosystem.ModuleConfig.
func toModuleConfig(lang types.LanguageChoice) ecosystem.ModuleConfig {
	return ecosystem.ToModuleConfig(lang)
}

// toModuleConfigWithProxy converts a LanguageChoice into a ModuleConfig with
// the registry proxy URL resolved.
func toModuleConfigWithProxy(lang types.LanguageChoice, infra types.InfraConfig) ecosystem.ModuleConfig {
	return ecosystem.ToModuleConfigWithProxy(lang, infra)
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

// isAccessible returns true when ACCESSIBLE or NO_COLOR env var is set.
func isAccessible() bool {
	if os.Getenv("ACCESSIBLE") != "" {
		return true
	}
	if os.Getenv("NO_COLOR") != "" {
		return true
	}
	return false
}
