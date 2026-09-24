// Package nix embeds the Nix derivations qsdev delivers into user projects.
//
// The derivation sources live at repo root under nix/ (the same files qsdev's
// own flake builds). Because //go:embed can only see files at or below the
// embedding package's directory, this package exists purely to expose them to
// consumers under internal/ (e.g. internal/sectools), which cannot embed a
// sibling directory.
package nix

import (
	_ "embed"
)

// opengrepDerivation is nix/opengrep/default.nix: the pinned, hash-verified
// prebuilt OpenGrep CLI package.
//
//go:embed opengrep/default.nix
var opengrepDerivation []byte

// OpengrepDerivation returns the content of the OpenGrep package derivation
// (nix/opengrep/default.nix). The caller receives its own copy.
func OpengrepDerivation() []byte {
	return append([]byte(nil), opengrepDerivation...)
}
