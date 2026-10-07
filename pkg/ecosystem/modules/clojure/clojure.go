// Package clojure implements the Clojure ecosystem module for
// qsdev. It detects Clojure projects by scanning for
// deps.edn (tools.deps) and project.clj (Leiningen), generates devenv.nix
// fragments with a warning about the lack of lockfile support, and provides
// pre-commit hooks, CI commands, wizard fields, and package manager metadata
// for the Clojure toolchain.
package clojure

import (
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.WizardFieldProvider = (*Module)(nil)
var _ ecosystem.PackageExprProvider = (*Module)(nil)
var _ ecosystem.SetupWarner = (*Module)(nil)

// Clojure build tool identifiers, as stored in Extras["build_tool"].
const (
	buildToolDeps      = "tools-deps"
	buildToolLeiningen = "leiningen"
)

// Extras keys Detect sets, to "true", when the project declares its build
// tool's vulnerability scanner. They are absent otherwise, so a later
// re-init merges them in once the project adds the scanner.
const (
	extraCljWatson = "clj_watson"
	extraLeinNVD   = "lein_nvd"
)

var (
	// cljWatsonAlias matches the :clj-watson keyword, the alias that
	// `clojure -M:clj-watson` runs, as a whole token.
	cljWatsonAlias = regexp.MustCompile(`(?:^|[\s,{\[(]):clj-watson(?:[\s,{}\[\]()]|$)`)
	// leinNVDPlugin matches the lein-nvd plugin symbol as a whole token.
	leinNVDPlugin = regexp.MustCompile(`(?:^|[\s,\[(/])lein-nvd(?:[\s,\[\]()]|$)`)
)

// declares reports whether the project's Clojure source file name declares
// token, ignoring comments and strings.
func declares(projectRoot, name string, token *regexp.Regexp) bool {
	return ecosystem.FileDeclares(filepath.Join(projectRoot, name), ';', token)
}

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
// tools-deps is preferred when both files are present. It records whether
// deps.edn declares a :clj-watson alias (Extras clj_watson) and whether
// project.clj uses the lein-nvd plugin (Extras lein_nvd).
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
		if declares(projectRoot, "deps.edn", cljWatsonAlias) {
			extras[extraCljWatson] = "true"
		}
	}
	if hasProjectClj {
		evidence = append(evidence, "project.clj found")
		if declares(projectRoot, "project.clj", leinNVDPlugin) {
			extras[extraLeinNVD] = "true"
		}
	}

	// Determine build tool. Prefer tools-deps if both are present.
	switch {
	case hasDepsEdn:
		extras["build_tool"] = buildToolDeps
	default:
		extras["build_tool"] = buildToolLeiningen
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

// DevenvPackageExprs returns Leiningen for Leiningen projects. devenv's
// languages.clojure provides only the clojure (tools.deps) CLI, so without it
// a project.clj project has no lein. It is built against the project JDK
// (languages.java.jdk.package), as devenv builds the clojure CLI.
func (m *Module) DevenvPackageExprs(config ecosystem.ModuleConfig) []string {
	if config.Extra("build_tool", buildToolDeps) != buildToolLeiningen {
		return nil
	}
	return []string{"(pkgs.leiningen.override { jdk = config.languages.java.jdk.package; })"}
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
			// Custom hook (BuiltIn:false): NixPackage provisions the binary so
			// the emitted `entry` resolves at commit time.
			BuiltIn:    false,
			NixPackage: "cljfmt",
		},
	}
}

// CICommands returns CI pipeline commands for the Clojure ecosystem: the
// build tool's vulnerability scan, only when the project declares the
// scanner (see Detect). Neither scanner is in nixpkgs and both resolve
// through the project's own configuration: the :clj-watson alias, or the
// lein-nvd plugin. A -T tool invocation is never emitted, since it would
// resolve a tool the project never pinned.
func (m *Module) CICommands(config ecosystem.ModuleConfig) []ecosystem.CICommand {
	if config.Extra("build_tool", buildToolDeps) == buildToolLeiningen {
		if config.Extra(extraLeinNVD, "") != "true" {
			return nil
		}
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
	if config.Extra(extraCljWatson, "") != "true" {
		return nil
	}
	return []ecosystem.CICommand{
		{
			Name:        "clj-watson-scan",
			Command:     "clojure -M:clj-watson scan -p deps.edn",
			Description: "Scan tools.deps dependencies for known vulnerabilities",
			Phase:       ecosystem.CIPhaseScan,
		},
	}
}

// SetupWarnings reports a project that does not declare its build tool's
// vulnerability scanner, so CICommands emits no scan step, and names the
// snippet to add. The project files are read as well as config, since an
// answers file saved before the project added the scanner lacks the extra
// that generation fills in from detection.
func (m *Module) SetupWarnings(projectRoot string, config ecosystem.ModuleConfig) []string {
	if config.Extra("build_tool", buildToolDeps) == buildToolLeiningen {
		if config.Extra(extraLeinNVD, "") == "true" || declares(projectRoot, "project.clj", leinNVDPlugin) {
			return nil
		}
		return []string{"Clojure dependency security scan not run: add the lein-nvd plugin to project.clj " +
			"(:plugins [[lein-nvd \"<version>\"]]), then run `qsdev init --update`"}
	}
	if config.Extra(extraCljWatson, "") == "true" || declares(projectRoot, "deps.edn", cljWatsonAlias) {
		return nil
	}
	return []string{"Clojure dependency security scan not run: add a :clj-watson alias to deps.edn " +
		"(:aliases {:clj-watson {:replace-deps {io.github.clj-holmes/clj-watson {:git/tag \"<tag>\" :git/sha \"<sha>\"}} " +
		":main-opts [\"-m\" \"clj-watson.cli\"]}}), then run `qsdev init --update`"}
}

// PackageManagers returns metadata about the Clojure package managers.
// Neither tools.deps nor Leiningen support lockfiles.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:     "tools-deps",
			LockFile: "",
		},
		{
			Name:     "leiningen",
			LockFile: "",
		},
	}
}

// WizardFields returns additional wizard form fields for Clojure configuration.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         "build_tool",
			Label:       "Build tool",
			Description: "Select the Clojure build tool for this project",
			Type:        ecosystem.FieldTypeSelect,
			Options: []ecosystem.WizardOption{
				{Label: "tools.deps", Value: buildToolDeps},
				{Label: "Leiningen", Value: buildToolLeiningen},
			},
			Default: buildToolDeps,
		},
	}
}

// VerificationCommands returns an empty set. Clojure does not define standard
// verification commands at the module level.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{}
}
