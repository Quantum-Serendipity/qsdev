// Package haskell implements the Haskell ecosystem module for
// qsdev. It detects Haskell projects by scanning for
// *.cabal, stack.yaml, and cabal.project files, generates devenv.nix fragments
// with optional Stack support, and provides pre-commit hooks, CI commands,
// deny rules, wizard fields, and package manager metadata for the Haskell
// toolchain.
package haskell

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.DenyRuleProvider = (*Module)(nil)
var _ ecosystem.WizardFieldProvider = (*Module)(nil)
var _ ecosystem.ManifestFileProvider = (*Module)(nil)
var _ ecosystem.SetupWarner = (*Module)(nil)
var _ ecosystem.ToolchainChecker = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Module implements ecosystem.EcosystemModule for the Haskell programming language.
type Module struct {
	// ghcVersion reports the version of the ghc on PATH; nil runs
	// `ghc --numeric-version`. Tests replace it.
	ghcVersion func(ctx context.Context) (string, error)
}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "haskell" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Haskell" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 3 }

// Detect scans projectRoot for Haskell ecosystem indicators: *.cabal files,
// stack.yaml, and cabal.project. It determines the build tool and stores it
// in Extras["build_tool"].
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	cabalMatches, _ := filepath.Glob(filepath.Join(projectRoot, "*.cabal"))
	hasCabal := len(cabalMatches) > 0
	hasStackYaml := fileutil.FileExists(projectRoot, "stack.yaml")
	hasCabalProject := fileutil.FileExists(projectRoot, "cabal.project")

	if !hasCabal && !hasStackYaml && !hasCabalProject {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	var evidence []string
	extras := make(map[string]string)

	if hasCabal {
		evidence = append(evidence, "*.cabal file found")
	}
	if hasStackYaml {
		evidence = append(evidence, "stack.yaml found")
	}
	if hasCabalProject {
		evidence = append(evidence, "cabal.project found")
	}

	// Determine build tool: stack.yaml presence implies Stack, and the GHC
	// version is the one its snapshot pins.
	version := ""
	if hasStackYaml {
		extras["build_tool"] = "stack"
		if sc, err := readStackCompiler(projectRoot); err == nil && sc.ghc != "" {
			version = sc.ghc
			evidence = append(evidence, fmt.Sprintf("stack.yaml %s uses GHC %s", sc.source, sc.ghc))
		}
	} else {
		extras["build_tool"] = "cabal"
	}

	return ecosystem.DetectionResult{
		Detected:   true,
		Confidence: ecosystem.ConfidenceCertain,
		Evidence:   evidence,
		SuggestedConfig: ecosystem.ModuleConfig{
			Version: version,
			Extras:  extras,
		},
	}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Haskell language support. When Stack is the build tool, Stack
// integration is also enabled.
//
// config.Version is the GHC to build with; for a Stack project it is the GHC
// stack.yaml's snapshot pins (see Detect). devenv runs Stack with
// --system-ghc --no-install-ghc, and Stack's default compiler check accepts
// only that exact GHC, so the fragment uses nixpkgs'
// haskell.compiler.ghcXYZ when the pinned nixpkgs provides it. When it does
// not, or the version is unknown, Stack gets only --no-nix and installs the
// snapshot's GHC itself; the fragment warns at evaluation time when a known
// version is missing from nixpkgs (SetupWarnings reports an unknown one).
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	buildTool := config.Extra("build_tool", "cabal")
	version := config.Version
	if version != "" && !ghcNixVersionRe.MatchString(version) {
		return "", fmt.Errorf("invalid GHC version %q: want a version such as 9.6.7 or 9.6", version)
	}

	cfg := ecosystem.NixLangConfig{
		EnablePath: "languages.haskell",
	}
	if buildTool == "stack" {
		cfg.ExtraBlocks = append(cfg.ExtraBlocks,
			"  languages.haskell.stack.enable = true;\n")
	}
	switch {
	case version != "" && buildTool == "stack":
		cfg.ExtraBlocks = append(cfg.ExtraBlocks, stackGHCBlock(version))
	case version != "":
		cfg.ExtraBlocks = append(cfg.ExtraBlocks, ghcPackageBlock(version,
			fmt.Sprintf("GHC %s is configured but nixpkgs has no haskell.compiler.%s; using GHC ${pkgs.ghc.version}", version, compilerAttr(version))))
	case buildTool == "stack":
		cfg.ExtraBlocks = append(cfg.ExtraBlocks,
			"  # qsdev could not tell which GHC stack.yaml's snapshot needs, so Stack\n"+
				"  # installs it itself instead of requiring the shell's GHC.\n"+
				"  languages.haskell.stack.args = [ \"--no-nix\" ];\n")
	}
	cfg.ExtraBlocks = append(cfg.ExtraBlocks,
		"  # NOTE: cabal.project.freeze is NOT a true lockfile — it pins\n"+
			"  # versions but does not record content hashes.\n")

	return ecosystem.BuildLanguageFragment(cfg), nil
}

