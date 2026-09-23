// Package java implements the Java/Kotlin (JVM) ecosystem module for
// qsdev. It detects Maven and Gradle projects, generates
// devenv.nix fragments with the appropriate JDK, produces security-hardened
// configuration files (settings.xml, gradle.properties), and provides
// pre-commit hooks, CI commands, deny rules, and wizard fields for JVM development.
package java

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
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
var _ ecosystem.SASTModule = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// JVM build tool identifiers.
const (
	buildToolMaven  = "maven"
	buildToolGradle = "gradle"
	buildToolBoth   = "both"

	// defaultBuildTool is used when neither the package manager nor the
	// detection extras name a build tool. It matches the wizard default.
	defaultBuildTool = buildToolMaven
)

// gradleCmd runs Gradle builds. It is the Nix-provisioned gradle that
// languages.java.gradle.enable puts on PATH (pinned, hash-verified, and built
// against the configured JDK), not the project's ./gradlew: the wrapper runs
// the committed gradle-wrapper.jar, which downloads whatever distribution
// gradle-wrapper.properties names, and neither is verified by anything.
const gradleCmd = "gradle"

// mavenConfigPath is the per-project Maven CLI options file (Maven >= 3.3.1),
// and mavenConfigContent the options qsdev puts in it.
const (
	mavenConfigPath    = ".mvn/maven.config"
	mavenConfigContent = "--strict-checksums\n"
)

// mavenManifests and gradleManifests list the files that declare JVM
// dependencies, plugins or their versions (path.Match patterns, see
// ecosystem.DetectedManifests); the fallbacks name the conventional build
// file when detection recorded none.
var (
	mavenManifests = []ecosystem.ManifestFileInfo{
		{Path: "pom.xml", Ecosystem: "maven", LockFilePolicy: ecosystem.LockFilePolicyNone},
	}
	gradleManifests = []ecosystem.ManifestFileInfo{
		{Path: "build.gradle", Ecosystem: "gradle", LockFile: "gradle.lockfile", LockFilePolicy: ecosystem.LockFilePolicyRecommended},
		{Path: "build.gradle.kts", Ecosystem: "gradle", LockFile: "gradle.lockfile", LockFilePolicy: ecosystem.LockFilePolicyRecommended},
		{Path: "settings.gradle", Ecosystem: "gradle", LockFilePolicy: ecosystem.LockFilePolicyNone},
		{Path: "settings.gradle.kts", Ecosystem: "gradle", LockFilePolicy: ecosystem.LockFilePolicyNone},
		{Path: "gradle/libs.versions.toml", Ecosystem: "gradle", LockFile: "gradle.lockfile", LockFilePolicy: ecosystem.LockFilePolicyRecommended},
	}
	gradleManifestFallback = gradleManifests[:1]
)

// manifestPatterns returns the Path of every candidate manifest.
func manifestPatterns() []string {
	var patterns []string
	for _, m := range append(slices.Clone(mavenManifests), gradleManifests...) {
		patterns = append(patterns, m.Path)
	}
	return patterns
}

// resolveBuildTool returns the configured JVM build tool. It is the single
// source of truth for every Module method: the --java-build-tool flag stores
// the tool as the language's PackageManager, while detection and
// prepopulation store it in Extras["build_tool"]. An explicit PackageManager
// wins; when neither is set the default applies. An unrecognized explicit
// value is reported as an error (alongside the Extras/default fallback) so
// callers that can fail do so rather than silently ignoring the user's choice.
func resolveBuildTool(config ecosystem.ModuleConfig) (string, error) {
	var invalid error
	for _, raw := range []string{config.PackageManager, config.Extra("build_tool", "")} {
		v := strings.ToLower(strings.TrimSpace(raw))
		switch v {
		case "":
			continue
		case buildToolMaven, buildToolGradle, buildToolBoth:
			return v, invalid
		default:
			if invalid == nil {
				invalid = fmt.Errorf("unsupported Java build tool %q (want %s, %s, or %s)",
					raw, buildToolMaven, buildToolGradle, buildToolBoth)
			}
		}
	}
	return defaultBuildTool, invalid
}

// buildTool is resolveBuildTool for methods that cannot report errors.
func buildTool(config ecosystem.ModuleConfig) string {
	bt, _ := resolveBuildTool(config)
	return bt
}

// usesMaven reports whether the build tool includes Maven.
func usesMaven(bt string) bool { return bt == buildToolMaven || bt == buildToolBoth }

