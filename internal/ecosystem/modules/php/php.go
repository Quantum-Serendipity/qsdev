// Package php implements the PHP (Composer) ecosystem module for
// qsdev. It detects PHP projects by scanning for
// composer.json and composer.lock, generates devenv.nix fragments with the
// appropriate PHP version, produces a security-hardened Composer configuration
// file, and provides pre-commit hooks, CI commands, deny rules, and wizard
// fields for the PHP toolchain.
package php

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/internal/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*Module)(nil)

func init() {
	ecosystem.RegisterModule(&Module{})
}

// phpVersionRe matches a PHP version constraint and extracts the major.minor
// portion, ignoring constraint operators like >=, ^, ~, etc.
var phpVersionRe = regexp.MustCompile(`[>=!^~]*\s*(\d+\.\d+)`)

// Module implements ecosystem.EcosystemModule for the PHP programming language.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "php" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "PHP" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 2 }

// Detect scans projectRoot for composer.json and composer.lock files and
// extracts the PHP version from the require.php field in composer.json.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	composerJSON := filepath.Join(projectRoot, "composer.json")
	composerLock := filepath.Join(projectRoot, "composer.lock")

	hasJSON := fileutil.FileExists(composerJSON)
	hasLock := fileutil.FileExists(composerLock)

	if !hasJSON && !hasLock {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	confidence := ecosystem.ConfidenceProbable
	var evidence []string

	if hasJSON {
		confidence = ecosystem.ConfidenceCertain
		evidence = append(evidence, "composer.json found")
	}
	if hasLock {
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		evidence = append(evidence, "composer.lock found")
	}

	version := parsePHPVersion(composerJSON)
	if version != "" {
		evidence = append(evidence, fmt.Sprintf("PHP version %s (from composer.json require.php)", version))
	}

	return ecosystem.DetectionResult{
		Detected:   true,
		Confidence: confidence,
		Evidence:   evidence,
		SuggestedConfig: ecosystem.ModuleConfig{
			Version: version,
		},
	}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for PHP language support with the appropriate PHP version package.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	pkg := phpPackage(config.Version)

	var b strings.Builder
	b.WriteString("  languages.php.enable = true;\n")
	fmt.Fprintf(&b, "  languages.php.package = pkgs.%s;\n", pkg)
	return b.String(), nil
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
// PHP does not require any additional inputs.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return nil
}

// SecurityConfigs returns a security-hardened Composer configuration file.
func (m *Module) SecurityConfigs(config ecosystem.ModuleConfig) []types.GeneratedFile {
	type composerRepo struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	}

	type securityConfigType struct {
		Comment      string         `json:"_comment"`
		Requires     string         `json:"_requires"`
		Repositories []composerRepo `json:"repositories,omitempty"`
		Config       struct {
			SecureHTTP       bool              `json:"secure-http"`
			Lock             bool              `json:"lock"`
			Audit            map[string]string `json:"audit"`
			AllowPlugins     map[string]any    `json:"allow-plugins"`
			PreferredInstall string            `json:"preferred-install"`
		} `json:"config"`
	}

	securityConfig := securityConfigType{
		Comment:  "Security-hardened Composer configuration — merge into your composer.json config section.",
		Requires: "Composer >= 2.9 for audit.block-insecure. Composer 2.9+ blocks known-vulnerable packages by default.",
	}

	if config.RegistryProxy != "" {
		securityConfig.Repositories = []composerRepo{
			{Type: "composer", URL: config.RegistryProxy},
		}
	}

	securityConfig.Config.SecureHTTP = true
	securityConfig.Config.Lock = true
	securityConfig.Config.Audit = map[string]string{"abandoned": "fail"}
	securityConfig.Config.AllowPlugins = map[string]any{}
	securityConfig.Config.PreferredInstall = "dist"

	content, err := json.MarshalIndent(securityConfig, "", "  ")
	if err != nil {
		return nil
	}
	content = append(content, '\n')

	return []types.GeneratedFile{
		{
			Path:     ".qsdev/composer-security.json",
			Content:  content,
			Mode:     0o644,
			Strategy: types.Overwrite,
		},
	}
}

