// Package shell implements the Bash/Shell ecosystem module for qsdev.
// It detects shell script projects by scanning for *.sh files in the project root,
// scripts/ directories, and .envrc files, then generates devenv.nix fragments with
// shellcheck and shfmt packages, pre-commit hooks, deny rules against pipe-to-shell
// patterns, and CI commands for a hardened shell scripting environment.
package shell

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*Module)(nil)

func init() {
	ecosystem.RegisterModule(&Module{})
}

// Module is the stateless Bash/Shell ecosystem module.
type Module struct{}

// Name returns the canonical module identifier.
func (m *Module) Name() string { return "shell" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Bash/Shell" }

// Tier returns the implementation priority tier (2 = standard).
func (m *Module) Tier() int { return 2 }

// Detect scans projectRoot for shell script indicators.
// *.sh files in the root and scripts/ directories yield Probable confidence.
// .envrc is recorded as evidence but does not boost confidence.
// Maximum confidence is Probable since shell scripts are ubiquitous.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	result := ecosystem.DetectionResult{}

	// Glob *.sh in root directory.
	shFiles, _ := filepath.Glob(filepath.Join(projectRoot, "*.sh"))
	if len(shFiles) > 0 {
		result.Detected = true
		result.Confidence = ecosystem.ConfidenceProbable
		result.Evidence = append(result.Evidence, fmt.Sprintf("%d .sh file(s) in root", len(shFiles)))
	}

	// Check for scripts/ directory.
	if fileutil.DirExists(projectRoot, "scripts") {
		result.Evidence = append(result.Evidence, "scripts/ directory found")
		if !result.Detected {
			result.Detected = true
			result.Confidence = ecosystem.ConfidenceProbable
		}
	}

	// Check for .envrc (evidence only, no confidence boost).
	if fileutil.FileExists(projectRoot, ".envrc") {
		result.Evidence = append(result.Evidence, ".envrc found")
	}

	return result
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for shell scripting support. Uses a packages-based approach.
func (m *Module) DevenvNixFragment(_ ecosystem.ModuleConfig) (string, error) {
	var b strings.Builder
	b.WriteString("  packages = with pkgs; [ shellcheck shfmt ];\n")
	return b.String(), nil
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
// Shell does not require any additional inputs.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return nil
}

// SecurityConfigs returns nil. Shell has no package manager that requires
// security configuration.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// PreCommitHooks returns pre-commit hook definitions for the Shell ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "shellcheck",
			Name:          "shellcheck",
			Description:   "Lint shell scripts with shellcheck",
			Entry:         "shellcheck",
			Language:      "system",
			Types:         []string{"shell"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       true,
		},
		{
			ID:            "shfmt",
			Name:          "shfmt",
			Description:   "Check shell script formatting with shfmt",
			Entry:         "shfmt -d",
			Language:      "system",
			Types:         []string{"shell"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       true,
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the Shell ecosystem.
// These prevent pipe-to-shell execution patterns which are common attack vectors.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"Bash(curl * | sh*)",
		"Bash(curl * | bash*)",
		"Bash(wget * | sh*)",
		"Bash(wget * | bash*)",
	}
}

// CICommands returns CI pipeline commands for the Shell ecosystem.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "shellcheck",
			Command:     "find . -name '*.sh' -type f -exec shellcheck {} +",
			Description: "Lint all shell scripts with shellcheck",
			Phase:       ecosystem.CIPhaseScan,
		},
		{
			Name:        "bash-syntax-check",
			Command:     "find . -name '*.sh' -type f -exec bash -n {} +",
			Description: "Check shell script syntax with bash -n",
			Phase:       ecosystem.CIPhaseTest,
		},
	}
}

// PackageManagers returns nil. Shell scripts have no package manager.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return nil
}

// WizardFields returns nil. Shell does not require additional wizard fields.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return nil
}

// VerificationCommands returns an empty set. Shell does not define standard
// verification commands at the module level.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{}
}

// ManifestFiles returns nil. Shell does not use a traditional manifest file.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return nil
}

