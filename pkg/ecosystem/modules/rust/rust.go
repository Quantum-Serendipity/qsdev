// Package rust implements the Rust ecosystem module for qsdev.
// It detects Rust projects via Cargo.toml, rust-toolchain.toml, and related files,
// then generates devenv.nix fragments, security configs, pre-commit hooks, deny rules,
// and CI commands for a hardened Rust development environment.
package rust

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
var _ ecosystem.SASTModule = (*Module)(nil)
var _ ecosystem.DevenvYamlInputProvider = (*Module)(nil)

// Module is the stateless Rust ecosystem module.
type Module struct{}

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Name returns the canonical module identifier.
func (m *Module) Name() string { return ecosystem.NameRust }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Rust" }

// Tier returns the implementation priority tier (1 = core).
func (m *Module) Tier() int { return 1 }

// Detect scans projectRoot for Rust ecosystem indicators.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	result := ecosystem.DetectionResult{
		SuggestedConfig: ecosystem.ModuleConfig{
			Extras: make(map[string]string),
		},
	}

	hasCargoToml := fileutil.FileExists(projectRoot, "Cargo.toml")
	hasCargoLock := fileutil.FileExists(projectRoot, "Cargo.lock")

	if !hasCargoToml && !hasCargoLock {
		return result
	}

	result.Detected = true

	if hasCargoToml {
		result.Confidence = ecosystem.ConfidenceCertain
		result.Evidence = append(result.Evidence, "Cargo.toml")
	}
	if hasCargoLock {
		if result.Confidence < ecosystem.ConfidenceProbable {
			result.Confidence = ecosystem.ConfidenceProbable
		}
		result.Evidence = append(result.Evidence, "Cargo.lock")
	}

	// Keep the toolchain spec only when devenv can express it; anything else
	// (a host-qualified toolchain, a custom toolchain name) falls back to
	// stable rather than producing an invalid devenv.nix.
	channel := parseToolchainChannel(projectRoot)
	if _, _, err := resolveToolchain(channel); err != nil {
		result.Evidence = append(result.Evidence, fmt.Sprintf("unsupported toolchain %q ignored", channel))
		channel = "stable"
	}
	result.SuggestedConfig.Extras["channel"] = channel

	return result
}

// DevenvNixFragment returns a Nix fragment that enables Rust in devenv.sh.
// The toolchain spec comes from the "channel" extra (rust-toolchain.toml) or,
// when that is unset or plain "stable", from Version (--rust-channel). It is
// split into devenv's channel enum and an optional version; see
// resolveToolchain.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	spec := config.Extra("channel", "")
	if (spec == "" || spec == "stable") && config.Version != "" {
		spec = config.Version
	}
	if spec == "" {
		spec = "stable"
	}

	channel, version, err := resolveToolchain(spec)
	if err != nil {
		return "", err
	}

	props := []ecosystem.NixProperty{
		{Key: "channel", Value: ecosystem.NixString(channel)},
	}
	if version != "" {
		props = append(props, ecosystem.NixProperty{Key: "version", Value: ecosystem.NixString(version)})
	}
	props = append(props, ecosystem.NixProperty{Key: "components", Value: `[ "rustfmt" "clippy" ]`})

	return ecosystem.BuildLanguageFragment(ecosystem.NixLangConfig{
		EnablePath: "languages.rust",
		Properties: props,
	}), nil
}

// DevenvYamlInputs contributes the rust-overlay flake input to devenv.yaml.
//
// DevenvNixFragment always emits languages.rust.channel, and devenv only
// accepts a non-"nixpkgs" Rust channel when the rust-overlay flake input is
// present. The input and the channel are therefore an invariant pair: the
// input is contributed here precisely because the channel is always emitted.
// Removing one without the other breaks `devenv` evaluation for Rust projects.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return []ecosystem.DevenvInput{
		{URL: "github:oxalica/rust-overlay", Follows: "nixpkgs"},
	}
}

