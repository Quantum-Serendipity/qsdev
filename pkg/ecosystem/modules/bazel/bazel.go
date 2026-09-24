// Package bazel implements the Bazel build system ecosystem module for
// qsdev. It detects Bazel projects by scanning for
// MODULE.bazel, WORKSPACE, WORKSPACE.bazel, and .bazelrc files, generates
// devenv.nix fragments with Bazel and Buildifier packages, produces a
// security-hardened .bazelrc.qsdev imported from .bazelrc, and provides pre-commit hooks,
// CI commands, deny rules, and package manager metadata for the Bazel toolchain.
package bazel

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

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.PackageProvider = (*Module)(nil)
var _ ecosystem.PackageExprProvider = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Module implements ecosystem.EcosystemModule for the Bazel build system.
type Module struct{}

// Name returns the canonical ecosystem identifier.
func (m *Module) Name() string { return "bazel" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "Bazel" }

// Tier returns the implementation priority tier.
func (m *Module) Tier() int { return 3 }

// Detect scans projectRoot for Bazel ecosystem indicators: MODULE.bazel,
// WORKSPACE, WORKSPACE.bazel, and .bazelrc files.
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	hasModuleBazel := fileutil.FileExists(projectRoot, "MODULE.bazel")
	hasWorkspace := fileutil.FileExists(projectRoot, "WORKSPACE")
	hasWorkspaceBazel := fileutil.FileExists(projectRoot, "WORKSPACE.bazel")
	hasBazelrc := fileutil.FileExists(projectRoot, ".bazelrc")

	if !hasModuleBazel && !hasWorkspace && !hasWorkspaceBazel && !hasBazelrc {
		return ecosystem.DetectionResult{
			Detected:   false,
			Confidence: ecosystem.ConfidenceAbsent,
		}
	}

	confidence := ecosystem.ConfidenceProbable
	var evidence []string

	if hasModuleBazel {
		confidence = ecosystem.ConfidenceCertain
		evidence = append(evidence, "MODULE.bazel found")
	}
	if hasWorkspace {
		confidence = ecosystem.ConfidenceCertain
		evidence = append(evidence, "WORKSPACE found")
	}
	if hasWorkspaceBazel {
		confidence = ecosystem.ConfidenceCertain
		evidence = append(evidence, "WORKSPACE.bazel found")
	}
	if hasBazelrc {
		evidence = append(evidence, ".bazelrc found")
	}

	version := parseBazelVersion(filepath.Join(projectRoot, ".bazelversion"))
	if version != "" {
		evidence = append(evidence, fmt.Sprintf("Bazel version %s (from .bazelversion)", version))
	}

	return ecosystem.DetectionResult{
		Detected:        true,
		Confidence:      confidence,
		Evidence:        evidence,
		SuggestedConfig: ecosystem.ModuleConfig{Version: version},
	}
}

// bazelVersionRe matches a Bazel release version as .bazelversion pins it
// (8.2.1, 8.0.0rc1, 7.x) and captures the major version.
var bazelVersionRe = regexp.MustCompile(`^([0-9]+)(\.[0-9A-Za-z*]+)*(-?[0-9A-Za-z]+)?$`)

// parseBazelVersion returns the Bazel version .bazelversion pins, or "" when
// the file is absent or names something else (a fork such as
// "mycorp/7.0.0", "latest", a commit).
func parseBazelVersion(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close() //nolint:errcheck // read-only
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if bazelVersionRe.MatchString(line) {
			return line
		}
		return ""
	}
	return ""
}

// DevenvPackages returns the Nix packages required for the Bazel ecosystem.
// Bazel itself is selected by DevenvPackageExprs.
func (m *Module) DevenvPackages(_ ecosystem.ModuleConfig) []string {
	return []string{"buildifier"}
}

