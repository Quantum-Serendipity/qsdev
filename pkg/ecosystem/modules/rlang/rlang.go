// Package rlang implements the R (renv) ecosystem module for
// qsdev. It detects R projects by scanning for
// DESCRIPTION, renv.lock, .Rprofile, and R/Rmd source files, generates
// devenv.nix fragments with R language support, and provides CI commands
// and package manager metadata for the R toolchain.
//
// Security limitations: CRAN has no package signing. The renv.lock file
// provides reproducibility (pinned versions + repositories) but not
// cryptographic integrity verification. There is no built-in vulnerability
// database for R packages. Security relies on renv.lock pinning and
// repository trust.
package rlang

import (
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*Module)(nil)

func init() {
	ecosystem.RegisterModule(&Module{})
}

// Module implements ecosystem.EcosystemModule for the R programming language.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "r" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "R" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 4 }

// Detect scans projectRoot for R ecosystem indicators: renv.lock (certain, sets
// PM to renv), DESCRIPTION (probable alone, certain when combined with NAMESPACE,
// *.Rproj, or renv.lock), .Rprofile (probable), *.R (probable), *.Rmd (probable).
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	var (
		evidence   []string
		confidence = ecosystem.ConfidenceAbsent
		detected   bool
		pm         string
	)

	// Strong R indicators (certain on their own).
	hasNamespace := fileutil.FileExists(projectRoot, "NAMESPACE")
	rprojMatches, _ := filepath.Glob(filepath.Join(projectRoot, "*.Rproj"))
	hasRproj := len(rprojMatches) > 0
	hasRenvLock := fileutil.FileExists(projectRoot, "renv.lock")

	if hasRenvLock {
		evidence = append(evidence, "renv.lock found")
		confidence = ecosystem.ConfidenceCertain
		detected = true
		pm = "renv"
	}

	// DESCRIPTION alone is only probable — many non-R projects (Debian, Perl)
	// use DESCRIPTION files. Upgrade to certain when combined with other R
	// indicators.
	hasDescription := fileutil.FileExists(projectRoot, "DESCRIPTION")
	if hasDescription {
		evidence = append(evidence, "DESCRIPTION found")
		if hasNamespace || hasRproj || hasRenvLock {
			confidence = ecosystem.ConfidenceCertain
		} else if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		detected = true
	}
	if hasNamespace {
		evidence = append(evidence, "NAMESPACE found")
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		detected = true
	}
	if hasRproj {
		evidence = append(evidence, "*.Rproj file found")
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		detected = true
	}

	// Probable indicators.
	if fileutil.FileExists(projectRoot, ".Rprofile") {
		evidence = append(evidence, ".Rprofile found")
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		detected = true
	}
	if rFiles, _ := filepath.Glob(filepath.Join(projectRoot, "*.R")); len(rFiles) > 0 {
		evidence = append(evidence, "*.R files found")
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		detected = true
	}
	if rmdFiles, _ := filepath.Glob(filepath.Join(projectRoot, "*.Rmd")); len(rmdFiles) > 0 {
		evidence = append(evidence, "*.Rmd files found")
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
		SuggestedConfig: ecosystem.ModuleConfig{
			PackageManager: pm,
		},
	}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for R language support.
func (m *Module) DevenvNixFragment(_ ecosystem.ModuleConfig) (string, error) {
	return "  languages.r.enable = true;\n", nil
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
// R does not require any additional inputs.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return nil
}

// SecurityConfigs returns generated security configuration files.
// CRAN has no signing mechanism, so no security configuration files are generated.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// PreCommitHooks returns pre-commit hook definitions for the R ecosystem.
// R formatting/linting tools (styler, lintr) are R packages, not standalone
// CLIs, so no pre-commit hooks are provided.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return nil
}

// DenyRules returns Claude Code deny-rule patterns for the R ecosystem.
// In data science contexts, direct package installation is expected workflow,
// so no deny rules are applied.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return nil
}

// CICommands returns CI pipeline commands for the R ecosystem.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "renv-restore",
			Command:     `Rscript -e "renv::restore()"`,
			Description: "Restore R package dependencies from renv.lock",
			Phase:       ecosystem.CIPhaseInstall,
		},
		{
			Name:        "renv-status",
			Command:     `Rscript -e "renv::status()"`,
			Description: "Check renv lock file consistency",
			Phase:       ecosystem.CIPhaseTest,
		},
	}
}

// PackageManagers returns metadata about R's renv package manager.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:                 "renv",
			LockFile:             "renv.lock",
			InstallCommand:       `Rscript -e "renv::restore()"`,
			FrozenInstallCommand: `Rscript -e "renv::restore()"`,
			AgeGatingSupport:     false,
		},
	}
}

// WizardFields returns additional wizard form fields for R configuration.
// R does not require any wizard fields.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return nil
}

// VerificationCommands returns an empty set. R does not define standard
// verification commands at the module level.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{}
}

// ManifestFiles returns nil. R does not use a traditional manifest file.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return nil
}

