package types

import "slices"

// SettingGoSource names the Go language extra that declares where the
// devenv shell's Go comes from (.qsdev.yaml: extras: [go_source=nixpkgs]).
// The generator cannot evaluate Nix, so a project whose nixpkgs (with its
// registered overlays) supplies the Go release go.mod requires says so here.
const SettingGoSource = "go_source"

// Go sources accepted by SettingGoSource.
const (
	// GoSourceGoOverlay, the default, selects the required Go release from
	// the github:purpleclay/go-overlay flake input whenever nixpkgs' Go is
	// older than it.
	GoSourceGoOverlay = "go-overlay"
	// GoSourceNixpkgs declares that pkgs.go is new enough: no go-overlay
	// input is generated, and evaluating the shell fails if pkgs.go is older
	// than the required release.
	GoSourceNixpkgs = "nixpkgs"
)

// GoSources returns the accepted SettingGoSource values.
func GoSources() []string {
	return []string{GoSourceGoOverlay, GoSourceNixpkgs}
}

// IsValidGoSource reports whether s is an accepted SettingGoSource value.
func IsValidGoSource(s string) bool {
	return slices.Contains(GoSources(), s)
}
