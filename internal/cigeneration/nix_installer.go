package cigeneration

import (
	_ "embed" // the Nix installer and its pins are embedded catalog assets
)

// Where a generated project keeps the Nix installer and its pins, relative to
// the project root. Generated CI runs
// `.github/scripts/install-nix.sh .github/nix-release.env` in place of a
// third-party Nix action.
const (
	InstallNixScriptPath = ".github/scripts/install-nix.sh"
	NixReleasePinsPath   = ".github/nix-release.env"
)

// installNixScriptSrc installs the official Nix release named by the pins,
// verifying the tarball's SHA-256 before anything from it runs.
//
//go:embed install-nix.sh
var installNixScriptSrc string

// nixReleasePinsSrc is the only source of the Nix release pins: NIX_VERSION
// and one NIX_SHA256_<SYSTEM> tarball digest per runner system.
//
//go:embed nix-release.env
var nixReleasePinsSrc string

// InstallNixScript returns the content of the generated
// .github/scripts/install-nix.sh. Each call returns a fresh copy.
func InstallNixScript() []byte {
	return []byte(installNixScriptSrc)
}

// NixReleasePins returns the content of the generated .github/nix-release.env.
// Each call returns a fresh copy.
func NixReleasePins() []byte {
	return []byte(nixReleasePinsSrc)
}