// SecurityConfigs returns security-hardened configuration files for Rust.
func (m *Module) SecurityConfigs(config ecosystem.ModuleConfig) []types.GeneratedFile {
	var content strings.Builder

	content.WriteString("# Security-hardened Cargo configuration.\n")
	content.WriteString("# " + branding.GeneratedBy() + ".\n")
	content.WriteString("# Note: cargo build --locked enforces reproducible builds (Cargo stable)\n")
	content.WriteString("# Registry-side age-gating infrastructure stabilized in Cargo 1.94 (Mar 2026)\n\n")
	content.WriteString("[net]\n")
	content.WriteString("# Use the system git binary for fetches, which respects\n")
	content.WriteString("# credential helpers and SSH configuration.\n")
	content.WriteString("git-fetch-with-cli = true\n")

	if config.RegistryProxy != "" {
		content.WriteString("\n[source.crates-io]\n")
		content.WriteString("replace-with = \"corporate-proxy\"\n")
		content.WriteString("\n[source.corporate-proxy]\n")
		fmt.Fprintf(&content, "registry = \"%s\"\n", ecosystem.TOMLEscapeString(config.RegistryProxy))
	}

	if config.Extra("build_cache", "") == "sccache" {
		content.WriteString("\n[build]\n")
		content.WriteString("# Use sccache as the rustc wrapper for shared build caching.\n")
		content.WriteString("rustc-wrapper = \"sccache\"\n")
	}

	return []types.GeneratedFile{
		{
			Path:           ".cargo/config.toml",
			Content:        []byte(content.String()),
			Mode:           fileutil.ModeReadWrite,
			Strategy:       types.Overwrite,
			SkipValidation: true,
		},
	}
}

// PreCommitHooks returns pre-commit hook definitions for Rust.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:          "rustfmt",
			Name:        "rustfmt",
			Description: "Check Rust code formatting with rustfmt",
			Entry:       "cargo fmt -- --check",
			Language:    "system",
			Types:       []string{"rust"},
			Stages:      []string{"pre-commit"},
			BuiltIn:     true,
		},
		{
			ID:          "clippy",
			Name:        "clippy",
			Description: "Lint Rust code with clippy",
			Entry:       "cargo clippy -- -D warnings",
			Language:    "system",
			Types:       []string{"rust"},
			Stages:      []string{"pre-commit"},
			BuiltIn:     true,
		},
	}
}

// CICommands returns CI pipeline commands for Rust.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "cargo-build-locked",
			Command:     "cargo build --locked",
			Description: "Build with locked dependencies to ensure reproducibility",
			Phase:       ecosystem.CIPhaseInstall,
		},
		{
			Name:        "cargo-audit",
			Command:     "cargo audit",
			Description: "Audit dependencies for known security vulnerabilities",
			Phase:       ecosystem.CIPhaseScan,
		},
	}
}

// PackageManagers returns metadata about Cargo.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:                 "cargo",
			LockFile:             "Cargo.lock",
			InstallCommand:       "cargo build",
			FrozenInstallCommand: "cargo build --locked",
			AuditCommand:         "cargo audit",
			AgeGatingSupport:     false,
		},
	}
}

// WizardFields returns wizard form fields for the Rust ecosystem.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         "rust_channel",
			Label:       "Rust channel",
			Description: "Select the Rust release channel",
			Type:        ecosystem.FieldTypeSelect,
			Options: []ecosystem.WizardOption{
				{Label: "Stable", Value: "stable"},
				{Label: "Nightly", Value: "nightly"},
			},
			Default: "stable",
		},
	}
}

// VerificationCommands returns project verification commands for the Rust ecosystem.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{
		Build:  []string{"cargo build"},
		Test:   []string{"cargo test"},
		Lint:   []string{"cargo clippy -- -D warnings"},
		Format: []string{"cargo fmt -- --check"},
	}
}

