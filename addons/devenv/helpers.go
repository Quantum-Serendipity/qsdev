package devenv

import (
	"strings"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// toModuleConfig converts a LanguageChoice from wizard answers into an
// ecosystem.ModuleConfig suitable for passing to EcosystemModule methods.
func toModuleConfig(lang types.LanguageChoice) ecosystem.ModuleConfig {
	return ecosystem.ModuleConfig{
		Version:        lang.Version,
		PackageManager: lang.PackageManager,
		Extras:         extrasMap(lang.Extras),
	}
}

// extrasMap converts the []string extras from LanguageChoice into a
// map[string]string for ModuleConfig.Extras. Each string is either:
//   - "key=value" → map[key] = value
//   - "key"       → map[key] = "true"
func extrasMap(extras []string) map[string]string {
	if len(extras) == 0 {
		return nil
	}
	m := make(map[string]string, len(extras))
	for _, e := range extras {
		if k, v, ok := strings.Cut(e, "="); ok {
			m[k] = v
		} else {
			m[e] = "true"
		}
	}
	return m
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
