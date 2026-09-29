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
	"slices"
	"strings"

	"github.com/BurntSushi/toml"

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
var _ ecosystem.PackageProvider = (*Module)(nil)
var _ ecosystem.ReadDenyRuleProvider = (*Module)(nil)

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
	channel, file := parseToolchainChannel(projectRoot)
	if _, _, err := resolveToolchain(channel); err != nil {
		result.Evidence = append(result.Evidence, fmt.Sprintf("unsupported toolchain %q ignored", channel))
		channel, file = "stable", ""
	}
	result.SuggestedConfig.Extras["channel"] = channel
	if file != "" {
		// The file also carries components, targets and profile, which
		// devenv reads itself through languages.rust.toolchainFile.
		result.Evidence = append(result.Evidence, file)
		result.SuggestedConfig.Extras[ExtraToolchainFile] = file
	}

	return result
}

// ExtraToolchainFile is the ModuleConfig extra naming the project's rustup
// toolchain file ("rust-toolchain.toml" or "rust-toolchain"). When set, the
// fragment hands the file to devenv instead of restating its channel.
const ExtraToolchainFile = "toolchain_file"

// toolchainFiles are the rustup toolchain file names, in rustup's order of
// precedence.
var toolchainFiles = []string{"rust-toolchain.toml", "rust-toolchain"}

// DevenvNixFragment returns a Nix fragment that enables Rust in devenv.sh.
//
// When the project has a rustup toolchain file (the "toolchain_file" extra)
// and no explicit Version, the fragment sets languages.rust.toolchainFile so
// devenv builds the toolchain the file describes; devenv rejects combining it
// with channel or version. Otherwise the toolchain spec comes from the
// "channel" extra or, when that is unset or plain "stable", from Version
// (--rust-channel), split into devenv's channel enum and an optional version;
// see resolveToolchain.
//
// components is never set: devenv's default is the full toolchain (rustc,
// cargo, clippy, rustfmt, rust-analyzer), and any explicit list replaces it.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	if file := config.Extra(ExtraToolchainFile, ""); file != "" && config.Version == "" {
		if !slices.Contains(toolchainFiles, file) {
			return "", fmt.Errorf("unsupported Rust toolchain file %q: want one of %s", file, strings.Join(toolchainFiles, ", "))
		}
		return ecosystem.BuildLanguageFragment(ecosystem.NixLangConfig{
			EnablePath: "languages.rust",
			Properties: []ecosystem.NixProperty{{Key: "toolchainFile", Value: "./" + file}},
		}), nil
	}

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

	return ecosystem.BuildLanguageFragment(ecosystem.NixLangConfig{
		EnablePath: "languages.rust",
		Properties: props,
	}), nil
}

// DevenvYamlInputs contributes the rust-overlay flake input to devenv.yaml.
//
// DevenvNixFragment always emits either a non-"nixpkgs" languages.rust.channel
// or languages.rust.toolchainFile, and devenv builds both from the
// rust-overlay flake input. The input and those options are therefore an
// invariant pair: the input is contributed here precisely because one of them
// is always emitted. Removing one without the other breaks `devenv`
// evaluation for Rust projects.
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
		fmt.Fprintf(&content, "registry = \"%s\"\n", ecosystem.TOMLEscapeString(cargoRegistryIndexURL(config.RegistryProxy)))
	}

	if usesSccache(config) {
		content.WriteString("\n[build]\n")
		content.WriteString("# Use sccache as the rustc wrapper for shared build caching.\n")
		content.WriteString("rustc-wrapper = \"sccache\"\n")
	}

	return []types.GeneratedFile{
		{
			Path:           ".cargo/config.toml",
			Content:        []byte(content.String()),
			Mode:           fileutil.ModeReadWrite,
			Strategy:       types.Skip,
			SkipValidation: true,
		},
	}
}

