// Package cpp implements the C/C++ ecosystem module for qsdev.
// It detects C/C++ projects by scanning for build system files (CMake, Meson, Make)
// and package manager markers (Conan, vcpkg), then generates devenv.nix fragments,
// security configurations, pre-commit hooks, deny rules, and CI commands for a
// hardened C/C++ development environment.
package cpp

import (
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface compliance checks.
var _ ecosystem.EcosystemModule = (*Module)(nil)
var _ ecosystem.PackageProvider = (*Module)(nil)
var _ ecosystem.SASTModule = (*Module)(nil)
var _ ecosystem.DenyRuleProvider = (*Module)(nil)
var _ ecosystem.WizardFieldProvider = (*Module)(nil)
var _ ecosystem.ManifestFileProvider = (*Module)(nil)

func init() {
	ecosystem.MustRegisterModule(&Module{})
}

// Module is the stateless C/C++ ecosystem module.
type Module struct{}

// Name returns the canonical module identifier.
func (m *Module) Name() string { return "cpp" }

// DisplayName returns the human-readable label.
func (m *Module) DisplayName() string { return "C/C++" }

// Tier returns the implementation priority tier (2 = standard).
func (m *Module) Tier() int { return 2 }

// Detect scans projectRoot for C/C++ ecosystem indicators including build system
// files (CMakeLists.txt, meson.build, Makefile) and package manager markers
// (conanfile.py, conanfile.txt, vcpkg.json, subprojects/*.wrap).
func (m *Module) Detect(projectRoot string) ecosystem.DetectionResult {
	result := ecosystem.DetectionResult{
		SuggestedConfig: ecosystem.ModuleConfig{
			Extras: make(map[string]string),
		},
	}

	certainBuildSystem := false

	// Check CMakeLists.txt (Certain, build_system="cmake").
	if fileutil.FileExists(projectRoot, "CMakeLists.txt") {
		result.Detected = true
		result.Confidence = ecosystem.ConfidenceCertain
		result.Evidence = append(result.Evidence, "CMakeLists.txt found")
		result.SuggestedConfig.Extras["build_system"] = "cmake"
		certainBuildSystem = true
	}

	// Check meson.build (Certain, build_system="meson").
	if fileutil.FileExists(projectRoot, "meson.build") {
		result.Detected = true
		result.Confidence = ecosystem.ConfidenceCertain
		result.Evidence = append(result.Evidence, "meson.build found")
		if !certainBuildSystem {
			result.SuggestedConfig.Extras["build_system"] = "meson"
		}
		certainBuildSystem = true
	}

	// Check conanfile.py / conanfile.txt (Certain, PM="conan").
	if fileutil.FileExists(projectRoot, "conanfile.py") || fileutil.FileExists(projectRoot, "conanfile.txt") {
		result.Detected = true
		result.Confidence = ecosystem.ConfidenceCertain
		result.Evidence = append(result.Evidence, "conanfile found")
		result.SuggestedConfig.Extras["package_manager"] = "conan"
		result.SuggestedConfig.PackageManager = "conan"
	}

	// Check vcpkg.json (Certain, PM="vcpkg").
	if fileutil.FileExists(projectRoot, "vcpkg.json") {
		result.Detected = true
		result.Confidence = ecosystem.ConfidenceCertain
		result.Evidence = append(result.Evidence, "vcpkg.json found")
		if result.SuggestedConfig.Extras["package_manager"] == "" {
			result.SuggestedConfig.Extras["package_manager"] = "vcpkg"
			result.SuggestedConfig.PackageManager = "vcpkg"
		}
	}

	// Check subprojects/*.wrap for meson-wrap PM.
	wrapFiles, _ := filepath.Glob(filepath.Join(projectRoot, "subprojects", "*.wrap"))
	if len(wrapFiles) > 0 {
		result.Detected = true
		result.Confidence = ecosystem.ConfidenceCertain
		result.Evidence = append(result.Evidence, "subprojects/*.wrap found (meson wrap)")
		if result.SuggestedConfig.Extras["package_manager"] == "" {
			result.SuggestedConfig.Extras["package_manager"] = "meson-wrap"
			result.SuggestedConfig.PackageManager = "meson-wrap"
		}
	}

	// Check Makefile (Probable, build_system="make" only if no certain build
	// system found). Makefiles front Go, Python, Node, docs and container
	// projects alike, so a Makefile only indicates C/C++ when C/C++ sources
	// sit next to it; otherwise it would enable a C/C++ toolchain, hooks and
	// build/test tasks for an unrelated project.
	if fileutil.FileExists(projectRoot, "Makefile") && hasCSources(projectRoot) {
		result.Evidence = append(result.Evidence, "Makefile with C/C++ sources found")
		if !result.Detected {
			result.Detected = true
			result.Confidence = ecosystem.ConfidenceProbable
		}
		if !certainBuildSystem {
			result.SuggestedConfig.Extras["build_system"] = "make"
		}
	}

	return result
}

// packageManager returns the configured C/C++ package manager: an explicit
// PackageManager (the wizard answer) wins over the one detection recorded in
// Extras["package_manager"].
func packageManager(config ecosystem.ModuleConfig) string {
	return config.PM(config.Extra(types.SettingPackageManager, ""))
}

// cSourceDirs are the conventional locations of C/C++ sources relative to
// the project root.
var cSourceDirs = []string{".", "src", "include", "lib"}

// cSourcePatterns match C and C++ source and header files.
var cSourcePatterns = []string{"*.c", "*.cc", "*.cpp", "*.cxx", "*.h", "*.hh", "*.hpp", "*.hxx"}

// hasCSources reports whether any conventional source directory under
// projectRoot contains C/C++ source or header files.
func hasCSources(projectRoot string) bool {
	for _, dir := range cSourceDirs {
		for _, pattern := range cSourcePatterns {
			if matches, _ := filepath.Glob(filepath.Join(projectRoot, dir, pattern)); len(matches) > 0 {
				return true
			}
		}
	}
	return false
}

// DevenvPackages returns Nix packages required by the C/C++ module based
// on the configured build system and optional build cache.
func (m *Module) DevenvPackages(config ecosystem.ModuleConfig) []string {
	var pkgs []string

	buildSystem := config.Extra("build_system", "")
	switch buildSystem {
	case "cmake":
		pkgs = append(pkgs, "cmake", "gnumake")
	case "meson":
		pkgs = append(pkgs, "meson", "ninja")
	case "make":
		pkgs = append(pkgs, "gnumake")
	}

	if config.Extra(ecosystem.ExtraBuildCache, "") == "sccache" {
		pkgs = append(pkgs, "sccache")
	}

	if len(pkgs) == 0 {
		return nil
	}
	return pkgs
}

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for C/C++ language support.
func (m *Module) DevenvNixFragment(_ ecosystem.ModuleConfig) (string, error) {
	return "  languages.cplusplus.enable = true;\n", nil
}

// SecurityConfigs returns nil. Neither C/C++ package manager has a
// project-level file qsdev can harden: Conan resolves profiles from
// CONAN_HOME (never the project) and has no conf that requires a lockfile,
// and a vcpkg registry baseline must be the project's real vcpkg commit, which
// a template cannot know (vcpkg rejects a placeholder). Lockfile use and
// baseline pinning are enforced by CICommands instead.
func (m *Module) SecurityConfigs(_ ecosystem.ModuleConfig) []types.GeneratedFile {
	return nil
}

// vcpkgBaselineCheck fails unless the project pins its vcpkg registry to a
// commit, either as vcpkg.json's builtin-baseline or as the default
// registry's baseline in vcpkg-configuration.json. Without one, versions float
// with whatever vcpkg checkout the machine has.
const vcpkgBaselineCheck = `jq -e '."builtin-baseline" | test("^[0-9a-f]{40}$")' vcpkg.json >/dev/null 2>&1 || ` +
	`jq -e '."default-registry".baseline | test("^[0-9a-f]{40}$")' vcpkg-configuration.json >/dev/null 2>&1 || ` +
	`{ echo 'vcpkg baseline is not pinned: set builtin-baseline in vcpkg.json (vcpkg x-update-baseline --add-initial-baseline)' >&2; false; }`

// cFamilyTypes are the identify tags of the files the C/C++ formatter owns.
var cFamilyTypes = []string{"c", "c++", "cuda", "objective-c"}

// PreCommitHooks returns pre-commit hook definitions for the C/C++ ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:          "clang-format",
			Name:        "clang-format",
			Description: "Format C/C++ source code with clang-format",
			Entry:       "clang-format -i",
			Language:    "system",
			// git-hooks.nix's built-in clang-format also runs on c#, java,
			// javascript, json and proto files, rewriting package.json,
			// qsdev's generated JSON and JS sources against the other
			// ecosystems' formatters. Scope it to the C-family files this
			// module owns.
			TypesOr:       cFamilyTypes,
			Stages:        []string{"pre-commit"},
			PassFilenames: true,
			BuiltIn:       true,
		},
		{
			ID:          "cppcheck",
			Name:        "cppcheck",
			Description: "Static analysis of C/C++ code with cppcheck",
			Entry:       "cppcheck --error-exitcode=1",
			Language:    "system",
			// types_or, not types: types is an AND filter and identify tags
			// only headers with both c and c++, so .c/.cpp files were skipped.
			TypesOr:       []string{"c", "c++"},
			Stages:        []string{"pre-commit"},
			Files:         `\.(c|cc|cpp|cxx|h|hh|hpp|hxx)$`,
			PassFilenames: true,
			// git-hooks.nix has no built-in "cppcheck" hook; render it as a
			// custom hook so an `entry` is always emitted and provision the
			// binary via NixPackage (otherwise the bare `cppcheck` command is
			// unresolved at commit time).
			BuiltIn:    false,
			NixPackage: "cppcheck",
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the C/C++ ecosystem.
// Rules are conditional on the detected package manager; if none is detected,
// both conan and vcpkg rules are included.
func (m *Module) DenyRules(config ecosystem.ModuleConfig) []string {
	pm := packageManager(config)

	switch pm {
	case "conan":
		return []string{
			"Bash(conan install * --update)",
		}
	case "vcpkg":
		return []string{
			"Bash(vcpkg install *)",
		}
	default:
		return []string{
			"Bash(conan install * --update)",
			"Bash(vcpkg install *)",
		}
	}
}

// CICommands returns CI pipeline commands for the C/C++ ecosystem.
// Build commands depend on the detected build system; cppcheck scan is always included.
func (m *Module) CICommands(config ecosystem.ModuleConfig) []ecosystem.CICommand {
	var cmds []ecosystem.CICommand

	buildSystem := config.Extra("build_system", "")

	switch buildSystem {
	case "cmake":
		cmds = append(cmds, ecosystem.CICommand{
			Name:        "cmake-build",
			Command:     "cmake -B build && cmake --build build",
			Description: "Configure and build with CMake",
			Phase:       ecosystem.CIPhaseTest,
		})
	case "meson":
		cmds = append(cmds, ecosystem.CICommand{
			Name:        "meson-build",
			Command:     "meson setup build && meson compile -C build",
			Description: "Configure and build with Meson",
			Phase:       ecosystem.CIPhaseTest,
		})
	case "make":
		cmds = append(cmds, ecosystem.CICommand{
			Name:        "make-build",
			Command:     "make",
			Description: "Build with Make",
			Phase:       ecosystem.CIPhaseTest,
		})
	}

	// Lockfile / baseline enforcement for the detected package manager.
	switch packageManager(config) {
	case "conan":
		// --lockfile is strict unless --lockfile-partial is given: any
		// requirement the lockfile does not pin fails the command.
		cmds = append(cmds, ecosystem.CICommand{
			Name:        "conan-lock-verify",
			Command:     "conan lock create . --lockfile=conan.lock --lockfile-out=/dev/null",
			Description: "Verify Conan lockfile is up to date",
			Phase:       ecosystem.CIPhaseInstall,
		})
	case "vcpkg":
		cmds = append(cmds, ecosystem.CICommand{
			Name:        "vcpkg-baseline-verify",
			Command:     vcpkgBaselineCheck,
			Description: "Verify the vcpkg registry baseline is pinned to a commit",
			Phase:       ecosystem.CIPhaseInstall,
		})
	}

	// cppcheck scan is always included.
	cmds = append(cmds, ecosystem.CICommand{
		Name:        "cppcheck-scan",
		Command:     "cppcheck --error-exitcode=1 --enable=warning,style,performance .",
		Description: "Static analysis scan with cppcheck",
		Phase:       ecosystem.CIPhaseScan,
	})

	return cmds
}

// PackageManagers returns metadata about C/C++ package managers.
func (m *Module) PackageManagers() []ecosystem.PackageManagerInfo {
	return []ecosystem.PackageManagerInfo{
		{
			Name:     "conan",
			LockFile: "conan.lock",
		},
		{
			Name:     "vcpkg",
			LockFile: "vcpkg.json",
		},
	}
}

// WizardFields returns wizard form fields for C/C++ configuration.
func (m *Module) WizardFields() []ecosystem.WizardField {
	return []ecosystem.WizardField{
		{
			Key:         "build_system",
			Label:       "Build system",
			Description: "Select the C/C++ build system for this project",
			Type:        ecosystem.FieldTypeSelect,
			Options: []ecosystem.WizardOption{
				{Label: "CMake", Value: "cmake"},
				{Label: "Meson", Value: "meson"},
				{Label: "Make", Value: "make"},
			},
			Default: "cmake",
		},
		{
			Key:         types.SettingPackageManager,
			Label:       "Package manager",
			Description: "Select the C/C++ package manager for this project",
			Type:        ecosystem.FieldTypeSelect,
			Options: []ecosystem.WizardOption{
				{Label: "Conan", Value: "conan"},
				{Label: "vcpkg", Value: "vcpkg"},
				{Label: "None", Value: "none"},
			},
			Default: "none",
		},
	}
}

// VerificationCommands returns build and test commands for C/C++ projects,
// switching on the configured build system (cmake, meson, or make).
func (m *Module) VerificationCommands(config ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	// Build commands include the configure step (matching CICommands) so
	// they work on a fresh checkout that has no build/ directory yet.
	switch config.Extra("build_system", "cmake") {
	case "cmake":
		return ecosystem.VerificationCommands{
			Build: []string{"cmake -B build && cmake --build build"},
			Test:  []string{"ctest --test-dir build"},
		}
	case "meson":
		return ecosystem.VerificationCommands{
			Build: []string{"(test -d build || meson setup build) && meson compile -C build"},
			Test:  []string{"meson test -C build"},
		}
	case "make":
		return ecosystem.VerificationCommands{
			Build: []string{"make"},
			Test:  []string{"make test"},
		}
	default:
		return ecosystem.VerificationCommands{}
	}
}

// ManifestFiles returns the CMakeLists.txt manifest file for C/C++ projects.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return []ecosystem.ManifestFileInfo{{Path: "CMakeLists.txt", Ecosystem: "cmake", LockFilePolicy: ecosystem.LockFilePolicyNone}}
}

// SemgrepRuleSets returns Semgrep rule set identifiers relevant to C/C++ projects.
func (m *Module) SemgrepRuleSets() []string {
	return []string{"p/cpp"}
}