// ghcNixVersionRe validates a configured GHC version before it is written
// into devenv.nix: a full release, or a major.minor series.
var ghcNixVersionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+(\.[0-9]+)?$`)

// ghcPackageBlock sets languages.haskell.package to nixpkgs'
// haskell.compiler attribute for version when it evaluates, is available on
// the host platform and has that version; otherwise it keeps pkgs.ghc,
// warning with missing (a Nix string body) when missing is not empty.
func ghcPackageBlock(version, missing string) string {
	fallback := "pkgs.ghc"
	if missing != "" {
		fallback = fmt.Sprintf("lib.warn %q pkgs.ghc", missing)
	}
	return fmt.Sprintf(`  languages.haskell.package =
    let
      ghc = builtins.tryEval (pkgs.haskell.compiler.%[1]s or null);
    in
    if ghc.success && ghc.value != null
      && lib.meta.availableOn pkgs.stdenv.hostPlatform ghc.value
      && !(ghc.value.meta.broken or false)
      && %[2]s
    then ghc.value
    else %[3]s;
`, compilerAttr(version), ghcVersionMatch("ghc.value.version", version), fallback)
}

// stackGHCBlock pins a Stack project's shell GHC to version (see
// ghcPackageBlock). When the shell's GHC is not that version, devenv's
// --system-ghc --no-install-ghc would make every Stack command fail, so
// Stack gets only --no-nix and installs the snapshot's GHC itself, with an
// evaluation-time warning saying so.
func stackGHCBlock(version string) string {
	warning := fmt.Sprintf("stack.yaml needs GHC %[1]s but the shell's GHC is "+
		"${config.languages.haskell.package.version} (nixpkgs has no usable haskell.compiler.%[2]s), "+
		"so Stack installs GHC %[1]s itself, outside Nix", version, compilerAttr(version))
	return fmt.Sprintf(`  # stack.yaml's snapshot needs GHC %[1]s exactly, and devenv runs Stack
  # with --system-ghc --no-install-ghc: use nixpkgs' haskell.compiler.%[2]s,
  # or let Stack install GHC %[1]s itself when nixpkgs lacks it.
%[3]s  languages.haskell.stack.args = lib.mkIf
    (!(%[4]s))
    (lib.warn %[5]q [ "--no-nix" ]);
`, version, compilerAttr(version), ghcPackageBlock(version, ""),
		ghcVersionMatch("config.languages.haskell.package.version", version), warning)
}

// ghcVersionMatch returns a Nix boolean expression testing whether the
// version string expr is version: equal to a full release, or in the
// major.minor series.
func ghcVersionMatch(expr, version string) string {
	if strings.Count(version, ".") == 1 {
		return fmt.Sprintf("lib.versions.majorMinor %s == %q", expr, version)
	}
	return fmt.Sprintf("%s == %q", expr, version)
}

// SecurityConfigs returns generated security configuration files.
// Haskell does not produce additional security config files.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// PreCommitHooks returns pre-commit hook definitions for the Haskell ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "ormolu",
			Name:          "ormolu",
			Description:   "Check Haskell code formatting with Ormolu",
			Entry:         "ormolu --mode check",
			Language:      "system",
			Types:         []string{"haskell"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			Files:         `\.hs$`,
			// Custom hook (BuiltIn:false): NixPackage provisions the binary so
			// the emitted `entry` resolves at commit time.
			BuiltIn:    false,
			NixPackage: "ormolu",
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the Haskell ecosystem.
// These prevent direct package installations outside of controlled workflows.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"Bash(cabal install *)",
		"Bash(stack install *)",
	}
}

// CICommands returns CI pipeline commands for the Haskell ecosystem.
// Commands vary based on the configured build tool. Stack's lock-file mode
// error-on-write fails the build when stack.yaml.lock is missing or would
// change (Stack has no --locked flag).
func (m *Module) CICommands(config ecosystem.ModuleConfig) []ecosystem.CICommand {
	buildTool := config.Extra("build_tool", "cabal")

	if buildTool == "stack" {
		return []ecosystem.CICommand{
			{
				Name:        "stack-build-locked",
				Command:     "stack build --lock-file=error-on-write",
				Description: "Build Haskell project with Stack, failing when stack.yaml.lock is missing or stale",
				Phase:       ecosystem.CIPhaseInstall,
			},
		}
	}

	return []ecosystem.CICommand{
		{
			Name:        "cabal-build",
			Command:     "cabal build",
			Description: "Build Haskell project with Cabal",
			Phase:       ecosystem.CIPhaseInstall,
		},
	}
}

// PackageManagers returns metadata about the Haskell package managers.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:     "cabal",
			LockFile: "cabal.project.freeze",
		},
		{
			Name:     "stack",
			LockFile: "stack.yaml.lock",
		},
	}
}

// WizardFields returns additional wizard form fields for Haskell configuration.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         "build_tool",
			Label:       "Build tool",
			Description: "Select the Haskell build tool for this project",
			Type:        ecosystem.FieldTypeSelect,
			Options: []ecosystem.WizardOption{
				{Label: "Cabal", Value: "cabal"},
				{Label: "Stack", Value: "stack"},
			},
			Default: "cabal",
		},
	}
}

// VerificationCommands returns build and test commands for Haskell projects,
// switching on the configured build tool (cabal or stack).
func (m *Module) VerificationCommands(config ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	switch config.Extra("build_tool", "cabal") {
	case "stack":
		return ecosystem.VerificationCommands{
			Build: []string{"stack build"},
			Test:  []string{"stack test"},
		}
	default:
		return ecosystem.VerificationCommands{
			Build: []string{"cabal build"},
			Test:  []string{"cabal test"},
		}
	}
}

// ManifestFiles returns the *.cabal manifest file for Haskell projects.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return []ecosystem.ManifestFileInfo{{Path: "*.cabal", Ecosystem: "cabal", LockFile: "cabal.project.freeze", LockFilePolicy: ecosystem.LockFilePolicyRecommended}}
}
