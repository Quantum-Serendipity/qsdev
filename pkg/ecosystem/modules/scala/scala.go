// Package scala implements the Scala (sbt/Mill) ecosystem module for
// qsdev. It detects Scala projects by scanning for
// build.sbt, build.sc, and project/ directories, generates devenv.nix
// fragments with the appropriate JDK and build tool, produces security
// plugin recommendations for sbt, and provides pre-commit hooks, CI commands,
// deny rules, and wizard fields for the Scala toolchain.
package scala

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.PackageProvider = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// scalaVersionRe matches a scalaVersion setting in build.sbt, with an optional
// ThisBuild / prefix.
var scalaVersionRe = regexp.MustCompile(`^\s*(?:ThisBuild\s*/\s*)?scalaVersion\s*:=\s*"([^"]+)"`)

// sbtVersionRe matches the sbt.version property in project/build.properties.
var sbtVersionRe = regexp.MustCompile(`^\s*sbt\.version\s*=\s*(.+)`)

// Module implements ecosystem.EcosystemModule for the Scala programming language.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "scala" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Scala" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 2 }

// Detect scans projectRoot for build.sbt, build.sc (Mill), and the project/
// directory. It extracts the Scala version from build.sbt and the sbt version
// from project/build.properties.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	hasBuildSbt := fileutil.FileExists(projectRoot, "build.sbt")
	hasBuildSc := fileutil.FileExists(projectRoot, "build.sc")
	hasProjectDir := fileutil.DirExists(projectRoot, "project")

	if !hasBuildSbt && !hasBuildSc && !hasProjectDir {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	confidence := ecosystem.ConfidenceProbable
	var evidence []string
	extras := make(map[string]string)

	if hasBuildSbt {
		confidence = ecosystem.ConfidenceCertain
		evidence = append(evidence, "build.sbt found")
		extras["build_tool"] = "sbt"
	}
	if hasBuildSc {
		confidence = ecosystem.ConfidenceCertain
		evidence = append(evidence, "build.sc found (Mill)")
		extras["build_tool"] = "mill"
	}
	// If both are present, prefer sbt.
	if hasBuildSbt && hasBuildSc {
		extras["build_tool"] = "sbt"
	}
	if hasProjectDir && !hasBuildSbt && !hasBuildSc {
		evidence = append(evidence, "project/ directory found")
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

	// Default JDK version.
	extras["jdk_version"] = "21"

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

// DevenvPackages returns Nix packages required by the Scala module.
// Mill projects need the mill package; sbt projects get sbt via the
// languages.scala.sbt.enable fragment.
func (m *Module) DevenvPackages(config ecosystem.ModuleConfig) []string {
	buildTool := config.Extras["build_tool"]
	if buildTool == "" {
		buildTool = "sbt"
	}
	if buildTool == "mill" {
		return []string{"mill"}
	}
	return nil
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Scala language support with the appropriate JDK and build tool.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	buildTool := config.Extras["build_tool"]
	if buildTool == "" {
		buildTool = "sbt"
	}

	jdkVer := config.Extras["jdk_version"]
	if jdkVer == "" {
		jdkVer = "21"
	}

	jdkPkg := jdkPackage(jdkVer)

	var b strings.Builder

	b.WriteString("  languages.scala = {\n")
	b.WriteString("    enable = true;\n")
	if buildTool == "sbt" {
		b.WriteString("    sbt.enable = true;\n")
	}
	b.WriteString("  };\n")

	b.WriteString("\n")
	b.WriteString("  languages.java = {\n")
	b.WriteString("    enable = true;\n")
	fmt.Fprintf(&b, "    jdk.package = pkgs.%s;\n", jdkPkg)
	b.WriteString("  };\n")

	return b.String(), nil
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
// Scala does not require any additional inputs.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return nil
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
			Mode:     0o644,
			Strategy: types.Overwrite,
		},
	}
}

// PreCommitHooks returns pre-commit hook definitions for the Scala ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "scalafmt",
			Name:          "scalafmt",
			Description:   "Check Scala source formatting with scalafmt",
			Entry:         "scalafmt --check",
			Language:      "system",
			Types:         []string{"scala"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       true,
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
			Options: []ecosystem.WizardOption{
				{Label: "JDK 21 (LTS)", Value: "21"},
				{Label: "JDK 17 (LTS)", Value: "17"},
				{Label: "JDK 11 (LTS)", Value: "11"},
			},
			Default: "21",
		},
	}
}

// VerificationCommands returns build and test commands for Scala projects.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{
		Build: []string{"sbt compile"},
		Test:  []string{"sbt test"},
	}
}

// ManifestFiles returns the build.sbt manifest file for Scala projects.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return []ecosystem.ManifestFileInfo{{Path: "build.sbt", Ecosystem: "sbt", LockFilePolicy: ecosystem.LockFilePolicyNone}}
}

// jdkPackage maps a version string to the corresponding Nix JDK package name.
func jdkPackage(version string) string {
	switch version {
	case "21":
		return "jdk21"
	case "17":
		return "jdk17"
	case "11":
		return "jdk11"
	default:
		return "jdk21"
	}
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
