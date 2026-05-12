// Package java implements the Java/Kotlin (JVM) ecosystem module for
// gdev-secure-devenv-bootstrap. It detects Maven and Gradle projects, generates
// devenv.nix fragments with the appropriate JDK, produces security-hardened
// configuration files (settings.xml, gradle.properties), and provides
// pre-commit hooks, CI commands, deny rules, and wizard fields for JVM development.
package java

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*Module)(nil)

func init() {
	if err := ecosystem.DefaultRegistry().Register(&Module{}); err != nil {
		panic(fmt.Sprintf("java: failed to register ecosystem module: %v", err))
	}
}

// Module implements ecosystem.EcosystemModule for the Java/Kotlin (JVM) ecosystem.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "java" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Java/Kotlin (JVM)" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 1 }

// Detect scans projectRoot for Maven and Gradle build files, a .java-version
// file, and Kotlin source files. It populates Extras with the detected build
// tool ("maven", "gradle", or "both") and whether Kotlin is present.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	hasMaven := fileExists(filepath.Join(projectRoot, "pom.xml"))

	hasGradle := fileExists(filepath.Join(projectRoot, "build.gradle")) ||
		fileExists(filepath.Join(projectRoot, "build.gradle.kts")) ||
		fileExists(filepath.Join(projectRoot, "settings.gradle")) ||
		fileExists(filepath.Join(projectRoot, "settings.gradle.kts"))

	if !hasMaven && !hasGradle {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	extras := make(map[string]string)
	var evidence []string

	// Determine build tool.
	switch {
	case hasMaven && hasGradle:
		extras["build_tool"] = "both"
		evidence = append(evidence, "pom.xml found", "Gradle build file found")
	case hasMaven:
		extras["build_tool"] = "maven"
		evidence = append(evidence, "pom.xml found")
	default:
		extras["build_tool"] = "gradle"
		evidence = append(evidence, "Gradle build file found")
	}

	// Parse Java version from .java-version file.
	version := parseJavaVersion(projectRoot)
	if version != "" {
		evidence = append(evidence, fmt.Sprintf("Java version %s (from .java-version)", version))
	}

	// Detect Kotlin via .kt files or build.gradle content.
	kotlin := detectKotlin(projectRoot)
	if kotlin {
		extras["kotlin"] = "true"
		evidence = append(evidence, "Kotlin detected")
	} else {
		extras["kotlin"] = "false"
	}

	return ecosystem.DetectionResult{
		Detected:   true,
		Confidence: ecosystem.ConfidenceCertain,
		Evidence:   evidence,
		SuggestedConfig: ecosystem.ModuleConfig{
			Version: version,
			Extras:  extras,
		},
	}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for JVM language support with the appropriate JDK and build tools.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	jdkPkg := jdkPackage(config.Version)
	buildTool := config.Extras["build_tool"]
	kotlin := config.Extras["kotlin"] == "true"

	var b strings.Builder

	b.WriteString("  languages.java = {\n")
	b.WriteString("    enable = true;\n")
	fmt.Fprintf(&b, "    jdk.package = pkgs.%s;\n", jdkPkg)
	if buildTool == "maven" || buildTool == "both" {
		b.WriteString("    maven.enable = true;\n")
	}
	if buildTool == "gradle" || buildTool == "both" {
		b.WriteString("    gradle.enable = true;\n")
	}
	b.WriteString("  };\n")

	if kotlin {
		b.WriteString("\n")
		b.WriteString("  languages.kotlin.enable = true;\n")
	}

	return b.String(), nil
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
// Java does not require any additional inputs.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return nil
}

