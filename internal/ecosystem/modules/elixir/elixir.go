// Package elixir implements the Elixir (Mix) ecosystem module for
// gdev-secure-devenv-bootstrap. It detects Elixir projects by scanning for
// mix.exs and mix.lock, generates devenv.nix fragments with Elixir language
// support, and provides pre-commit hooks, CI commands, deny rules, and package
// manager metadata for the Elixir toolchain.
package elixir

import (
	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/fileutil"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*Module)(nil)

func init() {
	ecosystem.RegisterModule(&Module{})
}

// Module implements ecosystem.EcosystemModule for the Elixir programming language.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "elixir" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Elixir" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 3 }

// Detect scans projectRoot for mix.exs and mix.lock files.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	hasMixExs := fileutil.FileExists(projectRoot, "mix.exs")
	hasMixLock := fileutil.FileExists(projectRoot, "mix.lock")

	if !hasMixExs && !hasMixLock {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	confidence := ecosystem.ConfidenceProbable
	var evidence []string

	if hasMixExs {
		confidence = ecosystem.ConfidenceCertain
		evidence = append(evidence, "mix.exs found")
	}
	if hasMixLock {
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		evidence = append(evidence, "mix.lock found")
	}

	return ecosystem.DetectionResult{
		Detected:   true,
		Confidence: confidence,
		Evidence:   evidence,
	}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Elixir language support.
func (m *Module) DevenvNixFragment(_ ecosystem.ModuleConfig) (string, error) {
	return "  languages.elixir.enable = true;\n", nil
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
// Elixir does not require any additional inputs.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return nil
}

// SecurityConfigs returns generated security configuration files.
// Elixir relies on mix.lock for integrity; no additional config files are needed.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// PreCommitHooks returns pre-commit hook definitions for the Elixir ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "mix-format",
			Name:          "mix-format",
			Description:   "Check Elixir code formatting with mix format",
			Entry:         "mix format --check-formatted",
			Language:      "system",
			Types:         []string{"elixir"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       true,
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the Elixir ecosystem.
// These prevent direct dependency fetching outside of controlled workflows.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"Bash(mix deps.get *)",
	}
}

// CICommands returns CI pipeline commands for the Elixir ecosystem.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "mix-deps-get-locked",
			Command:     "mix deps.get --check-locked",
			Description: "Install Elixir dependencies with lockfile verification",
			Phase:       ecosystem.CIPhaseInstall,
		},
		{
			Name:        "mix-deps-audit",
			Command:     "mix deps.audit",
			Description: "Audit Elixir dependencies for known vulnerabilities",
			Phase:       ecosystem.CIPhaseScan,
		},
	}
}

// PackageManagers returns metadata about the Elixir Mix package manager.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:                 "mix",
			LockFile:             "mix.lock",
			FrozenInstallCommand: "mix deps.get --check-locked",
			AuditCommand:         "mix deps.audit",
			AgeGatingSupport:     false,
		},
	}
}

// WizardFields returns additional wizard form fields for Elixir configuration.
// Elixir does not require any additional wizard fields.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return nil
}

