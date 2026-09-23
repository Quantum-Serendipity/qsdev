// Package ruby implements the Ruby (Bundler) ecosystem module for
// qsdev. It detects Ruby projects by scanning for
// Gemfile and Gemfile.lock, generates devenv.nix fragments with Bundler
// support and Bundler hardening env vars, produces a security-hardened
// RubyGems configuration file, and provides pre-commit hooks, CI commands, deny rules, and wizard
// fields for the Ruby toolchain.
package ruby

import (
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

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.WizardFieldProvider = (*Module)(nil)
var _ ecosystem.ManifestFileProvider = (*Module)(nil)
var _ ecosystem.DevenvYamlInputProvider = (*Module)(nil)
var _ ecosystem.DenyRuleProvider = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Module implements ecosystem.EcosystemModule for the Ruby programming language.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "ruby" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Ruby" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 2 }

// Detect scans projectRoot for Gemfile and Gemfile.lock files and reads
// the Ruby version from a .ruby-version file if present.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	gemfile := filepath.Join(projectRoot, "Gemfile")
	gemfileLock := filepath.Join(projectRoot, "Gemfile.lock")

	hasGemfile := fileutil.FileExists(gemfile)
	hasLock := fileutil.FileExists(gemfileLock)

	if !hasGemfile && !hasLock {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	confidence := ecosystem.ConfidenceProbable
	var evidence []string

	if hasGemfile {
		confidence = ecosystem.ConfidenceCertain
		evidence = append(evidence, "Gemfile found")
	}
	if hasLock {
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		evidence = append(evidence, "Gemfile.lock found")
	}

	version := parseRubyVersion(projectRoot)
	if version != "" {
		evidence = append(evidence, fmt.Sprintf("Ruby version %s (from .ruby-version)", version))
	}

	suggested := ecosystem.ModuleConfig{Version: version}
	if bundlesRubocop(gemfileLock) {
		evidence = append(evidence, "rubocop locked in Gemfile.lock")
		suggested.Extras = map[string]string{ExtraBundledRubocop: "true"}
	}

	return ecosystem.DetectionResult{
		Detected:        true,
		Confidence:      confidence,
		Evidence:        evidence,
		SuggestedConfig: suggested,
	}
}

// ExtraBundledRubocop marks a project whose Gemfile.lock pins rubocop.
const ExtraBundledRubocop = "bundled_rubocop"

// lockedRubocopRe matches rubocop's entry in a Gemfile.lock specs section
// ("    rubocop (1.66.1)").
var lockedRubocopRe = regexp.MustCompile(`(?m)^ {4}rubocop \(`)

// bundlesRubocop reports whether the Gemfile.lock at path pins rubocop.
func bundlesRubocop(path string) bool {
	data, err := os.ReadFile(path)
	return err == nil && lockedRubocopRe.Match(data)
}

// nixpkgsRubyInput is the flake input devenv resolves languages.ruby.version
// against (config.lib.getInput for "nixpkgs-ruby"). It must be present in
// devenv.yaml whenever the fragment pins a version.
const nixpkgsRubyInput = "github:bobvanderlinden/nixpkgs-ruby"

// rubyVersionRe matches the MRI version strings nixpkgs-ruby publishes as
// "ruby-<version>" attributes: 3.3, 3.3.0, 3.4.0-preview1.
var rubyVersionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+(\.[0-9]+)?(-[0-9A-Za-z.]+)?$`)

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Ruby language support with Bundler enabled. A configured Ruby version
// (from .ruby-version or the answers) pins languages.ruby.version; Bundler
// hardening is exported as environment variables rather than written to the
// user-owned .bundle/config.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	version, err := rubyVersion(config)
	if err != nil {
		return "", err
	}

	var props []ecosystem.NixProperty
	if version != "" {
		props = append(props, ecosystem.NixProperty{Key: "version", Value: ecosystem.NixString(version)})
	}
	props = append(props, ecosystem.NixProperty{Key: "bundler.enable", Value: "true"})

	return ecosystem.BuildLanguageFragment(ecosystem.NixLangConfig{
		EnablePath: "languages.ruby",
		Properties: props,
		EnvVars: []ecosystem.NixEnvVar{
			{Key: "BUNDLE_FROZEN", Value: ecosystem.NixString("true"), Comment: "Refuse to modify Gemfile.lock during bundle install"},
			{Key: "BUNDLE_DISABLE_EXEC_LOAD", Value: ecosystem.NixString("true"), Comment: "Run binstubs in a subprocess instead of Kernel.load"},
			// RubyGems never reads a .gemrc from the working directory, only
			// /etc/gemrc, ~/.gemrc and the files GEMRC names.
			{Key: "GEMRC", Value: `"${config.devenv.root}/.gemrc"`, Comment: "Load the project .gemrc generated by SecurityConfigs"},
		},
	}), nil
}

// DevenvYamlInputs contributes the nixpkgs-ruby flake input when the fragment
// pins languages.ruby.version; devenv refuses to evaluate the version option
// without it. The input and the version line are an invariant pair.
func (m *Module) DevenvYamlInputs(config ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	if version, err := rubyVersion(config); err != nil || version == "" {
		return nil
	}
	return []ecosystem.DevenvInput{{URL: nixpkgsRubyInput, Follows: "nixpkgs"}}
}

// rubyVersion returns the normalized Ruby version from config ("" when unset),
// or an error when the configured value is not an MRI version nixpkgs-ruby
// can provide.
func rubyVersion(config ecosystem.ModuleConfig) (string, error) {
	v := normalizeRubyVersion(config.Version)
	if v == "" {
		return "", nil
	}
	if !rubyVersionRe.MatchString(v) {
		return "", fmt.Errorf("invalid Ruby version %q: want an MRI version such as 3.3 or 3.3.0", config.Version)
	}
	return v, nil
}

// normalizeRubyVersion trims whitespace and the optional "ruby-" prefix that
// .ruby-version files commonly carry.
func normalizeRubyVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "ruby-")
}

// SecurityConfigs returns a security-hardened RubyGems configuration file,
// loaded through the GEMRC variable the fragment exports. It sets the default
// source list only: `gem --source` still overrides it and Bundler uses the
// Gemfile's source, so plain-HTTP sources are refused by DenyRules instead.
// It uses the Skip strategy so an existing user .gemrc is never replaced.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	gemrc := "# Security-hardened RubyGems configuration.\n" +
		"# " + branding.GeneratedBy() + ".\n" +
		"# Default gem source (HTTPS); loaded via GEMRC in the devenv shell.\n" +
		"---\n" +
		":sources:\n" +
		"- https://rubygems.org\n" +
		"gem: --no-document\n"

	return []types.GeneratedFile{
		{
			Path:     ".gemrc",
			Content:  []byte(gemrc),
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.Skip,
		},
	}
}

// PreCommitHooks returns pre-commit hook definitions for the Ruby ecosystem.
//
// A project that bundles rubocop runs its own locked version through Bundler:
// only that one can load the gems its .rubocop.yml names (inherit_gem:
// rubocop-rails-omakase, plugins: rubocop-rails, ...), which a standalone
// nixpkgs rubocop cannot see. Other projects get the nixpkgs rubocop.
// --force-exclusion keeps .rubocop.yml Exclude patterns effective for the
// file names the hook passes explicitly.
func (m *Module) PreCommitHooks(config ecosystem.ModuleConfig) []ecosystem.HookConfig {
	hook := ecosystem.HookConfig{
		ID:            "rubocop",
		Name:          "rubocop",
		Description:   "Run RuboCop linter and formatter for Ruby",
		Entry:         "rubocop --autocorrect --force-exclusion",
		Language:      "system",
		Types:         []string{"ruby"},
		Stages:        []string{"pre-commit"},
		PassFilenames: true,
		BuiltIn:       false,
		NixPackage:    "rubocop",
	}
	if config.Extra(ExtraBundledRubocop, "") == "true" {
		hook.Entry = "bundle exec " + hook.Entry
		hook.NixPackage = ""
	}
	return []ecosystem.HookConfig{hook}
}

// DenyRules returns Claude Code deny-rule patterns for the Ruby ecosystem:
// gem commands that name a plain-HTTP source (gem install --source http://...,
// gem sources --add http://...), which .gemrc cannot prevent.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"Bash(gem *http://*)",
	}
}

// CICommands returns CI pipeline commands for the Ruby ecosystem.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "bundle-install",
			Command:     "bundle install --frozen",
			Description: "Install Ruby dependencies from frozen Gemfile.lock",
			Phase:       ecosystem.CIPhaseInstall,
		},
		{
			Name:        "bundle-audit",
			Command:     "bundle audit check --update",
			Description: "Audit Ruby dependencies for known vulnerabilities",
			Phase:       ecosystem.CIPhaseScan,
		},
	}
}

// PackageManagers returns metadata about Ruby's Bundler package manager.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:                 "bundler",
			LockFile:             "Gemfile.lock",
			InstallCommand:       "bundle install",
			FrozenInstallCommand: "bundle install --frozen",
			AuditCommand:         "bundle audit check",
			AgeGatingSupport:     false,
		},
	}
}

// WizardFields returns additional wizard form fields for Ruby configuration.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         "ruby_version",
			Label:       "Ruby version",
			Description: "Specify the Ruby version to use (e.g. 3.3)",
			Type:        ecosystem.FieldTypeInput,
			Default:     "",
		},
	}
}

// VerificationCommands returns test and lint commands for Ruby projects.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{
		Test: []string{"bundle exec rspec"},
		Lint: []string{"bundle exec rubocop"},
	}
}

// ManifestFiles returns the Gemfile manifest file for Ruby projects.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return []ecosystem.ManifestFileInfo{{Path: "Gemfile", Ecosystem: "bundler", LockFile: "Gemfile.lock", LockFilePolicy: ecosystem.LockFilePolicyRequired}}
}

// parseRubyVersion reads .ruby-version in projectRoot and returns the
// normalized version from its first line (e.g. "ruby-3.3.0" becomes "3.3.0").
// Returns an empty string if the file does not exist, cannot be read, or does
// not name an MRI version (e.g. "jruby-9.4"), so repository content can never
// inject an unusable value into devenv.nix.
func parseRubyVersion(projectRoot string) string {
	data, err := os.ReadFile(filepath.Join(projectRoot, ".ruby-version"))
	if err != nil {
		return ""
	}
	first, _, _ := strings.Cut(string(data), "\n")
	v := normalizeRubyVersion(first)
	if !rubyVersionRe.MatchString(v) {
		return ""
	}
	return v
}
