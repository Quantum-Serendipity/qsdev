// Package clojure implements the Clojure ecosystem module for
// qsdev. It detects Clojure projects by scanning for
// deps.edn (tools.deps) and project.clj (Leiningen), generates devenv.nix
// fragments with a warning about the lack of lockfile support, and provides
// pre-commit hooks, CI commands, wizard fields, and package manager metadata
// for the Clojure toolchain.
package clojure

import (
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

// Module implements ecosystem.EcosystemModule for the Clojure programming language.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "clojure" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Clojure" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 3 }

// Detect scans projectRoot for deps.edn and project.clj files.
// It determines the build tool and stores it in Extras["build_tool"].
// tools-deps is preferred when both files are present.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	hasDepsEdn := fileutil.FileExists(projectRoot, "deps.edn")
	hasProjectClj := fileutil.FileExists(projectRoot, "project.clj")

	if !hasDepsEdn && !hasProjectClj {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	var evidence []string
	extras := make(map[string]string)

	if hasDepsEdn {
		evidence = append(evidence, "deps.edn found")
	}
	if hasProjectClj {
		evidence = append(evidence, "project.clj found")
	}

	// Determine build tool. Prefer tools-deps if both are present.
	switch {
	case hasDepsEdn:
		extras["build_tool"] = "tools-deps"
	default:
		extras["build_tool"] = "leiningen"
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
// for Clojure language support. Includes a prominent warning about the lack
// of lockfile support in the Clojure ecosystem.
func (m *Module) DevenvNixFragment(_ ecosystem.ModuleConfig) (string, error) {
	var b strings.Builder
	b.WriteString("  languages.clojure.enable = true;\n")
	b.WriteString("  # WARNING: Clojure (tools.deps and Leiningen) has no lockfile support.\n")
	b.WriteString("  # Dependency versions are pinned in deps.edn / project.clj but content\n")
	b.WriteString("  # hashes are not verified. Consider using clj-watson or lein-nvd for\n")
	b.WriteString("  # vulnerability scanning.\n")
	return b.String(), nil
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
// Clojure does not require any additional inputs.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return nil
}

// SecurityConfigs returns generated security configuration files.
// Clojure does not produce additional security config files.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// PreCommitHooks returns pre-commit hook definitions for the Clojure ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "cljfmt",
			Name:          "cljfmt",
			Description:   "Check Clojure code formatting with cljfmt",
			Entry:         "cljfmt check",
			Language:      "system",
			Types:         []string{"clojure"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       false,
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the Clojure ecosystem.
// Clojure uses config-file based dependency management, so no deny rules
// are needed.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return nil
}

// CICommands returns CI pipeline commands for the Clojure ecosystem.
// Commands vary based on the configured build tool.
func (m *Module) CICommands(config ecosystem.ModuleConfig) []ecosystem.CICommand {
	buildTool := resolveBuildTool(config)

	if buildTool == "leiningen" {
		return []ecosystem.CICommand{
			{
				Name:        "lein-nvd-check",
				Command:     "lein nvd check",
				Description: "Scan Leiningen dependencies for known vulnerabilities",
				Phase:       ecosystem.CIPhaseScan,
			},
		}
	}

	// Default: tools-deps
	return []ecosystem.CICommand{
		{
			Name:        "clj-watson-scan",
			Command:     "clojure -Tclj-watson scan",
			Description: "Scan tools.deps dependencies for known vulnerabilities",
			Phase:       ecosystem.CIPhaseScan,
		},
	}
}

// PackageManagers returns metadata about the Clojure package managers.
// Neither tools.deps nor Leiningen support lockfiles.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:             "tools-deps",
			LockFile:         "",
			AgeGatingSupport: false,
		},
		{
			Name:             "leiningen",
			LockFile:         "",
			AgeGatingSupport: false,
		},
	}
}

// WizardFields returns additional wizard form fields for Clojure configuration.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         "clojure_build_tool",
			Label:       "Build tool",
			Description: "Select the Clojure build tool for this project",
			Type:        ecosystem.FieldTypeSelect,
			Options: []ecosystem.WizardOption{
				{Label: "tools.deps", Value: "tools-deps"},
				{Label: "Leiningen", Value: "leiningen"},
			},
			Default: "tools-deps",
		},
	}
}

// VerificationCommands returns an empty set. Clojure does not define standard
// verification commands at the module level.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{}
}

// ManifestFiles returns nil. Clojure does not use a traditional manifest file.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return nil
}

// resolveBuildTool reads the build_tool from config.Extras, defaulting to "tools-deps".
func resolveBuildTool(config ecosystem.ModuleConfig) string {
	if config.Extras != nil {
		if bt, ok := config.Extras["build_tool"]; ok && bt != "" {
			return bt
		}
	}
	return "tools-deps"
}