// usesGradle reports whether the build tool includes Gradle.
func usesGradle(bt string) bool { return bt == buildToolGradle || bt == buildToolBoth }

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
	hasMaven := fileutil.FileExists(projectRoot, "pom.xml")

	hasGradle := fileutil.FileExists(projectRoot, "build.gradle") ||
		fileutil.FileExists(projectRoot, "build.gradle.kts") ||
		fileutil.FileExists(projectRoot, "settings.gradle") ||
		fileutil.FileExists(projectRoot, "settings.gradle.kts")

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
		extras["build_tool"] = buildToolBoth
		evidence = append(evidence, "pom.xml found", "Gradle build file found")
	case hasMaven:
		extras["build_tool"] = buildToolMaven
		evidence = append(evidence, "pom.xml found")
	default:
		extras["build_tool"] = buildToolGradle
		evidence = append(evidence, "Gradle build file found")
	}

	if manifests := ecosystem.RecordManifests(projectRoot, manifestPatterns()...); manifests != "" {
		extras[ecosystem.ExtraManifests] = manifests
	}

	// Parse Java version from .java-version file.
	version := parseJavaVersion(projectRoot)
	if version != "" {
		if _, err := ecosystem.JDKPackage(version); err != nil {
			// Suggesting it would make generation fail; leave the version
			// unset (default JDK) and say why, so the user can pick one.
			evidence = append(evidence, fmt.Sprintf(".java-version %q ignored: %v", version, err))
			version = ""
		} else {
			evidence = append(evidence, fmt.Sprintf("Java version %s (from .java-version)", version))
		}
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
//
// Options are written as dotted leaf assignments. Other JVM modules (Scala,
// Clojure) and the LSP section also contribute languages.java settings to the
// same devenv.nix attribute set, where Nix merges distinct leaves but rejects
// any leaf defined twice.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	jdkPkg, err := ecosystem.JDKPackage(config.Version)
	if err != nil {
		return "", fmt.Errorf("java: %w", err)
	}
	bt, err := resolveBuildTool(config)
	if err != nil {
		return "", err
	}
	kotlin := config.Extra("kotlin", "") == "true"

	var b strings.Builder

	b.WriteString("  languages.java.enable = true;\n")
	fmt.Fprintf(&b, "  languages.java.jdk.package = pkgs.%s;\n", jdkPkg)
	if usesMaven(bt) {
		b.WriteString("  languages.java.maven.enable = true;\n")
	}
	if usesGradle(bt) {
		b.WriteString("  languages.java.gradle.enable = true;\n")
	}

	if kotlin {
		b.WriteString("\n")
		b.WriteString("  languages.kotlin.enable = true;\n")
	}

	return b.String(), nil
}

// SecurityConfigs returns generated security configuration files for Maven
// and/or Gradle based on the detected build tool.
func (m *Module) SecurityConfigs(config ecosystem.ModuleConfig) []types.GeneratedFile {
	bt := buildTool(config)
	var files []types.GeneratedFile

	if usesMaven(bt) {
		settings := buildSecuritySettings()
		if config.RegistryProxy != "" {
			// Replace the default central-only mirror with the corporate proxy.
			settings.Mirrors.Mirror = []Mirror{
				{
					ID:       "corporate-proxy",
					Name:     "Corporate registry proxy",
					URL:      config.RegistryProxy,
					MirrorOf: "*",
				},
			}
		}
		// A render failure drops only the Maven file: the Gradle hardening
		// below is independent and must still be generated for "both".
		// SecurityConfigs has no error return, so the failure is logged.
		if content, err := renderSettingsXML(settings); err != nil {
			slog.Warn("java: skipping .mvn/settings.xml: rendering failed", "error", err)
		} else {
			files = append(files, types.GeneratedFile{
				Path:     ".mvn/settings.xml",
				Content:  content,
				Mode:     fileutil.ModeReadWrite,
				Strategy: types.Skip,
			})
		}
		// Maven reads .mvn/maven.config from the project root on every run.
		// --strict-checksums makes a checksum mismatch fatal for dependencies
		// and build plugins alike, whichever repository serves them.
		files = append(files, types.GeneratedFile{
			Path:     mavenConfigPath,
			Content:  []byte(mavenConfigContent),
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.Skip,
		})
	}

	if usesGradle(bt) {
		content := buildGradleProperties()
		files = append(files, types.GeneratedFile{
			Path:     "gradle.properties",
			Content:  []byte(content),
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.Skip,
		})

		if config.RegistryProxy != "" {
			initGradle := buildInitGradle(config.RegistryProxy)
			files = append(files, types.GeneratedFile{
				Path:     "init.gradle",
				Content:  []byte(initGradle),
				Mode:     fileutil.ModeReadWrite,
				Strategy: types.Skip,
			})
		}
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
			// Custom hook (BuiltIn:false): NixPackage provisions the binary so
			// the emitted `entry` resolves at commit time.
			BuiltIn:    false,
			NixPackage: "google-java-format",
		},
		{
			// SpotBugs is not packaged in nixpkgs, and its bytecode analysis is a
			// poor fit for a source-stage pre-commit hook (it needs compiled
			// .class files). PMD provides equivalent Java static analysis over
			// SOURCE with its bundled quickstart ruleset (no project config file),
			// and is packaged as the top-level `pmd` attribute.
			//
			// Only the staged Java files are analyzed: pre-commit appends them
			// after -d, which takes a variable number of paths in PMD 6 (the
			// nixpkgs release). Scanning "." instead failed every commit on
			// violations in untouched files, build output and generated sources.
			ID:            "pmd",
			Name:          "pmd",
			Description:   "Run PMD static analysis on staged Java source (quickstart ruleset)",
			Entry:         "pmd -R rulesets/java/quickstart.xml -f text --no-cache -d",
			Language:      "system",
			Types:         []string{"java"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       false,
			NixPackage:    "pmd",
		},
	}

	if config.Extra("kotlin", "") == "true" {
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
			NixPackage:    "ktlint",
		})
	}

	return hooks
}

