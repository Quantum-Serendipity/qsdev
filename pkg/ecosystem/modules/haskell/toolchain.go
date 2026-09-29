package haskell

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// ghcProbeTimeout bounds the `ghc --numeric-version` probe.
const ghcProbeTimeout = 5 * time.Second

// SetupWarnings reports when a Stack project's GHC will not come from Nix as
// intended: the GHC stack.yaml needs cannot be determined, so devenv.nix
// lets Stack install one itself (see DevenvNixFragment), the haskell version
// is unset (a .qsdev.yaml from before qsdev recorded it), or the configured
// haskell version is not exactly the GHC stack.yaml needs (Stack's default
// compiler check accepts no other), as after a snapshot bump, since
// .qsdev.yaml keeps the version detected at init. Cabal projects need no
// check.
func (m *Module) SetupWarnings(projectRoot string, config ecosystem.ModuleConfig) []string {
	if config.Extra("build_tool", "cabal") != "stack" {
		return nil
	}
	sc, err := readStackCompiler(projectRoot)
	switch {
	case config.Version != "":
		if err != nil || sc.ghc == "" || config.Version == sc.ghc {
			return nil
		}
		return []string{fmt.Sprintf("the configured GHC %s does not match GHC %s that stack.yaml's %s needs, "+
			"so Stack commands fail or install GHC %s themselves; set the haskell version in .qsdev.yaml to %s and run `qsdev init --update`",
			config.Version, sc.ghc, sc.source, sc.ghc, sc.ghc)}
	case err != nil:
		return []string{fmt.Sprintf("cannot tell which GHC the project needs (%v), so devenv.nix lets Stack install GHC itself, outside Nix; "+
			"set the haskell version in .qsdev.yaml to the snapshot's GHC to use nixpkgs' GHC", err)}
	case sc.ghc == "":
		return []string{fmt.Sprintf("cannot tell which GHC stack.yaml's %s needs, so devenv.nix lets Stack install GHC itself, outside Nix; "+
			"set the haskell version in .qsdev.yaml to the snapshot's GHC to use nixpkgs' GHC", sc.source)}
	default:
		// A .qsdev.yaml written before qsdev recorded the snapshot's GHC:
		// init --update keeps its empty version, so devenv.nix falls back
		// to --no-nix although nixpkgs may well have the GHC.
		return []string{fmt.Sprintf("the haskell version in .qsdev.yaml is not set, so devenv.nix lets Stack install GHC %s (stack.yaml's %s) itself, outside Nix; "+
			"set the haskell version to %s and run `qsdev init --update` to use nixpkgs' haskell.compiler.%s where it exists",
			sc.ghc, sc.source, sc.ghc, compilerAttr(sc.ghc))}
	}
}

// ToolchainWarnings reports when a Stack project's stack.yaml needs a GHC
// other than the ghc on PATH. devenv runs Stack with --system-ghc
// --no-install-ghc, so Stack then fails with "No compiler found" — or, when
// devenv.nix fell back to --no-nix because nixpkgs lacks that GHC, installs
// it itself outside Nix. Nothing is reported when the GHC stack.yaml needs is
// unknown or no ghc is on PATH.
func (m *Module) ToolchainWarnings(ctx context.Context, projectRoot string, config ecosystem.ModuleConfig) []string {
	if config.Extra("build_tool", "cabal") != "stack" {
		return nil
	}
	sc, err := readStackCompiler(projectRoot)
	if err != nil || sc.ghc == "" {
		return nil
	}
	probe := m.ghcVersion
	if probe == nil {
		probe = pathGHCVersion
	}
	shellGHC, err := probe(ctx)
	if err != nil || shellGHC == sc.ghc {
		return nil
	}
	return []string{fmt.Sprintf("stack.yaml's %s needs GHC %s but the ghc on PATH is %s, "+
		"so Stack fails with \"No compiler found\" or installs GHC %s itself, outside Nix; "+
		"set the haskell version in .qsdev.yaml to %s and run `qsdev init --update` to use nixpkgs' haskell.compiler.%s where it exists",
		sc.source, sc.ghc, shellGHC, sc.ghc, sc.ghc, compilerAttr(sc.ghc))}
}

// pathGHCVersion returns the version of the ghc on PATH.
func pathGHCVersion(ctx context.Context) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, ghcProbeTimeout)
	defer cancel()
	out, err := exec.CommandContext(ctx, "ghc", "--numeric-version").Output()
	if err != nil {
		return "", fmt.Errorf("running ghc --numeric-version: %w", err)
	}
	v := strings.TrimSpace(string(out))
	if !ghcVersionRe.MatchString(v) {
		return "", errors.New("ghc --numeric-version printed no version")
	}
	return v, nil
}
