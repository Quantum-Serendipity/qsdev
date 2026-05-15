// Package haskell implements the Haskell ecosystem module for
// qsdev. It detects Haskell projects by scanning for
// *.cabal, stack.yaml, and cabal.project files, generates devenv.nix fragments
// with optional Stack support, and provides pre-commit hooks, CI commands,
// deny rules, wizard fields, and package manager metadata for the Haskell
// toolchain.
package haskell

import (
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/internal/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*Module)(nil)

func init() {
	ecosystem.RegisterModule(&Module{})
}

// Module implements ecosystem.EcosystemModule for the Haskell programming language.
type Module struct{}

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

	// Determine build tool: stack.yaml presence implies Stack.
	if hasStackYaml {
		extras["build_tool"] = "stack"
	} else {
		extras["build_tool"] = "cabal"
	}

	return ecosystem.DetectionResult{
		Detected:   true,
		Confidence: ecosystem.ConfidenceCertain,
		Evidence:   evidence,
		SuggestedConfig: ecosystem.ModuleConfig{
			Extras: extras,
		},
	}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Haskell language support. When Stack is the build tool, Stack integration
// is also enabled.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	buildTool := resolveBuildTool(config)

	var b strings.Builder
	b.WriteString("  languages.haskell.enable = true;\n")
	if buildTool == "stack" {
		b.WriteString("  languages.haskell.stack.enable = true;\n")
	}
	b.WriteString("  # NOTE: cabal.project.freeze is NOT a true lockfile — it pins\n")
	b.WriteString("  # versions but does not record content hashes.\n")
	return b.String(), nil
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
// Haskell does not require any additional inputs.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return nil
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
			BuiltIn:       false,
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
// Commands vary based on the configured build tool.
func (m *Module) CICommands(config ecosystem.ModuleConfig) []ecosystem.CICommand {
	buildTool := resolveBuildTool(config)

	if buildTool == "stack" {
		return []ecosystem.CICommand{
			{
				Name:        "stack-build-locked",
				Command:     "stack build --locked",
				Description: "Build Haskell project with Stack using locked dependencies",
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
			Name:             "cabal",
			LockFile:         "cabal.project.freeze",
			AgeGatingSupport: false,
		},
		{
			Name:             "stack",
			LockFile:         "stack.yaml.lock",
			AgeGatingSupport: false,
		},
	}
}

// WizardFields returns additional wizard form fields for Haskell configuration.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         "haskell_build_tool",
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

// VerificationCommands returns build and test commands for Haskell projects.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{
		Build: []string{"cabal build"},
		Test:  []string{"cabal test"},
	}
}

// ManifestFiles returns the *.cabal manifest file for Haskell projects.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return []ecosystem.ManifestFileInfo{{Path: "*.cabal", Ecosystem: "cabal", LockFile: "cabal.project.freeze", LockFilePolicy: ecosystem.LockFilePolicyRecommended}}
}

// resolveBuildTool reads the build_tool from config.Extras, defaulting to "cabal".
func resolveBuildTool(config ecosystem.ModuleConfig) string {
	if config.Extras != nil {
		if bt, ok := config.Extras["build_tool"]; ok && bt != "" {
			return bt
		}
	}
	return "cabal"
}

