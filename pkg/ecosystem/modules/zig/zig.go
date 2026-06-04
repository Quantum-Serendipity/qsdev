// Package zig implements the Zig ecosystem module for
// qsdev. It detects Zig projects by scanning for
// build.zig and build.zig.zon, generates devenv.nix fragments with Zig
// language support, and provides pre-commit hooks, CI commands, and package
// manager metadata for the Zig toolchain.
//
// Security model: Zig has an EXEMPLARY content-addressed dependency model.
// Every dependency in build.zig.zon requires a mandatory SHA256 hash.
// Mutable references are impossible by design — if a dependency's content
// changes, the hash will not match and the build will fail. This is the
// strongest integrity model of any language ecosystem after Nix itself.
// No deny rules are needed because the content-addressed model inherently
// prevents supply chain attacks.
package zig

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

// Module implements ecosystem.EcosystemModule for the Zig programming language.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "zig" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Zig" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 4 }

// Detect scans projectRoot for Zig ecosystem indicators: build.zig (certain)
// and build.zig.zon (certain — the dependency manifest with SHA256 hashes).
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	var (
		evidence   []string
		confidence = ecosystem.ConfidenceAbsent
		detected   bool
	)

	// Certain indicators.
	if fileutil.FileExists(projectRoot, "build.zig") {
		evidence = append(evidence, "build.zig found")
		confidence = ecosystem.ConfidenceCertain
		detected = true
	}
	if fileutil.FileExists(projectRoot, "build.zig.zon") {
		evidence = append(evidence, "build.zig.zon found (dependency manifest with SHA256 hashes)")
		confidence = ecosystem.ConfidenceCertain
		detected = true
	}

	if !detected {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	return ecosystem.DetectionResult{
		Detected:   true,
		Confidence: confidence,
		Evidence:   evidence,
	}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Zig language support.
func (m *Module) DevenvNixFragment(_ ecosystem.ModuleConfig) (string, error) {
	return "  languages.zig.enable = true;\n", nil
}

// SecurityConfigs returns generated security configuration files.
// Zig's content-addressed build system (mandatory SHA256 hashes in
// build.zig.zon) provides integrity by design, so no additional security
// configuration is needed.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// PreCommitHooks returns pre-commit hook definitions for the Zig ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "zig-fmt",
			Name:          "zig-fmt",
			Description:   "Check Zig source formatting with zig fmt",
			Entry:         "zig fmt --check",
			Language:      "system",
			Types:         []string{"zig"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       true,
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the Zig ecosystem.
// Zig's content-addressed model (mandatory SHA256 hashes) prevents supply
// chain attacks by design, so no deny rules are needed.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return nil
}

// CICommands returns CI pipeline commands for the Zig ecosystem.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "zig-build",
			Command:     "zig build",
			Description: "Build the Zig project (validates dependency hashes)",
			Phase:       ecosystem.CIPhaseTest,
		},
	}
}

// PackageManagers returns metadata about Zig's build system.
// build.zig.zon contains inline SHA256 hashes and serves as both the
// dependency manifest and the lockfile.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:             "zig-build",
			LockFile:         "build.zig.zon",
			AgeGatingSupport: false,
		},
	}
}

// WizardFields returns additional wizard form fields for Zig configuration.
// Zig does not require any wizard fields.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return nil
}

// VerificationCommands returns an empty set. Zig does not define standard
// verification commands at the module level.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{}
}

// ManifestFiles returns nil. Zig does not use a traditional manifest file.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return nil
}
