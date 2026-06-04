// Package nixlang implements the Nix ecosystem module for
// qsdev. It detects Nix projects by scanning for
// flake.nix, flake.lock, default.nix, and shell.nix, generates devenv.nix
// fragments with Nix language support, and provides pre-commit hooks (statix,
// deadnix, nixfmt), CI commands, deny rules, and package manager metadata for
// the Nix ecosystem.
package nixlang

import (
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Module implements ecosystem.EcosystemModule for the Nix ecosystem.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "nix" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Nix" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 3 }

// Detect scans projectRoot for Nix ecosystem indicators: flake.nix,
// flake.lock, default.nix, and shell.nix.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	hasFlakeNix := fileutil.FileExists(projectRoot, "flake.nix")
	hasFlakeLock := fileutil.FileExists(projectRoot, "flake.lock")
	hasDefaultNix := fileutil.FileExists(projectRoot, "default.nix")
	hasShellNix := fileutil.FileExists(projectRoot, "shell.nix")

	if !hasFlakeNix && !hasFlakeLock && !hasDefaultNix && !hasShellNix {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	confidence := ecosystem.ConfidenceProbable
	var evidence []string

	if hasFlakeNix {
		confidence = ecosystem.ConfidenceCertain
		evidence = append(evidence, "flake.nix found")
	}
	if hasFlakeLock {
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		evidence = append(evidence, "flake.lock found")
	}
	if hasDefaultNix {
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		evidence = append(evidence, "default.nix found")
	}
	if hasShellNix {
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		evidence = append(evidence, "shell.nix found")
	}

	return ecosystem.DetectionResult{
		Detected:   true,
		Confidence: confidence,
		Evidence:   evidence,
	}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Nix language support.
func (m *Module) DevenvNixFragment(_ ecosystem.ModuleConfig) (string, error) {
	return "  languages.nix.enable = true;\n", nil
}

// SecurityConfigs returns generated security configuration files.
// nix.conf hardening is handled at the system level; no additional config needed.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// PreCommitHooks returns pre-commit hook definitions for the Nix ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "statix",
			Name:          "statix",
			Description:   "Lint Nix code with statix",
			Entry:         "statix check",
			Language:      "system",
			Types:         []string{"nix"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       false,
		},
		{
			ID:            "deadnix",
			Name:          "deadnix",
			Description:   "Find dead code in Nix files with deadnix",
			Entry:         "deadnix --fail",
			Language:      "system",
			Types:         []string{"nix"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       false,
		},
		{
			ID:            "nixfmt",
			Name:          "nixfmt",
			Description:   "Check Nix code formatting with nixfmt",
			Entry:         "nixfmt --check",
			Language:      "system",
			Types:         []string{"nix"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			Files:         `\.nix$`,
			BuiltIn:       true,
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the Nix ecosystem.
// Prevents imperative package installations that bypass the declarative model.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"Bash(nix-env -i *)",
	}
}

// CICommands returns CI pipeline commands for the Nix ecosystem.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "nix-flake-check",
			Command:     "nix flake check",
			Description: "Run Nix flake checks",
			Phase:       ecosystem.CIPhaseTest,
		},
		{
			Name:        "nix-flake-lock-drift",
			Command:     "nix flake lock --update-input nixpkgs && git diff --exit-code flake.lock",
			Description: "Detect nixpkgs input drift in flake.lock",
			Phase:       ecosystem.CIPhaseScan,
		},
	}
}

// PackageManagers returns metadata about the Nix flake package manager.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:             "nix-flake",
			LockFile:         "flake.lock",
			InstallCommand:   "nix develop",
			AgeGatingSupport: false,
		},
	}
}

// WizardFields returns additional wizard form fields for Nix configuration.
// Nix does not require any additional wizard fields.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return nil
}

// VerificationCommands returns an empty set. Nix does not define standard
// verification commands at the module level.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{}
}

// ManifestFiles returns nil. Nix does not use a traditional manifest file.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return nil
}
