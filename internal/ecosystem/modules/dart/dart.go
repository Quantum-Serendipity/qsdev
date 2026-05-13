// Package dart implements the Dart/Flutter ecosystem module for
// gdev-secure-devenv-bootstrap. It detects Dart and Flutter projects by
// scanning for pubspec.yaml and pubspec.lock, generates devenv.nix fragments
// with optional Flutter support, and provides pre-commit hooks, CI commands,
// deny rules, and wizard fields for the Dart toolchain.
package dart

import (
	"os"
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

// Module implements ecosystem.EcosystemModule for the Dart/Flutter ecosystem.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "dart" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Dart/Flutter" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 3 }

// Detect scans projectRoot for pubspec.yaml and pubspec.lock files.
// If pubspec.yaml contains a "flutter:" section, Flutter is detected and
// recorded in Extras["flutter"].
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	hasPubspec := fileutil.FileExists(projectRoot, "pubspec.yaml")
	hasPubspecLock := fileutil.FileExists(projectRoot, "pubspec.lock")

	if !hasPubspec && !hasPubspecLock {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	confidence := ecosystem.ConfidenceProbable
	var evidence []string
	extras := make(map[string]string)

	if hasPubspec {
		confidence = ecosystem.ConfidenceCertain
		evidence = append(evidence, "pubspec.yaml found")

		// Detect Flutter by checking for "flutter:" as a top-level YAML key
		// (at the start of a line with no indentation). A bare Contains would
		// match comments and nested keys in unrelated contexts.
		data, err := os.ReadFile(filepath.Join(projectRoot, "pubspec.yaml"))
		content := string(data)
		if err == nil && (strings.HasPrefix(content, "flutter:") || strings.Contains(content, "\nflutter:")) {
			extras["flutter"] = "true"
			evidence = append(evidence, "Flutter dependency detected in pubspec.yaml")
		} else {
			extras["flutter"] = "false"
		}
	}
	if hasPubspecLock {
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		evidence = append(evidence, "pubspec.lock found")
	}

	return ecosystem.DetectionResult{
		Detected:   true,
		Confidence: confidence,
		Evidence:   evidence,
		SuggestedConfig: ecosystem.ModuleConfig{
			Extras: extras,
		},
	}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Dart language support. When Flutter is detected, the Flutter package
// is added as well.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	var b strings.Builder
	b.WriteString("  languages.dart.enable = true;\n")

	if config.Extras["flutter"] == "true" {
		b.WriteString("  packages = [ pkgs.flutter ];\n")
	}

	return b.String(), nil
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
// Dart does not require any additional inputs.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return nil
}

// SecurityConfigs returns generated security configuration files.
// pubspec.lock contains SHA256 hashes; no additional config files are needed.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// PreCommitHooks returns pre-commit hook definitions for the Dart ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "dart-format",
			Name:          "dart-format",
			Description:   "Check Dart code formatting with dart format",
			Entry:         "dart format --set-exit-if-changed",
			Language:      "system",
			Types:         []string{"dart"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       true,
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the Dart ecosystem.
// These prevent direct dependency additions outside of controlled workflows.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"Bash(dart pub add *)",
		"Bash(flutter pub add *)",
	}
}

// CICommands returns CI pipeline commands for the Dart ecosystem.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "dart-pub-get-locked",
			Command:     "dart pub get --enforce-lockfile",
			Description: "Install Dart dependencies with lockfile enforcement",
			Phase:       ecosystem.CIPhaseInstall,
		},
		{
			Name:        "dart-pub-outdated",
			Command:     "dart pub outdated",
			Description: "Check for outdated Dart dependencies",
			Phase:       ecosystem.CIPhaseScan,
		},
	}
}

// PackageManagers returns metadata about the Dart pub package manager.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:                 "pub",
			LockFile:             "pubspec.lock",
			FrozenInstallCommand: "dart pub get --enforce-lockfile",
			AgeGatingSupport:     false,
		},
	}
}

// WizardFields returns additional wizard form fields for Dart configuration.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         "dart_flutter",
			Label:       "Flutter support",
			Description: "Enable Flutter SDK alongside Dart",
			Type:        ecosystem.FieldTypeConfirm,
			Default:     "false",
		},
	}
}

// VerificationCommands returns test, lint, and format commands for Dart projects.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{
		Test:   []string{"dart test"},
		Lint:   []string{"dart analyze"},
		Format: []string{"dart format --set-exit-if-changed ."},
	}
}

// ManifestFiles returns the pubspec.yaml manifest file for Dart projects.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return []ecosystem.ManifestFileInfo{{Path: "pubspec.yaml", Ecosystem: "pub", LockFile: "pubspec.lock", LockFilePolicy: ecosystem.LockFilePolicyRequired}}
}