// SecurityConfigs returns generated security configuration files for Maven
// and/or Gradle based on the detected build tool.
func (m *Module) SecurityConfigs(config ecosystem.ModuleConfig) []types.GeneratedFile {
	buildTool := config.Extras["build_tool"]
	var files []types.GeneratedFile

	if buildTool == "maven" || buildTool == "both" {
		settings := buildSecuritySettings()
		content, err := renderSettingsXML(settings)
		if err != nil {
			// Fallback: return an empty slice rather than crashing.
			return nil
		}
		files = append(files, types.GeneratedFile{
			Path:     ".mvn/settings.xml",
			Content:  content,
			Mode:     0o644,
			Strategy: types.Overwrite,
		})
	}

	if buildTool == "gradle" || buildTool == "both" {
		content := buildGradleProperties()
		files = append(files, types.GeneratedFile{
			Path:     "gradle.properties",
			Content:  []byte(content),
			Mode:     0o644,
			Strategy: types.Overwrite,
		})
	}

	return files
}

// PreCommitHooks returns pre-commit hook definitions for the JVM ecosystem.
// When Kotlin is detected, an additional ktlint hook is included.
func (m *Module) PreCommitHooks(config ecosystem.ModuleConfig) []ecosystem.HookConfig {
	hooks := []ecosystem.HookConfig{
		{
			ID:            "google-java-format",
			Name:          "google-java-format",
			Description:   "Format Java source code with google-java-format",
			Entry:         "google-java-format --replace",
			Language:      "system",
			Types:         []string{"java"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       false,
		},
		{
			ID:            "spotbugs",
			Name:          "spotbugs",
			Description:   "Run SpotBugs static analysis on Java bytecode",
			Entry:         "spotbugs",
			Language:      "system",
			Types:         []string{"java"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       false,
		},
	}

	if config.Extras["kotlin"] == "true" {
		hooks = append(hooks, ecosystem.HookConfig{
			ID:            "ktlint",
			Name:          "ktlint",
			Description:   "Lint and format Kotlin source code with ktlint",
			Entry:         "ktlint --format",
			Language:      "system",
			Types:         []string{"kotlin"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			Files:         `\.kts?$`,
			BuiltIn:       false,
		})
	}

	return hooks
}

// DenyRules returns Claude Code deny-rule patterns for the JVM ecosystem.
// Rules are included conditionally based on the detected build tool.
func (m *Module) DenyRules(config ecosystem.ModuleConfig) []string {
	buildTool := config.Extras["build_tool"]
	var rules []string

	if buildTool == "maven" || buildTool == "both" {
		rules = append(rules,
			"Bash(mvn install *)",
			"Bash(mvn dependency:resolve *)",
		)
	}

	if buildTool == "gradle" || buildTool == "both" {
		rules = append(rules,
			"Bash(gradle dependencies *)",
			"Bash(./gradlew dependencies *)",
		)
	}

	return rules
}

// CICommands returns CI pipeline commands for the JVM ecosystem.
func (m *Module) CICommands(config ecosystem.ModuleConfig) []ecosystem.CICommand {
	buildTool := config.Extras["build_tool"]
	var cmds []ecosystem.CICommand

	if buildTool == "maven" || buildTool == "both" {
		cmds = append(cmds, ecosystem.CICommand{
			Name:        "maven-verify",
			Command:     "mvn verify --strict-checksums",
			Description: "Build and verify Maven project with strict checksum enforcement",
			Phase:       ecosystem.CIPhaseTest,
		})
	}

	if buildTool == "gradle" || buildTool == "both" {
		cmds = append(cmds, ecosystem.CICommand{
			Name:        "gradle-build",
			Command:     "./gradlew build",
			Description: "Build Gradle project",
			Phase:       ecosystem.CIPhaseTest,
		})
		cmds = append(cmds, ecosystem.CICommand{
			Name:        "gradle-verification-metadata",
			Command:     "./gradlew --write-verification-metadata sha256,pgp",
			Description: "Generate Gradle dependency verification metadata",
			Phase:       ecosystem.CIPhaseScan,
		})
	}

	return cmds
}

// PackageManagers returns metadata about the JVM ecosystem's package managers.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:                 "maven",
			LockFile:             "pom.xml",
			InstallCommand:       "mvn install",
			FrozenInstallCommand: "mvn dependency:resolve --strict-checksums",
			AuditCommand:         "mvn org.owasp:dependency-check-maven:check",
			AgeGatingSupport:     false,
		},
		{
			Name:                 "gradle",
			LockFile:             "gradle.lockfile",
			InstallCommand:       "./gradlew build",
			FrozenInstallCommand: "./gradlew build --dependency-verification strict",
			AuditCommand:         "./gradlew dependencyCheckAnalyze",
			AgeGatingSupport:     false,
		},
	}
}

