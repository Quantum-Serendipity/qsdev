// Package scala implements the Scala (sbt/Mill) ecosystem module for
// qsdev. It detects Scala projects by scanning for
// build.sbt, Mill build files, and sbt metadata under project/, generates
// devenv.nix fragments with the appropriate JDK and build tool, produces security
// plugin recommendations for sbt, and provides pre-commit hooks, CI commands,
// deny rules, and wizard fields for the Scala toolchain.
package scala

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.DenyRuleProvider = (*Module)(nil)
var _ ecosystem.WizardFieldProvider = (*Module)(nil)
var _ ecosystem.ManifestFileProvider = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// scalaVersionRe matches a scalaVersion setting in build.sbt, with an optional
// ThisBuild / prefix.
var scalaVersionRe = regexp.MustCompile(`^\s*(?:ThisBuild\s*/\s*)?scalaVersion\s*:=\s*"([^"]+)"`)

// sbtVersionRe matches the sbt.version property in project/build.properties.
var sbtVersionRe = regexp.MustCompile(`^\s*sbt\.version\s*=\s*(.+)`)

// millBuildFiles are the root build files Mill recognizes, newest convention
// first. Mill 1.x (the nixpkgs release) reads only build.mill and
// build.mill.yaml; build.sc is the Mill 0.x name.
var millBuildFiles = []string{"build.mill", "build.mill.yaml", "build.sc"}

// legacyMillBuildFile is the Mill 0.x build file name Mill 1.x ignores.
const legacyMillBuildFile = "build.sc"

// Manifests that declare Scala dependencies or build plugins, as path.Match
// patterns for ecosystem.DetectedManifests; the fallbacks name the
// conventional build file when detection recorded none.
var (
	sbtManifests = []ecosystem.ManifestFileInfo{
		{Path: "build.sbt", Ecosystem: "sbt", LockFilePolicy: ecosystem.LockFilePolicyNone},
		{Path: "project/*.sbt", Ecosystem: "sbt", LockFilePolicy: ecosystem.LockFilePolicyNone},
	}
	millManifests = []ecosystem.ManifestFileInfo{
		{Path: "build.mill", Ecosystem: "mill", LockFilePolicy: ecosystem.LockFilePolicyNone},
		{Path: "build.mill.yaml", Ecosystem: "mill", LockFilePolicy: ecosystem.LockFilePolicyNone},
		{Path: "build.sc", Ecosystem: "mill", LockFilePolicy: ecosystem.LockFilePolicyNone},
	}
)

// Module implements ecosystem.EcosystemModule for the Scala programming language.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "scala" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Scala" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 2 }

// Detect scans projectRoot for build.sbt, a Mill build file (build.mill,
// build.mill.yaml, or the legacy build.sc), and sbt metadata under project/.
// A project/ directory counts only when it holds sbt files (build.properties,
// *.sbt, *.scala): many non-Scala repositories have an unrelated project/
// folder. It extracts the Scala version from build.sbt, the sbt version from
// project/build.properties, and the Mill version from .mill-version.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	hasBuildSbt := fileutil.FileExists(projectRoot, "build.sbt")
	millFile := ""
	for _, name := range millBuildFiles {
		if fileutil.FileExists(projectRoot, name) {
			millFile = name
			break
		}
	}
	hasSbtProjectDir := hasSbtMetadata(projectRoot)

	if !hasBuildSbt && millFile == "" && !hasSbtProjectDir {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	confidence := ecosystem.ConfidenceProbable
	var evidence []string
	extras := make(map[string]string)

	switch {
	case hasBuildSbt:
		// sbt wins when both build tools are present.
		confidence = ecosystem.ConfidenceCertain
		evidence = append(evidence, "build.sbt found")
		extras["build_tool"] = "sbt"
	case millFile != "":
		confidence = ecosystem.ConfidenceCertain
		evidence = append(evidence, fmt.Sprintf("%s found (Mill)", millFile))
		extras["build_tool"] = "mill"
		evidence = append(evidence, millEvidence(projectRoot, millFile, extras)...)
	default:
		evidence = append(evidence, "project/ directory with sbt build files found")
		extras["build_tool"] = "sbt"
	}

	// Parse Scala version from build.sbt.
	version := ""
	if hasBuildSbt {
		version = parseScalaVersion(filepath.Join(projectRoot, "build.sbt"))
		if version != "" {
			evidence = append(evidence, fmt.Sprintf("Scala version %s (from build.sbt)", version))
		}
	}

	// Parse sbt version from project/build.properties.
	sbtVersion := parseSbtVersion(filepath.Join(projectRoot, "project", "build.properties"))
	if sbtVersion != "" {
		extras["sbt_version"] = sbtVersion
		evidence = append(evidence, fmt.Sprintf("sbt version %s (from project/build.properties)", sbtVersion))
	}

	var patterns []string
	for _, mf := range append(slices.Clone(sbtManifests), millManifests...) {
		patterns = append(patterns, mf.Path)
	}
	if manifests := ecosystem.RecordManifests(projectRoot, patterns...); manifests != "" {
		extras[ecosystem.ExtraManifests] = manifests
	}

	// Default JDK version.
	extras["jdk_version"] = strconv.Itoa(ecosystem.DefaultJDKMajor)

	return ecosystem.DetectionResult{
		Detected:   true,
		Confidence: confidence,
		Evidence:   evidence,
		SuggestedConfig: ecosystem.ModuleConfig{
			Version: version,
			Extras:  extras,
		},
	}
}

