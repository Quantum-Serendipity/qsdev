// Package swift implements the Swift (SPM) ecosystem module for
// qsdev. It detects Swift projects by scanning for
// Package.swift, Package.resolved, and *.xcodeproj, generates devenv.nix
// fragments with Swift language support and SE-0391 TOFU commentary, and
// provides pre-commit hooks, CI commands, deny rules, and package manager
// metadata for the Swift toolchain.
package swift

import (
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Module implements ecosystem.EcosystemModule for the Swift programming language.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "swift" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Swift" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 3 }

// Detect scans projectRoot for Swift ecosystem indicators: Package.swift,
// Package.resolved, and *.xcodeproj directories.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	hasPackageSwift := fileutil.FileExists(projectRoot, "Package.swift")
	hasPackageResolved := fileutil.FileExists(projectRoot, "Package.resolved")
	xcodeprojMatches, _ := filepath.Glob(filepath.Join(projectRoot, "*.xcodeproj"))
	hasXcodeproj := len(xcodeprojMatches) > 0

	if !hasPackageSwift && !hasPackageResolved && !hasXcodeproj {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	confidence := ecosystem.ConfidenceProbable
	var evidence []string

	if hasPackageSwift {
		confidence = ecosystem.ConfidenceCertain
		evidence = append(evidence, "Package.swift found")
	}
	if hasPackageResolved {
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		evidence = append(evidence, "Package.resolved found")
	}
	if hasXcodeproj {
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		evidence = append(evidence, "*.xcodeproj found")
	}

	return ecosystem.DetectionResult{
		Detected:   true,
		Confidence: confidence,
		Evidence:   evidence,
	}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Swift language support. Includes a comment about SE-0391 TOFU
// (Trust On First Use) for package integrity.
func (m *Module) DevenvNixFragment(_ ecosystem.ModuleConfig) (string, error) {
	var b strings.Builder
	b.WriteString("  languages.swift.enable = true;\n")
	b.WriteString("  # SE-0391: Package.resolved provides TOFU (Trust On First Use) integrity.\n")
	b.WriteString("  # Always commit Package.resolved to version control.\n")
	return b.String(), nil
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
// Swift does not require any additional inputs.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return nil
}

// SecurityConfigs returns generated security configuration files.
// Swift relies on Package.resolved for integrity; no additional config needed.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// PreCommitHooks returns pre-commit hook definitions for the Swift ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "swiftformat",
			Name:          "swiftformat",
			Description:   "Lint Swift source code with SwiftFormat",
			Entry:         "swiftformat --lint",
			Language:      "system",
			Types:         []string{"swift"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       false,
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the Swift ecosystem.
// These prevent direct dependency updates outside of controlled workflows.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"Bash(swift package update *)",
	}
}

// CICommands returns CI pipeline commands for the Swift ecosystem.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "swift-package-resolve",
			Command:     "swift package resolve",
			Description: "Resolve Swift package dependencies",
			Phase:       ecosystem.CIPhaseInstall,
		},
		{
			Name:        "swift-build",
			Command:     "swift build",
			Description: "Build the Swift project",
			Phase:       ecosystem.CIPhaseTest,
		},
	}
}

// PackageManagers returns metadata about the Swift Package Manager.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:             "spm",
			LockFile:         "Package.resolved",
			AgeGatingSupport: false,
		},
	}
}

// WizardFields returns additional wizard form fields for Swift configuration.
// Swift does not require any additional wizard fields.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return nil
}

// VerificationCommands returns build and test commands for Swift projects.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{
		Build: []string{"swift build"},
		Test:  []string{"swift test"},
	}
}

// ManifestFiles returns the Package.swift manifest file for Swift projects.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return []ecosystem.ManifestFileInfo{{Path: "Package.swift", Ecosystem: "spm", LockFile: "Package.resolved", LockFilePolicy: ecosystem.LockFilePolicyRecommended}}
}