// WizardFields returns additional wizard form fields for JVM configuration.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         "java_build_tool",
			Label:       "Build tool",
			Description: "Select the primary JVM build tool for this project",
			Type:        ecosystem.FieldTypeSelect,
			Options: []ecosystem.WizardOption{
				{Label: "Maven", Value: "maven"},
				{Label: "Gradle", Value: "gradle"},
				{Label: "Both", Value: "both"},
			},
			Default:  "maven",
			Required: true,
		},
		{
			Key:         "java_jdk_version",
			Label:       "JDK version",
			Description: "Select the JDK version to use",
			Type:        ecosystem.FieldTypeSelect,
			Options: []ecosystem.WizardOption{
				{Label: "JDK 21 (LTS)", Value: "21"},
				{Label: "JDK 17 (LTS)", Value: "17"},
				{Label: "JDK 11 (LTS)", Value: "11"},
			},
			Default:  "21",
			Required: true,
		},
		{
			Key:         "java_kotlin",
			Label:       "Kotlin support",
			Description: "Enable Kotlin language support alongside Java",
			Type:        ecosystem.FieldTypeConfirm,
			Default:     "false",
		},
	}
}

// jdkPackage maps a version string to the corresponding Nix JDK package name.
func jdkPackage(version string) string {
	switch version {
	case "17":
		return "jdk17"
	case "11":
		return "jdk11"
	case "21":
		return "jdk21"
	default:
		return "jdk21"
	}
}

// buildGradleProperties returns the content of a security-hardened
// gradle.properties file.
func buildGradleProperties() string {
	var b strings.Builder
	b.WriteString("# Generated by gdev-secure-devenv-bootstrap — supply-chain security hardened.\n")
	b.WriteString("# Requires: Gradle >= 6.1 (dependency locking), >= 6.2 (dependency verification).\n")
	b.WriteString("#\n")
	b.WriteString("# Strict dependency locking requires all dependencies to be locked.\n")
	b.WriteString("# Strict dependency verification validates checksums and signatures.\n")
	b.WriteString("\n")
	b.WriteString("# Enforce strict dependency locking across all configurations.\n")
	b.WriteString("dependencyLocking.lockMode=STRICT\n")
	b.WriteString("\n")
	b.WriteString("# Require strict dependency verification (checksum + signature validation).\n")
	b.WriteString("systemProp.org.gradle.dependency.verification=strict\n")
	return b.String()
}

// fileExists reports whether a file at the given path exists and is not a directory.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// parseJavaVersion reads .java-version in projectRoot and returns the
// trimmed content. Returns an empty string if the file does not exist
// or cannot be read.
func parseJavaVersion(projectRoot string) string {
	data, err := os.ReadFile(filepath.Join(projectRoot, ".java-version"))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// detectKotlin checks for the presence of Kotlin in a project by looking for
// .kt files and by scanning Gradle build files for the "kotlin" substring.
func detectKotlin(projectRoot string) bool {
	// Check for .kt files.
	matches, _ := filepath.Glob(filepath.Join(projectRoot, "*.kt"))
	if len(matches) > 0 {
		return true
	}

	// Also check in src/ subtree — common convention.
	matches, _ = filepath.Glob(filepath.Join(projectRoot, "src", "**", "*.kt"))
	if len(matches) > 0 {
		return true
	}

	// Check Gradle build files for "kotlin" substring.
	for _, name := range []string{"build.gradle", "build.gradle.kts"} {
		path := filepath.Join(projectRoot, name)
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if strings.Contains(strings.ToLower(string(data)), "kotlin") {
			return true
		}
	}

	return false
}
