// Package powershell implements the PowerShell ecosystem module for
// gdev-secure-devenv-bootstrap. It detects PowerShell projects by scanning for
// requirements.psd1 and PowerShell script files, generates devenv.nix fragments
// with the PowerShell package, and provides CI commands, deny rules, and package
// manager metadata for the PowerShell toolchain.
//
// Security limitations: PSGallery (the primary PowerShell module repository) has
// no age-gating, no install-script blocking, and limited signing enforcement.
// While PowerShell supports Authenticode signatures, PSGallery does not require
// modules to be signed, and Install-Module does not verify signatures by default.
// The requirements.psd1 manifest provides version pinning but no integrity
// verification.
package powershell

import (
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/fileutil"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*Module)(nil)

func init() {
	ecosystem.RegisterModule(&Module{})
}

// Module implements ecosystem.EcosystemModule for the PowerShell scripting language.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "powershell" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "PowerShell" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 4 }

// Detect scans projectRoot for PowerShell ecosystem indicators:
// requirements.psd1 (certain), *.ps1 (probable), and *.psm1 (probable).
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	var (
		evidence   []string
		confidence = ecosystem.ConfidenceAbsent
		detected   bool
	)

	// Certain indicator.
	if fileutil.FileExists(projectRoot, "requirements.psd1") {
		evidence = append(evidence, "requirements.psd1 found")
		confidence = ecosystem.ConfidenceCertain
		detected = true
	}

	// Probable indicators.
	if ps1Files, _ := filepath.Glob(filepath.Join(projectRoot, "*.ps1")); len(ps1Files) > 0 {
		evidence = append(evidence, "*.ps1 files found")
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		detected = true
	}
	if psmFiles, _ := filepath.Glob(filepath.Join(projectRoot, "*.psm1")); len(psmFiles) > 0 {
		evidence = append(evidence, "*.psm1 files found")
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
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
// for PowerShell support. PowerShell has no devenv.sh languages module, so
// the package is added directly.
func (m *Module) DevenvNixFragment(_ ecosystem.ModuleConfig) (string, error) {
	var b strings.Builder
	b.WriteString("  packages = with pkgs; [ powershell ];\n")
	return b.String(), nil
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
// PowerShell does not require any additional inputs.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return nil
}

// SecurityConfigs returns generated security configuration files.
// PSGallery has no age-gating and no install-script blocking, so no security
// configuration files are generated.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// PreCommitHooks returns pre-commit hook definitions for the PowerShell ecosystem.
// PSScriptAnalyzer is a PowerShell module, not a standalone CLI, so no
// pre-commit hooks are provided.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return nil
}

// DenyRules returns Claude Code deny-rule patterns for the PowerShell ecosystem.
// These prevent direct PSGallery module installation outside of controlled workflows.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"Bash(Install-Module *)",
		"Bash(pwsh -Command *Install-Module*)",
	}
}

// CICommands returns CI pipeline commands for the PowerShell ecosystem.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "psscriptanalyzer",
			Command:     `pwsh -Command "Invoke-ScriptAnalyzer -Path . -Recurse -Severity Error"`,
			Description: "Scan PowerShell scripts for issues with PSScriptAnalyzer",
			Phase:       ecosystem.CIPhaseScan,
		},
	}
}

// PackageManagers returns metadata about PowerShell's PSGallery package manager.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:             "psgallery",
			LockFile:         "",
			AgeGatingSupport: false,
		},
	}
}

// WizardFields returns additional wizard form fields for PowerShell configuration.
// PowerShell does not require any wizard fields.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return nil
}