// cargoRegistryIndexURL returns the index URL cargo should use for a registry
// proxy. Cargo picks the index protocol from the URL: only a "sparse+" prefix
// selects the sparse (HTTP) protocol, and anything else is fetched with git.
// Registry proxies (Nexus, Artifactory, ...) serve a sparse index over HTTP,
// so a plain http(s) URL gets the prefix and the trailing slash cargo requires
// of sparse index URLs. URLs that already name a protocol (sparse+, git+,
// ssh://, file://) or a git repository (*.git) are kept as-is.
func cargoRegistryIndexURL(proxy string) string {
	lower := strings.ToLower(proxy)
	if !strings.HasPrefix(lower, "https://") && !strings.HasPrefix(lower, "http://") {
		return proxy
	}
	if strings.HasSuffix(strings.TrimRight(lower, "/"), ".git") {
		return proxy
	}
	if !strings.HasSuffix(proxy, "/") {
		proxy += "/"
	}
	return "sparse+" + proxy
}

// usesSccache reports whether the build_cache extra (set explicitly or from
// infrastructure.build_cache) selects sccache.
func usesSccache(config ecosystem.ModuleConfig) bool {
	return config.Extra(ecosystem.ExtraBuildCache, "") == "sccache"
}

// DevenvPackages returns cargo-audit, which the generated CI runs as
// `cargo audit` in the devenv shell, plus sccache when it is the configured
// build cache: .cargo/config.toml then names it as the rustc wrapper, and
// cargo fails every compile when the wrapper is not on PATH.
func (m *Module) DevenvPackages(config ecosystem.ModuleConfig) []string {
	pkgs := []string{"cargo-audit"}
	if usesSccache(config) {
		pkgs = append(pkgs, "sccache")
	}
	return pkgs
}

// ReadDenyRules returns the Cargo credential files, which hold crates.io and
// alternate-registry tokens able to publish new versions of the user's crates.
func (m *Module) ReadDenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"~/.cargo/credentials.toml",
		// Cargo still reads the pre-1.39 name when the .toml file is absent.
		"~/.cargo/credentials",
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
			Name:           "cargo",
			LockFile:       "Cargo.lock",
			InstallCommand: "cargo build",
		},
	}
}

// WizardFields returns wizard form fields for the Rust ecosystem.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         "channel",
			Label:       "Rust channel",
			Description: "Select the Rust release channel",
			Type:        ecosystem.FieldTypeSelect,
			Options: []ecosystem.WizardOption{
				{Label: "Stable", Value: "stable"},
				{Label: "Beta", Value: "beta"},
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

// parseToolchainChannel extracts the Rust toolchain channel from the
// project's rustup toolchain file, rust-toolchain.toml (preferred) or the
// legacy rust-toolchain, and returns it with the name of the file it came
// from. It returns "stable" and no file if neither provides a channel.
func parseToolchainChannel(projectRoot string) (channel, file string) {
	for _, name := range toolchainFiles {
		data, err := os.ReadFile(filepath.Join(projectRoot, name))
		if err != nil {
			continue
		}
		if ch := toolchainFileChannel(data); ch != "" {
			return ch, name
		}
	}
	return "stable", ""
}

// rustupToolchainFile is the TOML form of a rustup toolchain file.
type rustupToolchainFile struct {
	Toolchain struct {
		Channel string `toml:"channel"`
	} `toml:"toolchain"`
}

// toolchainFileChannel returns the channel a rustup toolchain file selects,
// or "" when it names none. Like rustup (and rust-overlay, which devenv uses
// to read the file), it accepts TOML in either file name and falls back to
// the legacy single-line form ("nightly") only for content that is not TOML.
func toolchainFileChannel(data []byte) string {
	var tf rustupToolchainFile
	if _, err := toml.Decode(string(data), &tf); err == nil {
		return strings.TrimSpace(tf.Toolchain.Channel)
	}
	content := strings.TrimSpace(string(data))
	if strings.ContainsAny(content, "\r\n") {
		// Multi-line content that is not valid TOML is not a legacy file.
		return ""
	}
	return content
}

// SemgrepRuleSets returns Semgrep rule set identifiers relevant to Rust projects.
func (m *Module) SemgrepRuleSets() []string {
	return []string{"p/rust", "p/owasp-top-ten"}
}