// hasSbtMetadata reports whether projectRoot/project holds sbt build files.
func hasSbtMetadata(projectRoot string) bool {
	if fileutil.FileExists(projectRoot, filepath.Join("project", "build.properties")) {
		return true
	}
	for _, pattern := range []string{"*.sbt", "*.scala"} {
		if matches, _ := filepath.Glob(filepath.Join(projectRoot, "project", pattern)); len(matches) > 0 {
			return true
		}
	}
	return false
}

// millEvidence records the Mill version a project requests (.mill-version)
// in extras and returns evidence lines warning about build files or versions
// the provisioned Mill 1.x cannot use.
func millEvidence(projectRoot, millFile string, extras map[string]string) []string {
	var evidence []string
	if data, err := os.ReadFile(filepath.Join(projectRoot, ".mill-version")); err == nil {
		if v := strings.TrimSpace(string(data)); v != "" {
			extras["mill_version"] = v
			evidence = append(evidence, fmt.Sprintf("Mill version %s (from .mill-version)", v))
			if strings.HasPrefix(v, "0.") {
				evidence = append(evidence, fmt.Sprintf("warning: .mill-version requests Mill %s, but the provisioned Mill is 1.x", v))
			}
		}
	}
	if millFile == legacyMillBuildFile {
		evidence = append(evidence, "warning: build.sc is a Mill 0.x build file; Mill 1.x only reads build.mill, so rename it (see the Mill 1.0 migration guide)")
	}
	return evidence
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Scala language support with the appropriate JDK and build tool.
//
// devenv's Scala module enables languages.java itself and builds sbt, Mill,
// Metals and scalafmt against languages.java.jdk.package. The Java module
// sets that option too, and a leaf defined twice in the devenv.nix attribute
// set is a Nix error, so Scala sets it with lib.mkDefault inside a separate
// imported module: the module system merges the two definitions and the
// Java module's explicit choice wins when both languages are enabled.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	jdkPkg, err := ecosystem.JDKPackage(config.Extra("jdk_version", ""))
	if err != nil {
		return "", fmt.Errorf("scala: %w", err)
	}

	var scalaProps []ecosystem.NixProperty
	switch config.Extra("build_tool", "sbt") {
	case "mill":
		// devenv builds Mill against the project JDK; a bare pkgs.mill would
		// run on whichever JDK nixpkgs defaults to.
		scalaProps = append(scalaProps, ecosystem.NixProperty{Key: "mill.enable", Value: "true"})
	default:
		scalaProps = append(scalaProps, ecosystem.NixProperty{Key: "sbt.enable", Value: "true"})
	}

	jdkBlock := "  # Scala's JDK is a default: an explicit Java module JDK takes precedence.\n" +
		fmt.Sprintf("  imports = [ { languages.java.jdk.package = lib.mkDefault pkgs.%s; } ];\n", jdkPkg)

	return ecosystem.BuildLanguageFragment(ecosystem.NixLangConfig{
		EnablePath:  "languages.scala",
		Properties:  scalaProps,
		ExtraBlocks: []string{jdkBlock},
	}), nil
}

// SecurityConfigs returns security plugin recommendations for sbt.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	content := `// Security plugins for sbt — generated by qsdev.
// Add these lines to your project/plugins.sbt to enable dependency security checks.
//
// Requires: sbt >= 1.0

addSbtPlugin("software.purpledragon" % "sbt-dependency-lock" % "1.1.3")
addSbtPlugin("net.vonbuchholtz" % "sbt-dependency-check" % "5.1.0")
`

	return []types.GeneratedFile{
		{
			Path:     "." + branding.Get().AppName + "/sbt-security-plugins.sbt",
			Content:  []byte(content),
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.Overwrite,
		},
	}
}

// scalafmtHookScript checks the staged Scala files with the Nix-provisioned
// scalafmt. scalafmt refuses to run without a .scalafmt.conf, so a project
// without one is skipped rather than blocked. When the config pins a version
// other than the provisioned one, scalafmt would download that release from
// Maven Central and execute it inside the hook, outside the Nix pin and the
// package guard; the hook fails with instructions instead.
const scalafmtHookScript = `if [ ! -f .scalafmt.conf ]; then
  echo "scalafmt: no .scalafmt.conf in the repository root; skipping the format check." >&2
  exit 0