// DevenvPackageExprs returns the Bazel package matching the configured
// (.bazelversion) major version, nixpkgs' bazel_<major>, so a Bazel 8 project
// does not get Bazel 7 (which cannot read Bazel 8 MODULE.bazel features and
// rewrites MODULE.bazel.lock in its own format). A major nixpkgs does not ship (or
// has removed, like bazel_6) falls back to its default Bazel with an evaluation warning; no version
// selects the nixpkgs default.
func (m *Module) DevenvPackageExprs(config ecosystem.ModuleConfig) []string {
	parts := bazelVersionRe.FindStringSubmatch(config.Version)
	if parts == nil {
		return []string{"pkgs.bazel"}
	}
	return []string{ecosystem.NixPkgsAttrOr("bazel_"+parts[1], "pkgs.bazel",
		"Bazel "+parts[1]+" is not in nixpkgs; using Bazel ${pkgs.bazel.version}")}
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for Bazel support. Packages are provided via DevenvPackages.
func (m *Module) DevenvNixFragment(_ ecosystem.ModuleConfig) (string, error) {
	return "", nil
}

// qsdevBazelrc is the qsdev-owned Bazel rc file holding the hardening flags.
// The project's own .bazelrc (a detection marker, typically holding remote
// cache, toolchain and --config definitions) stays user-owned and pulls it in
// with bazelrcImport.
const qsdevBazelrc = ".bazelrc.qsdev"

// bazelrcImport is the line that makes Bazel read qsdevBazelrc. try-import
// tolerates the file being absent, so the line is always safe to keep.
const bazelrcImport = "try-import %workspace%/" + qsdevBazelrc

// SecurityConfigs returns the security-hardened .bazelrc.qsdev plus a minimal
// .bazelrc that imports it. The .bazelrc uses the Skip strategy so an existing
// user .bazelrc is never replaced; such projects add bazelrcImport themselves
// (the instruction is in the generated .bazelrc.qsdev header).
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	hardened := "# Security-hardened Bazel configuration.\n" +
		"# " + branding.GeneratedBy() + "; regenerated on update, do not edit.\n" +
		"# Requires: Bazel >= 7.0 for bzlmod lockfile support.\n" +
		"# Activate by adding this line to your .bazelrc:\n" +
		"#   " + bazelrcImport + "\n" +
		"\n" +
		"# MODULE.bazel.lock must match MODULE.bazel: an unreviewed dependency\n" +
		"# change fails every command instead of silently rewriting the lock.\n" +
		"# After an intended MODULE.bazel edit, refresh it explicitly with:\n" +
		"#   bazel mod deps --lockfile_mode=update\n" +
		"common --lockfile_mode=error\n" +
		"\n" +
		"build --spawn_strategy=sandboxed\n" +
		"build --sandbox_default_allow_network=false\n"

	bazelrc := "# Bazel configuration.\n" +
		"# Security hardening is maintained by qsdev in " + qsdevBazelrc + ".\n" +
		bazelrcImport + "\n"

	return []types.GeneratedFile{
		{
			Path:     qsdevBazelrc,
			Content:  []byte(hardened),
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.Overwrite,
		},
		{
			Path:     ".bazelrc",
			Content:  []byte(bazelrc),
			Mode:     fileutil.ModeReadWrite,
			Strategy: types.Skip,
		},
	}
}

// PreCommitHooks returns pre-commit hook definitions for the Bazel ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "buildifier",
			Name:          "buildifier",
			Description:   "Lint Bazel BUILD and .bzl files with Buildifier",
			Entry:         "buildifier -lint=warn",
			Language:      "system",
			Types:         []string{"bazel"},
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			// MODULE.bazel (and *.MODULE.bazel includes) is the Bzlmod
			// dependency manifest, the default since Bazel 7.
			Files:   `(^|/)(BUILD|WORKSPACE|MODULE)(\.bazel)?$|\.bzl$|\.MODULE\.bazel$`,
			BuiltIn: false,
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the Bazel ecosystem.
// Prevents running arbitrary external repository targets.
func (m *Module) DenyRules(_ ecosystem.ModuleConfig) []string {
	return []string{
		"Bash(bazel run @*)",
	}
}

// CICommands returns CI pipeline commands for the Bazel ecosystem.
func (m *Module) CICommands(_ ecosystem.ModuleConfig) []ecosystem.CICommand {
	return []ecosystem.CICommand{
		{
			Name:        "bazel-build",
			Command:     "bazel build //...",
			Description: "Build all Bazel targets",
			Phase:       ecosystem.CIPhaseTest,
		},
		{
			Name:        "bazel-test",
			Command:     "bazel test //...",
			Description: "Run all Bazel tests",
			Phase:       ecosystem.CIPhaseTest,
		},
	}
}

// PackageManagers returns metadata about the Bazel module system (bzlmod).
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:     "bzlmod",
			LockFile: "MODULE.bazel.lock",
		},
	}
}

// VerificationCommands returns an empty set. Bazel does not define standard
// verification commands at the module level.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{}
}

// Compile-time check that the Bazel module declares its manifest.
var _ ecosystem.ManifestFileProvider = (*Module)(nil)

// ManifestFiles declares MODULE.bazel and its bzlmod lock file so
// Version-Sentinel coverage reports list Bazel dependencies as uncovered
// instead of omitting them.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return []ecosystem.ManifestFileInfo{{
		Path:           "MODULE.bazel",
		Ecosystem:      "bzlmod",
		LockFile:       "MODULE.bazel.lock",
		LockFilePolicy: ecosystem.LockFilePolicyRecommended,
	}}
}
