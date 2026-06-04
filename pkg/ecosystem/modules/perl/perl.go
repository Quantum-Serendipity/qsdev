// Package perl implements the Perl (CPAN/Carton) ecosystem module for
// qsdev. It detects Perl projects by scanning for
// cpanfile, Makefile.PL, Build.PL, and cpanfile.snapshot, generates devenv.nix
// fragments with Perl language support, and provides pre-commit hooks, CI
// commands, deny rules, and package manager metadata for the Perl toolchain.
//
// Security limitations: CPAN has no package signing and no age-gating.
// Perl supply-chain security relies entirely on cpanfile + cpanfile.snapshot
// pinning via Carton. There is no built-in vulnerability audit database
// comparable to npm audit or cargo-audit; cpan-audit provides partial coverage.
package perl

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

// Module implements ecosystem.EcosystemModule for the Perl programming language.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "perl" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Perl" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 4 }

// Detect scans projectRoot for Perl ecosystem indicators: cpanfile (certain),
// Makefile.PL (probable), Build.PL (probable), and cpanfile.snapshot (probable,
// sets package manager to carton).
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	var (
		evidence   []string
		confidence = ecosystem.ConfidenceAbsent
		detected   bool
		pm         string
	)

	// Certain indicator.
	if fileutil.FileExists(projectRoot, "cpanfile") {
		evidence = append(evidence, "cpanfile found")
		confidence = ecosystem.ConfidenceCertain
		detected = true
	}

	// Probable indicators.
	if fileutil.FileExists(projectRoot, "Makefile.PL") {
		evidence = append(evidence, "Makefile.PL found")
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		detected = true
	}
	if fileutil.FileExists(projectRoot, "Build.PL") {
		evidence = append(evidence, "Build.PL found")
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		detected = true
	}
	if fileutil.FileExists(projectRoot, "cpanfile.snapshot") {
		evidence = append(evidence, "cpanfile.snapshot found")
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		detected = true
		pm = "carton"
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
// for Perl language support.
func (m *Module) DevenvNixFragment(_ ecosystem.ModuleConfig) (string, error) {
	return "  languages.perl.enable = true;\n", nil
}

// SecurityConfigs returns generated security configuration files.
// CPAN has no signing mechanism, so no security configuration files are generated.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// PreCommitHooks returns pre-commit hook definitions for the Perl ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "perltidy",
			Name:          "perltidy",
			Description:   "Check Perl source formatting with perltidy",
			Entry:         "perltidy --check",
			Language:      "system",
			Types:         []string{"perl"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       false,
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the Perl ecosystem.
// These prevent direct CPAN module installation outside of controlled workflows.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"Bash(cpan install *)",
		"Bash(cpanm *)",
	}
}

// CICommands returns CI pipeline commands for the Perl ecosystem.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "carton-install",
			Command:     "carton install --deployment",
			Description: "Install Perl dependencies from cpanfile.snapshot in deployment mode",
			Phase:       ecosystem.CIPhaseInstall,
		},
		{
			Name:        "cpan-audit",
			Command:     "cpan-audit installed",
			Description: "Audit installed Perl modules for known vulnerabilities",
			Phase:       ecosystem.CIPhaseScan,
		},
	}
}

// PackageManagers returns metadata about the Perl Carton package manager.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:                 "carton",
			LockFile:             "cpanfile.snapshot",
			FrozenInstallCommand: "carton install --deployment",
			AuditCommand:         "cpan-audit installed",
			AgeGatingSupport:     false,
		},
	}
}

// WizardFields returns additional wizard form fields for Perl configuration.
// Perl does not require any wizard fields.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return nil
}

// VerificationCommands returns an empty set. Perl does not define standard
// verification commands at the module level.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{}
}

// ManifestFiles returns the cpanfile manifest for Perl projects.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return []ecosystem.ManifestFileInfo{
		{
			Path:           "cpanfile",
			Ecosystem:      "cpan",
			LockFile:       "cpanfile.snapshot",
			LockFilePolicy: ecosystem.LockFilePolicyRecommended,
		},
	}
}