// DenyRules returns Claude Code deny-rule patterns for the JVM ecosystem.
// Rules are included conditionally based on the configured build tool.
func (m *Module) DenyRules(config ecosystem.ModuleConfig) []string {
	bt := buildTool(config)
	var rules []string

	if usesMaven(bt) {
		rules = append(rules,
			"Bash(mvn install *)",
			"Bash(mvn dependency:resolve *)",
			// dependency:get and dependency:copy fetch an arbitrary artifact
			// named on the command line (-Dartifact=g:a:v), bypassing the
			// project's declared and reviewed dependencies. The globs cover
			// the Maven wrapper, options placed before the goal, and the fully
			// qualified plugin form (maven-dependency-plugin:<ver>:get).
			"Bash(mvn *dependency*:get*)",
			"Bash(mvn *dependency*:copy*)",
			"Bash(./mvnw *dependency*:get*)",
			"Bash(./mvnw *dependency*:copy*)",
		)
	}

	if usesGradle(bt) {
		rules = append(rules,
			"Bash(gradle dependencies *)",
			"Bash(./gradlew dependencies *)",
		)
	}

	return rules
}

// CICommands returns CI pipeline commands for the JVM ecosystem.
func (m *Module) CICommands(config ecosystem.ModuleConfig) []ecosystem.CICommand {
	bt := buildTool(config)
	var cmds []ecosystem.CICommand

	if usesMaven(bt) {
		cmds = append(cmds, ecosystem.CICommand{
			Name:        "maven-verify",
			Command:     "mvn verify --strict-checksums",
			Description: "Build and verify Maven project with strict checksum enforcement",
			Phase:       ecosystem.CIPhaseTest,
		})
	}

	if usesGradle(bt) {
		// CI verifies against the committed gradle/verification-metadata.xml.
		// It must never (re)generate that file: --write-verification-metadata
		// in CI would trust whatever the network serves on that run.
		cmds = append(cmds, ecosystem.CICommand{
			Name:        "gradle-build",
			Command:     gradleCmd + " build --dependency-verification strict",
			Description: "Build Gradle project with strict dependency verification",
			Phase:       ecosystem.CIPhaseTest,
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
			InstallCommand:       gradleCmd + " build",
			FrozenInstallCommand: gradleCmd + " build --dependency-verification strict",
			AuditCommand:         gradleCmd + " dependencyCheckAnalyze",
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
			Options:     ecosystem.JDKWizardOptions(),
			Default:     strconv.Itoa(ecosystem.DefaultJDKMajor),
			Required:    true,
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

// VerificationCommands returns project verification commands for the JVM
// ecosystem, switching on the configured build tool (maven, gradle, or both).
func (m *Module) VerificationCommands(config ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	switch buildTool(config) {
	case buildToolGradle:
		return ecosystem.VerificationCommands{
			Build: []string{gradleCmd + " build"},
			Test:  []string{gradleCmd + " test"},
		}
	case buildToolBoth:
		return ecosystem.VerificationCommands{
			Build: []string{"mvn compile", gradleCmd + " build"},
			Test:  []string{"mvn test", gradleCmd + " test"},
		}
	default:
		return ecosystem.VerificationCommands{
			Build: []string{"mvn compile"},
			Test:  []string{"mvn test"},
		}
	}
}

// ManifestFiles returns manifest file metadata for the JVM ecosystem: the
// build tool's manifests that Detect found (Kotlin-DSL build scripts, settings
// scripts and the gradle/libs.versions.toml version catalog included), or the
// conventional build file when none were recorded.
func (m *Module) ManifestFiles(config ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	bt := buildTool(config)
	var out []ecosystem.ManifestFileInfo
	if usesMaven(bt) {
		out = append(out, ecosystem.DetectedManifests(config, mavenManifests, mavenManifests)...)
	}
	if usesGradle(bt) {
		out = append(out, ecosystem.DetectedManifests(config, gradleManifests, gradleManifestFallback)...)
	}
	return out
}

// buildGradleProperties returns the content of a security-hardened
// gradle.properties file.
//
// Only settings Gradle actually reads from gradle.properties belong here.
// Dependency locking has no gradle.properties switch (it is configured in a
// build or settings script), so the file documents the required bootstrap
// steps instead of emitting a property Gradle would silently ignore.
func buildGradleProperties() string {
	var b strings.Builder
	b.WriteString("# " + branding.GeneratedBy() + " — supply-chain security hardened.\n")
	b.WriteString("# Requires: Gradle >= 6.1 (dependency locking), >= 6.2 (dependency verification).\n")
	b.WriteString("\n")
	b.WriteString("# Strict dependency verification (checksums + signatures). Gradle enforces it\n")
	b.WriteString("# only once gradle/verification-metadata.xml exists. Bootstrap it once, review\n")
	b.WriteString("# the result, and commit it (never regenerate it in CI):\n")
	b.WriteString("#   gradle --write-verification-metadata sha256,pgp help\n")
	b.WriteString("org.gradle.dependency.verification=strict\n")
	b.WriteString("\n")
	b.WriteString("# Dependency locking cannot be enabled from gradle.properties. Add to your\n")
	b.WriteString("# build script, then run gradle dependencies --write-locks and commit the\n")
	b.WriteString("# generated gradle.lockfile:\n")
	b.WriteString("#   dependencyLocking { lockAllConfigurations(); lockMode = LockMode.STRICT }\n")
	return b.String()
}

// buildInitGradle returns the content of an init.gradle file that configures
// a corporate registry proxy for all Gradle projects.
func buildInitGradle(proxyURL string) string {
	var b strings.Builder
	b.WriteString("// " + branding.GeneratedBy() + " — corporate registry proxy.\n")
	b.WriteString("allprojects {\n")
	b.WriteString("    repositories {\n")
	b.WriteString("        all { ArtifactRepository repo ->\n")
	b.WriteString("            if (repo instanceof MavenArtifactRepository) {\n")
	b.WriteString("                remove repo\n")
	b.WriteString("            }\n")
	b.WriteString("        }\n")
	b.WriteString("        maven {\n")
	fmt.Fprintf(&b, "            url '%s'\n", ecosystem.GradleEscapeString(proxyURL))
	b.WriteString("        }\n")
	b.WriteString("    }\n")
	b.WriteString("}\n")
	return b.String()
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
	// Check for .kt files at project root.
	matches, _ := filepath.Glob(filepath.Join(projectRoot, "*.kt"))
	if len(matches) > 0 {
		return true
	}

	// Walk src/ subtree for .kt files.
	srcDir := filepath.Join(projectRoot, "src")
	if info, err := os.Stat(srcDir); err == nil && info.IsDir() {
		found := false
		_ = filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // skip unreadable dirs
			}
			if !d.IsDir() && strings.HasSuffix(d.Name(), ".kt") {
				found = true
				return filepath.SkipAll
			}
			return nil
		})
		if found {
			return true
		}
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

// SemgrepRuleSets returns Semgrep rule set identifiers relevant to Java/Kotlin projects.
func (m *Module) SemgrepRuleSets() []string {
	return []string{"p/java", "p/kotlin", "p/spring", "p/owasp-top-ten"}
}