// PreCommitHooks returns pre-commit hook definitions for the PHP ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "phpcs",
			Name:          "phpcs",
			Description:   "Run PHP_CodeSniffer to check coding standards",
			Entry:         "phpcs",
			Language:      "system",
			Types:         []string{"php"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       true,
		},
		{
			ID:            "phpstan",
			Name:          "phpstan",
			Description:   "Run PHPStan static analysis",
			Entry:         "phpstan analyse",
			Language:      "system",
			Types:         []string{"php"},
			Stages:        []string{"pre-commit"},
			PassFilenames: false,
			BuiltIn:       false,
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the PHP ecosystem.
// These prevent direct dependency modification outside of controlled workflows.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"Bash(composer require *)",
		"Bash(composer remove *)",
		"Bash(composer update *)",
	}
}

// CICommands returns CI pipeline commands for the PHP ecosystem.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "composer-install",
			Command:     "composer install --no-dev --no-scripts --no-interaction",
			Description: "Install PHP dependencies without dev packages or scripts",
			Phase:       ecosystem.CIPhaseInstall,
		},
		{
			Name:        "composer-validate",
			Command:     "composer validate --strict",
			Description: "Validate composer.json schema and consistency",
			Phase:       ecosystem.CIPhaseTest,
		},
		{
			Name:        "composer-audit",
			Command:     "composer audit",
			Description: "Audit PHP dependencies for known vulnerabilities",
			Phase:       ecosystem.CIPhaseScan,
		},
	}
}

// PackageManagers returns metadata about PHP's Composer package manager.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:                 "composer",
			LockFile:             "composer.lock",
			InstallCommand:       "composer install",
			FrozenInstallCommand: "composer install --no-dev --no-scripts",
			AuditCommand:         "composer audit",
			AgeGatingSupport:     false,
		},
	}
}

// WizardFields returns additional wizard form fields for PHP configuration.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         "php_version",
			Label:       "PHP version",
			Description: "Select the PHP version to use",
			Type:        ecosystem.FieldTypeSelect,
			Options: []ecosystem.WizardOption{
				{Label: "8.3", Value: "8.3"},
				{Label: "8.2", Value: "8.2"},
				{Label: "8.1", Value: "8.1"},
			},
			Default: "8.3",
		},
	}
}

// VerificationCommands returns test and lint commands for PHP projects.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{
		Test: []string{"composer test"},
		Lint: []string{"composer run lint"},
	}
}

// ManifestFiles returns the composer.json manifest file for PHP projects.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return []ecosystem.ManifestFileInfo{{Path: "composer.json", Ecosystem: "composer", LockFile: "composer.lock", LockFilePolicy: ecosystem.LockFilePolicyRequired}}
}

// phpPackage maps a version string to the corresponding Nix PHP package name.
func phpPackage(version string) string {
	switch version {
	case "8.3":
		return "php83"
	case "8.2":
		return "php82"
	case "8.1":
		return "php81"
	default:
		return "php83"
	}
}

// parsePHPVersion reads composer.json and extracts the PHP version from the
// require.php field. Returns an empty string if the field is not found,
// the file cannot be read, or the version cannot be parsed.
func parsePHPVersion(composerJSONPath string) string {
	data, err := os.ReadFile(composerJSONPath)
	if err != nil {
		return ""
	}

	var composerFile struct {
		Require map[string]string `json:"require"`
	}
	if err := json.Unmarshal(data, &composerFile); err != nil {
		return ""
	}

	phpConstraint, ok := composerFile.Require["php"]
	if !ok {
		return ""
	}

	matches := phpVersionRe.FindStringSubmatch(phpConstraint)
	if matches == nil {
		return ""
	}

	return matches[1]
}
