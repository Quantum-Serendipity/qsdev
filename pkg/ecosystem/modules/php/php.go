// Package php implements the PHP (Composer) ecosystem module for
// qsdev. It detects PHP projects by scanning for
// composer.json and composer.lock, generates devenv.nix fragments with the
// appropriate PHP version, produces a security-hardened Composer configuration
// file, and provides pre-commit hooks, CI commands, deny rules, and wizard
// fields for the PHP toolchain.
package php

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.WizardFieldProvider = (*Module)(nil)
var _ ecosystem.ManifestFileProvider = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// ErrUnsupportedPHPVersion is returned when a configured PHP version or
// constraint cannot be satisfied by any PHP series qsdev can provision.
var ErrUnsupportedPHPVersion = errors.New("unsupported PHP version")

// phpSeries is a PHP minor release series (e.g. 8.3).
type phpSeries struct{ major, minor int }

func (s phpSeries) String() string { return fmt.Sprintf("%d.%d", s.major, s.minor) }

// nixAttr returns the nixpkgs attribute providing this series (e.g. php83).
func (s phpSeries) nixAttr() string { return fmt.Sprintf("php%d%d", s.major, s.minor) }

// supportedPHPSeries lists the PHP series available in the pinned nixpkgs,
// newest first. End-of-life series are removed from nixpkgs (php81 is a
// throw-alias that fails evaluation), so they must not be listed here.
var supportedPHPSeries = []phpSeries{{8, 5}, {8, 4}, {8, 3}, {8, 2}}

// defaultPHPSeries is provisioned when no version is configured.
var defaultPHPSeries = phpSeries{8, 3}

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

	var version string
	if constraint := parsePHPConstraint(composerJSON); constraint != "" {
		if series, ok := resolvePHPConstraint(constraint); ok {
			version = series.String()
			evidence = append(evidence, fmt.Sprintf("PHP version %s (satisfies composer.json require.php %q)", version, constraint))
		} else {
			evidence = append(evidence, fmt.Sprintf("composer.json require.php %q is not satisfied by any supported PHP version (%s)", constraint, supportedPHPList()))
		}
	}

	suggested := ecosystem.ModuleConfig{Version: version}
	if cfg, ok := findPHPCSConfig(projectRoot); ok {
		evidence = append(evidence, cfg+" found")
		suggested.Extras = map[string]string{ExtraPHPCSConfig: "true"}
	}

	return ecosystem.DetectionResult{
		Detected:        true,
		Confidence:      confidence,
		Evidence:        evidence,
		SuggestedConfig: suggested,
	}
}

// ExtraPHPCSConfig marks a project that ships its own PHP_CodeSniffer ruleset.
const ExtraPHPCSConfig = "phpcs_config"

// phpcsConfigFiles are the rulesets PHP_CodeSniffer loads from the working
// directory when no --standard is given.
var phpcsConfigFiles = []string{".phpcs.xml", "phpcs.xml", ".phpcs.xml.dist", "phpcs.xml.dist"}

// findPHPCSConfig returns the project's PHP_CodeSniffer ruleset, if any.
func findPHPCSConfig(projectRoot string) (string, bool) {
	for _, name := range phpcsConfigFiles {
		if fileutil.FileExists(projectRoot, name) {
			return name, true
		}
	}
	return "", false
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for PHP language support with the appropriate PHP version package.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	pkg, err := phpPackage(config.Version)
	if err != nil {
		return "", err
	}

	var b strings.Builder
	b.WriteString("  languages.php.enable = true;\n")
	fmt.Fprintf(&b, "  languages.php.package = pkgs.%s;\n", pkg)
	return b.String(), nil
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
			Path:     "." + branding.Get().AppName + "/composer-security.json",
			Content:  content,
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.Overwrite,
		},
	}
}

// PreCommitHooks returns pre-commit hook definitions for the PHP ecosystem.
func (m *Module) PreCommitHooks(config ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		phpcsHook(config),
		{
			ID:          "phpstan",
			Name:        "phpstan",
			Description: "Run PHPStan static analysis",
			Entry:       "phpstan analyse --no-progress",
			Language:    "system",
			Types:       []string{"php"},
			Stages:      []string{"pre-commit"},
			// PHPStan needs paths on the command line unless phpstan.neon
			// sets parameters.paths, which most Composer projects lack
			// ("At least one path must be specified to analyse").
			PassFilenames: true,
			BuiltIn:       false,
			// phpstan is a top-level nixpkgs attribute; the old
			// `phpPackages.phpstan` is a removed throw-alias ("has been removed,
			// use phpstan instead") that fails devenv.nix evaluation.
			NixPackage: "phpstan",
		},
	}
}