fi
have=$(scalafmt --version) || exit 1
have=${have##* }
want=""
while IFS= read -r line || [ -n "$line" ]; do
  if [[ $line =~ ^[[:space:]]*version[[:space:]]*[=:][[:space:]]*\"?([^\"[:space:]]+) ]]; then
    want=${BASH_REMATCH[1]}
    break
  fi
done < .scalafmt.conf
if [ -z "$want" ]; then
  echo "scalafmt: .scalafmt.conf does not set the required version." >&2
  echo "Set version = \"$have\" (the provisioned scalafmt) in .scalafmt.conf to run this check." >&2
  exit 1
fi
if [ "$want" != "$have" ]; then
  echo "scalafmt: .scalafmt.conf pins version \"$want\" but the provisioned scalafmt is $have." >&2
  echo "Running it would download scalafmt $want from Maven Central outside the Nix pin." >&2
  echo "Set version = \"$have\" in .scalafmt.conf to run this check." >&2
  exit 1
fi
exec scalafmt --check --non-interactive --respect-project-filters "$@"`

// PreCommitHooks returns pre-commit hook definitions for the Scala ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "scalafmt",
			Name:          "scalafmt",
			Description:   "Check formatting of staged Scala sources with scalafmt",
			Entry:         "scalafmt --check",
			Script:        scalafmtHookScript,
			Language:      "system",
			Types:         []string{"scala"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       false,
			NixPackage:    "scalafmt",
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the Scala ecosystem.
// These prevent direct dependency modification outside of controlled workflows.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"Bash(sbt update *)",
		"Bash(sbt dependencyUpdates *)",
	}
}

// CICommands returns CI pipeline commands for the Scala ecosystem.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "sbt-dependency-lock-check",
			Command:     "sbt dependencyLockCheck",
			Description: "Verify sbt dependency lock file is up to date",
			Phase:       ecosystem.CIPhaseTest,
		},
		{
			Name:        "sbt-dependency-check",
			Command:     "sbt dependencyCheck",
			Description: "Scan Scala dependencies for known vulnerabilities",
			Phase:       ecosystem.CIPhaseScan,
		},
	}
}

// PackageManagers returns metadata about Scala's sbt package manager.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:                 "sbt",
			LockFile:             "build.sbt.lock",
			InstallCommand:       "sbt compile",
			FrozenInstallCommand: "sbt compile",
			AuditCommand:         "sbt dependencyCheck",
			AgeGatingSupport:     false,
		},
	}
}

// WizardFields returns additional wizard form fields for Scala configuration.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         "scala_build_tool",
			Label:       "Build tool",
			Description: "Select the Scala build tool for this project",
			Type:        ecosystem.FieldTypeSelect,
			Options: []ecosystem.WizardOption{
				{Label: "sbt", Value: "sbt"},
				{Label: "Mill", Value: "mill"},
			},
			Default: "sbt",
		},
		{
			Key:         "scala_jdk_version",
			Label:       "JDK version",
			Description: "Select the JDK version to use",
			Type:        ecosystem.FieldTypeSelect,
			Options:     ecosystem.JDKWizardOptions(),
			Default:     strconv.Itoa(ecosystem.DefaultJDKMajor),
		},
	}
}

// VerificationCommands returns build and test commands for Scala projects,
// switching on the configured build tool (sbt or mill).
func (m *Module) VerificationCommands(config ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	switch config.Extra("build_tool", "sbt") {
	case "mill":
		return ecosystem.VerificationCommands{
			Build: []string{"mill __.compile"},
			Test:  []string{"mill __.test"},
		}
	default:
		return ecosystem.VerificationCommands{
			Build: []string{"sbt compile"},
			Test:  []string{"sbt test"},
		}
	}
}

// ManifestFiles returns the dependency manifests of the configured build
// tool that Detect found (sbt: build.sbt and project/*.sbt, which holds the
// build plugins; Mill: its build file), or the conventional build file when
// none were recorded.
func (m *Module) ManifestFiles(config ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	if config.Extra("build_tool", "sbt") == "mill" {
		return ecosystem.DetectedManifests(config, millManifests, millManifests[:1])
	}
	return ecosystem.DetectedManifests(config, sbtManifests, sbtManifests[:1])
}

// parseScalaVersion reads a build.sbt file and extracts the Scala version
// from the scalaVersion setting. Returns an empty string if the setting
// is not found or the file cannot be read.
func parseScalaVersion(buildSbtPath string) string {
	f, err := os.Open(buildSbtPath)
	if err != nil {
		return ""
	}
	defer f.Close() //nolint:errcheck // best-effort read

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if matches := scalaVersionRe.FindStringSubmatch(scanner.Text()); matches != nil {
			return matches[1]
		}
	}
	return ""
}

// parseSbtVersion reads project/build.properties and extracts the sbt version.
// Returns an empty string if the property is not found or the file cannot be read.
func parseSbtVersion(buildPropertiesPath string) string {
	f, err := os.Open(buildPropertiesPath)
	if err != nil {
		return ""
	}
	defer f.Close() //nolint:errcheck // best-effort read

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if matches := sbtVersionRe.FindStringSubmatch(scanner.Text()); matches != nil {
			return strings.TrimSpace(matches[1])
		}
	}
	return ""
}