// ManifestFiles returns manifest file metadata for the Rust ecosystem.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return []ecosystem.ManifestFileInfo{
		{
			Path:           "Cargo.toml",
			Ecosystem:      "cargo",
			VSSupported:    true,
			LockFile:       "Cargo.lock",
			LockFilePolicy: ecosystem.LockFilePolicyRecommended,
		},
	}
}

// --- helpers ---

// channelRegexp matches the channel key in a TOML file, e.g.:
//
//	channel = "stable"
var channelRegexp = regexp.MustCompile(`^\s*channel\s*=\s*"([^"]+)"`)

// rustReleaseRe matches a pinned stable release: "1.80" or "1.80.1".
var rustReleaseRe = regexp.MustCompile(`^[0-9]+\.[0-9]+(\.[0-9]+)?$`)

// rustDatedRe matches a dated beta or nightly toolchain such as
// "nightly-2024-05-01", capturing the channel and the date.
var rustDatedRe = regexp.MustCompile(`^(beta|nightly)-([0-9]{4}-[0-9]{2}-[0-9]{2})$`)

// resolveToolchain splits a rustup toolchain spec into devenv's
// languages.rust.channel, which only accepts the enum "stable", "beta" or
// "nightly", and languages.rust.version, which selects a release or date on
// that channel (empty means devenv's default, the latest one):
//
//	stable | beta | nightly      -> channel only
//	1.80.1                       -> channel "stable", version "1.80.1"
//	1.80                         -> channel "stable", version "1.80.0"
//	nightly-2024-05-01           -> channel "nightly", version "2024-05-01"
//
// rust-overlay only publishes full X.Y.Z stable versions, so a bare X.Y is
// pinned to its .0 release. Any other spec is rejected with an error.
func resolveToolchain(spec string) (channel, version string, err error) {
	switch {
	case spec == "stable" || spec == "beta" || spec == "nightly":
		return spec, "", nil
	case rustReleaseRe.MatchString(spec):
		if strings.Count(spec, ".") == 1 {
			spec += ".0"
		}
		return "stable", spec, nil
	}
	if m := rustDatedRe.FindStringSubmatch(spec); m != nil {
		return m[1], m[2], nil
	}
	return "", "", fmt.Errorf("unsupported Rust toolchain %q: want stable, beta, nightly, a release such as 1.80.0, or a dated channel such as nightly-2024-05-01", spec)
}

// parseToolchainChannel extracts the Rust toolchain channel from
// rust-toolchain.toml (preferred) or the legacy rust-toolchain file.
// It returns "stable" if neither file provides a channel.
func parseToolchainChannel(projectRoot string) string {
	// Prefer rust-toolchain.toml (TOML format).
	if ch := parseToolchainToml(filepath.Join(projectRoot, "rust-toolchain.toml")); ch != "" {
		return ch
	}

	// Fall back to legacy rust-toolchain (plain text).
	if ch := parseLegacyToolchain(filepath.Join(projectRoot, "rust-toolchain")); ch != "" {
		return ch
	}

	return "stable"
}

// parseToolchainToml reads a rust-toolchain.toml file and extracts the channel
// using a regex. Returns "" if the file does not exist or no channel is found.
func parseToolchainToml(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	for line := range strings.SplitSeq(string(data), "\n") {
		if matches := channelRegexp.FindStringSubmatch(line); len(matches) == 2 {
			return matches[1]
		}
	}
	return ""
}

// parseLegacyToolchain reads a plain-text rust-toolchain file and returns
// the trimmed first line as the channel. Returns "" if the file does not exist
// or is empty.
func parseLegacyToolchain(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	channel := strings.TrimSpace(string(data))
	if channel == "" {
		return ""
	}
	// Take only the first line.
	if idx := strings.IndexByte(channel, '\n'); idx != -1 {
		channel = strings.TrimSpace(channel[:idx])
	}
	return channel
}

// SemgrepRuleSets returns Semgrep rule set identifiers relevant to Rust projects.
func (m *Module) SemgrepRuleSets() []string {
	return []string{"p/rust", "p/owasp-top-ten"}
}