// phpcsHook returns the PHP_CodeSniffer hook. A project ruleset is picked up
// by git-hooks.nix's built-in phpcs; without one PHP_CodeSniffer falls back
// to the PEAR standard, which rejects ordinary PSR-12 and Laravel code, so
// the hook runs PSR-12 explicitly.
func phpcsHook(config ecosystem.ModuleConfig) ecosystem.HookConfig {
	hook := ecosystem.HookConfig{
		ID:            "phpcs",
		Name:          "phpcs",
		Description:   "Run PHP_CodeSniffer to check coding standards",
		Entry:         "phpcs",
		Language:      "system",
		Types:         []string{"php"},
		Stages:        []string{"pre-commit"},
		PassFilenames: true,
		BuiltIn:       true,
	}
	if config.Extra(ExtraPHPCSConfig, "") != "true" {
		hook.Entry = "phpcs --standard=PSR12"
		hook.BuiltIn = false
		hook.NixPackage = "phpPackages.php-codesniffer"
	}
	return hook
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
			Name:           "composer",
			LockFile:       "composer.lock",
			InstallCommand: "composer install",
		},
	}
}

// WizardFields returns additional wizard form fields for PHP configuration.
func (m *Module) WizardFields() []ecosystem.WizardField {
	options := make([]ecosystem.WizardOption, 0, len(supportedPHPSeries))
	for _, s := range supportedPHPSeries {
		options = append(options, ecosystem.WizardOption{Label: s.String(), Value: s.String()})
	}
	return []ecosystem.WizardField{
		{
			Key:         types.SettingVersion,
			Label:       "PHP version",
			Description: "Select the PHP version to use",
			Type:        ecosystem.FieldTypeSelect,
			Options:     options,
			Default:     defaultPHPSeries.String(),
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

// phpPackage maps a configured version to the nixpkgs PHP attribute. The
// version may be a series ("8.4"), a full version ("8.4.1"), or a Composer
// constraint ("^8.2"); it resolves to the newest supported series satisfying
// it. An empty version selects the default series. Versions no supported
// series satisfies (including end-of-life ones) are an error rather than a
// silent substitution.
func phpPackage(version string) (string, error) {
	v := strings.TrimSpace(version)
	if v == "" {
		return defaultPHPSeries.nixAttr(), nil
	}
	series, ok := resolvePHPConstraint(v)
	if !ok {
		return "", fmt.Errorf("%w %q: supported versions are %s", ErrUnsupportedPHPVersion, version, supportedPHPList())
	}
	return series.nixAttr(), nil
}

// supportedPHPList renders supportedPHPSeries for messages.
func supportedPHPList() string {
	names := make([]string, len(supportedPHPSeries))
	for i, s := range supportedPHPSeries {
		names[i] = s.String()
	}
	return strings.Join(names, ", ")
}

// parsePHPConstraint reads composer.json and returns the raw require.php
// constraint. Returns an empty string if the field is not found or the file
// cannot be read or parsed.
func parsePHPConstraint(composerJSONPath string) string {
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

	return strings.TrimSpace(composerFile.Require["php"])
}

// resolvePHPConstraint returns the newest supported PHP series that can
// satisfy a Composer version constraint, and false when none can (or the
// constraint is not understood).
func resolvePHPConstraint(constraint string) (phpSeries, bool) {
	alternatives, ok := parseComposerConstraint(constraint)
	if !ok {
		return phpSeries{}, false
	}
	for _, s := range supportedPHPSeries {
		for _, alt := range alternatives {
			if alt.admitsSeries(s) {
				return s, true
			}
		}
	}
	return phpSeries{}, false
}

// versionBound is a single Composer version comparison, e.g. ">=8.1" or "^8.2".
type versionBound struct {
	op       string // "", "=", "==", "!=", ">", ">=", "<", "<=", "^", "~"
	parts    []int  // 1-3 numeric components as written
	wildcard bool   // trailing ".*" / ".x" (or a bare "*")
}

// conjunction is a set of bounds that must all hold (space/comma separated).
type conjunction []versionBound

// parseComposerConstraint splits a Composer constraint into OR-ed ("||" or
// "|") conjunctions of bounds. It returns false for syntax it does not
// understand, so callers never guess.
func parseComposerConstraint(constraint string) ([]conjunction, bool) {
	var out []conjunction
	for _, alt := range strings.Split(strings.ReplaceAll(constraint, "||", "|"), "|") {
		tokens := strings.FieldsFunc(alt, func(r rune) bool { return unicode.IsSpace(r) || r == ',' })
		var conj conjunction
		for i := 0; i < len(tokens); i++ {
			tok := tokens[i]
			// An operator separated from its version by whitespace (">= 8.1").
			if strings.Trim(tok, "<>=!^~") == "" && i+1 < len(tokens) {
				i++
				tok += tokens[i]
			}
			b, ok := parseVersionBound(tok)
			if !ok {
				return nil, false
			}
			conj = append(conj, b)
		}
		if len(conj) == 0 {
			return nil, false
		}
		out = append(out, conj)
	}
	return out, true
}

func parseVersionBound(tok string) (versionBound, bool) {
	var b versionBound
	for _, op := range []string{">=", "<=", "!=", "==", ">", "<", "=", "^", "~"} {
		if strings.HasPrefix(tok, op) {
			b.op, tok = op, tok[len(op):]
			break
		}
	}
	tok = strings.TrimPrefix(tok, "v")
	// Drop stability flags and pre-release suffixes ("8.2@dev", "8.2.0-beta").
	if i := strings.IndexAny(tok, "@-"); i >= 0 {
		tok = tok[:i]
	}
	if tok == "*" || tok == "x" {
		b.wildcard = true
		return b, b.op == ""
	}
	fields := strings.Split(tok, ".")
	if len(fields) > 3 {
		return versionBound{}, false
	}
	for i, f := range fields {
		if (f == "*" || f == "x") && i == len(fields)-1 && i > 0 {
			b.wildcard = true
			break
		}
		n, err := strconv.Atoi(f)
		if err != nil || n < 0 {
			return versionBound{}, false
		}
		b.parts = append(b.parts, n)
	}
	return b, !b.wildcard || b.op == ""
}

// admitsSeries reports whether some patch release of s satisfies every bound
// in the conjunction. Candidate patches are the extremes plus every patch
// number the bounds mention and its successor (so ">8.3.5 <8.3.7" finds
// 8.3.6), which is exact for the bound types supported.
func (c conjunction) admitsSeries(s phpSeries) bool {
	patches := []int{0, 1 << 20}
	for _, b := range c {
		if len(b.parts) == 3 {
			patches = append(patches, b.parts[2], b.parts[2]+1)
		}
	}
	for _, p := range patches {
		v := [3]int{s.major, s.minor, p}
		all := true
		for _, b := range c {
			if !b.admits(v) {
				all = false
				break
			}
		}
		if all {
			return true
		}
	}
	return false
}

// admits reports whether version v satisfies the bound.
func (b versionBound) admits(v [3]int) bool {
	var lo [3]int
	copy(lo[:], b.parts)
	switch {
	case b.wildcard:
		if len(b.parts) == 0 {
			return true
		}
		return cmpVersion(v, lo) >= 0 && cmpVersion(v, bumpAt(b.parts, len(b.parts)-1)) < 0
	case b.op == "^":
		// Next significant release: ^8.2 < 9.0.0, ^0.3 < 0.4.0.
		idx := 0
		for idx < len(b.parts)-1 && b.parts[idx] == 0 {
			idx++
		}
		return cmpVersion(v, lo) >= 0 && cmpVersion(v, bumpAt(b.parts, idx)) < 0
	case b.op == "~":
		// ~8.2 < 9.0.0; ~8.2.1 < 8.3.0; ~8 < 9.0.0.
		idx := len(b.parts) - 2
		if idx < 0 {
			idx = 0
		}
		return cmpVersion(v, lo) >= 0 && cmpVersion(v, bumpAt(b.parts, idx)) < 0
	case b.op == ">=":
		return cmpVersion(v, lo) >= 0
	case b.op == ">":
		return cmpVersion(v, lo) > 0
	case b.op == "<=":
		return cmpVersion(v, lo) <= 0
	case b.op == "<":
		return cmpVersion(v, lo) < 0
	case b.op == "!=":
		return cmpVersion(v, lo) != 0
	default: // "", "=", "==": match the components as written ("8.2" is any 8.2.x).
		for i, n := range b.parts {
			if v[i] != n {
				return false
			}
		}
		return true
	}
}

// bumpAt returns parts with component idx incremented and later components
// zeroed, as a full version (e.g. bumpAt([8 2 1], 1) = 8.3.0).
func bumpAt(parts []int, idx int) [3]int {
	var out [3]int
	copy(out[:idx+1], parts[:idx+1])
	out[idx]++
	return out
}

func cmpVersion(a, b [3]int) int {
	for i := range a {
		if a[i] != b[i] {
			if a[i] < b[i] {
				return -1
			}
			return 1
		}
	}
	return 0
}
