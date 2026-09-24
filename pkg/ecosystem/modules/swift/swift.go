// Package swift implements the Swift (SPM) ecosystem module for
// qsdev. It detects Swift projects by scanning for
// Package.swift, Package.resolved, and *.xcodeproj, generates devenv.nix
// fragments with Swift language support and SE-0391 TOFU commentary, and
// provides pre-commit hooks, CI commands, deny rules, and package manager
// metadata for the Swift toolchain.
package swift

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.DenyRuleProvider = (*Module)(nil)
var _ ecosystem.ManifestFileProvider = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Module implements ecosystem.EcosystemModule for the Swift programming language.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "swift" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Swift" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 3 }

// Detect scans projectRoot for Swift ecosystem indicators: Package.swift,
// Package.resolved, and *.xcodeproj directories.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	hasPackageSwift := fileutil.FileExists(projectRoot, "Package.swift")
	hasPackageResolved := fileutil.FileExists(projectRoot, "Package.resolved")
	xcodeprojMatches, _ := filepath.Glob(filepath.Join(projectRoot, "*.xcodeproj"))
	hasXcodeproj := len(xcodeprojMatches) > 0

	if !hasPackageSwift && !hasPackageResolved && !hasXcodeproj {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	confidence := ecosystem.ConfidenceProbable
	var evidence []string

	if hasPackageSwift {
		confidence = ecosystem.ConfidenceCertain
		evidence = append(evidence, "Package.swift found")
	}
	if hasPackageResolved {
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		evidence = append(evidence, "Package.resolved found")
	}
	if hasXcodeproj {
		if confidence < ecosystem.ConfidenceProbable {
			confidence = ecosystem.ConfidenceProbable
		}
		evidence = append(evidence, "*.xcodeproj found")
	}

	toolsVersion := parseToolsVersion(filepath.Join(projectRoot, "Package.swift"))
	if toolsVersion != "" {
		evidence = append(evidence, fmt.Sprintf("swift-tools-version %s (from Package.swift)", toolsVersion))
	}

	return ecosystem.DetectionResult{
		Detected:        true,
		Confidence:      confidence,
		Evidence:        evidence,
		SuggestedConfig: ecosystem.ModuleConfig{Version: toolsVersion},
	}
}

// toolsVersionRe matches SwiftPM's mandatory first-line tools-version
// comment: "// swift-tools-version:5.9", "// swift-tools-version: 6.0".
var toolsVersionRe = regexp.MustCompile(`^//\s*swift-tools-version:\s*([0-9]+\.[0-9]+(?:\.[0-9]+)?)`)

// swiftVersionRe validates a configured Swift version before it is written
// into devenv.nix.
var swiftVersionRe = regexp.MustCompile(`^[0-9]+\.[0-9]+(\.[0-9]+)?$`)

// parseToolsVersion returns the swift-tools-version Package.swift declares on
// its first line, or "" when the file or the declaration is missing.
func parseToolsVersion(packageSwift string) string {
	f, err := os.Open(packageSwift)
	if err != nil {
		return ""
	}
	defer f.Close() //nolint:errcheck // read-only
	sc := bufio.NewScanner(f)
	if !sc.Scan() {
		return ""
	}
	m := toolsVersionRe.FindStringSubmatch(strings.TrimSpace(sc.Text()))
	if m == nil {
		return ""
	}
	return m[1]
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Swift language support. Includes a comment about SE-0391 TOFU
// (Trust On First Use) for package integrity.
//
// config.Version is the swift-tools-version the package requires. nixpkgs'
// Swift can lag it (SwiftPM refuses to load a tools-version newer than the
// toolchain), so the fragment compares versions at evaluation time and warns
// with the remedy instead of leaving `swift build` to fail with a
// tools-version error.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	var b strings.Builder
	b.WriteString("  languages.swift.enable = true;\n")
	if v := config.Version; v != "" {
		if !swiftVersionRe.MatchString(v) {
			return "", fmt.Errorf("invalid Swift tools version %q: want a version such as 5.9 or 6.0", v)
		}
		fmt.Fprintf(&b, `  languages.swift.package =
    if lib.versionOlder pkgs.swift.version "%[1]s" then
      lib.warn "Package.swift requires swift-tools-version %[1]s but nixpkgs provides Swift ${pkgs.swift.version}; build with the host Xcode or a swiftly-managed toolchain" pkgs.swift
    else
      pkgs.swift;
`, v)
	}
	b.WriteString("  # SE-0391: Package.resolved provides TOFU (Trust On First Use) integrity.\n")
	b.WriteString("  # Always commit Package.resolved to version control.\n")
	return b.String(), nil
}

// SecurityConfigs returns generated security configuration files.
// Swift relies on Package.resolved for integrity; no additional config needed.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// PreCommitHooks returns pre-commit hook definitions for the Swift ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "swiftformat",
			Name:          "swiftformat",
			Description:   "Lint Swift source code with SwiftFormat",
			Entry:         "swiftformat --lint",
			Language:      "system",
			Types:         []string{"swift"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true, // the formatter needs file operands
			// Custom hook (BuiltIn:false): NixPackage provisions the binary so
			// the emitted `entry` resolves at commit time.
			BuiltIn:    false,
			NixPackage: "swiftformat",
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the Swift ecosystem.
// These prevent direct dependency updates outside of controlled workflows.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		// No space before the glob: bare `swift package update` bumps every
		// dependency past Package.resolved, so the rule must match it too.
		"Bash(swift package update*)",
	}
}

// CICommands returns CI pipeline commands for the Swift ecosystem.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "swift-package-resolve",
			Command:     "swift package resolve",
			Description: "Resolve Swift package dependencies",
			Phase:       ecosystem.CIPhaseInstall,
		},
		{
			Name:        "swift-build",
			Command:     "swift build",
			Description: "Build the Swift project",
			Phase:       ecosystem.CIPhaseTest,
		},
	}
}

// PackageManagers returns metadata about the Swift Package Manager.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:     "spm",
			LockFile: "Package.resolved",
		},
	}
}

// VerificationCommands returns build and test commands for Swift projects.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{
		Build: []string{"swift build"},
		Test:  []string{"swift test"},
	}
}

// ManifestFiles returns the Package.swift manifest file for Swift projects.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return []ecosystem.ManifestFileInfo{{Path: "Package.swift", Ecosystem: "spm", LockFile: "Package.resolved", LockFilePolicy: ecosystem.LockFilePolicyRecommended}}
}
