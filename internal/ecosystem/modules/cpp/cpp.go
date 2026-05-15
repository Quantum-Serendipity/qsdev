// Package cpp implements the C/C++ ecosystem module for qsdev.
// It detects C/C++ projects by scanning for build system files (CMake, Meson, Make)
// and package manager markers (Conan, vcpkg), then generates devenv.nix fragments,
// security configurations, pre-commit hooks, deny rules, and CI commands for a
// hardened C/C++ development environment.
package cpp

import (
	"fmt"
	"path/filepath"
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

	// Check Makefile (Probable, build_system="make" only if no certain build system found).
	if fileutil.FileExists(projectRoot, "Makefile") {
		result.Evidence = append(result.Evidence, "Makefile found")
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

// DevenvNixFragment returns the Nix code fragment to include in devenv.nix
// for C/C++ language support with build-system-appropriate packages.
func (m *Module) DevenvNixFragment(config ecosystem.ModuleConfig) (string, error) {
	var b strings.Builder
	b.WriteString("  languages.cplusplus.enable = true;\n")
	b.WriteString("\n")

	// Build system packages.
	buildSystem := config.Extras["build_system"]
	var pkgs []string
	switch buildSystem {
	case "cmake":
		pkgs = append(pkgs, "pkgs.cmake", "pkgs.gnumake")
	case "meson":
		pkgs = append(pkgs, "pkgs.meson", "pkgs.ninja")
	case "make":
		pkgs = append(pkgs, "pkgs.gnumake")
	}

	// Optional build cache.
	if config.Extras["build_cache"] == "sccache" {
		pkgs = append(pkgs, "pkgs.sccache")
	}

	if len(pkgs) > 0 {
		b.WriteString("  packages = [\n")
		for _, pkg := range pkgs {
			fmt.Fprintf(&b, "    %s\n", pkg)
		}
		b.WriteString("  ];\n")
	}

	return b.String(), nil
}

// DevenvYamlInputs returns additional flake inputs for devenv.yaml.
// C/C++ does not require any additional inputs.
func (m *Module) DevenvYamlInputs(_ ecosystem.ModuleConfig) []ecosystem.DevenvInput {
	return nil
}

// SecurityConfigs returns generated security configuration files for the
// detected C/C++ package manager.
func (m *Module) SecurityConfigs(config ecosystem.ModuleConfig) []types.GeneratedFile {
	pm := config.Extras["package_manager"]

	switch pm {
	case "conan":
		return []types.GeneratedFile{conanSecurityConfig()}
	case "vcpkg":
		return []types.GeneratedFile{vcpkgSecurityConfig()}
	default:
		return nil
	}
}

// conanSecurityConfig generates a Conan 2 security profile that enforces
// lockfile usage.
func conanSecurityConfig() types.GeneratedFile {
	var b strings.Builder
	b.WriteString("# Security-hardened Conan 2 profile\n")
	b.WriteString("# Generated by qsdev.\n")
	b.WriteString("# Requires: Conan >= 2.0 for lockfile_policy support.\n")
	b.WriteString("\n")
	b.WriteString("[conf]\n")
	b.WriteString("tools.graph:lockfile_policy=require\n")

	return types.GeneratedFile{
		Path:     ".conan2/profiles/security",
		Content:  []byte(b.String()),
		Mode:     0o644,
		Strategy: types.Overwrite,
	}
}

// vcpkgSecurityConfig generates a vcpkg-configuration.json with a baseline
// pinning template for supply chain security.
func vcpkgSecurityConfig() types.GeneratedFile {
	var b strings.Builder
	b.WriteString("{\n")
	b.WriteString("  \"$comment\": \"Security-hardened vcpkg configuration. Generated by qsdev. Pin the baseline to a specific vcpkg commit for reproducible builds.\",\n")
	b.WriteString("  \"default-registry\": {\n")
	b.WriteString("    \"kind\": \"git\",\n")
	b.WriteString("    \"repository\": \"https://github.com/microsoft/vcpkg\",\n")
	b.WriteString("    \"baseline\": \"REPLACE_WITH_VCPKG_COMMIT_SHA\"\n")
	b.WriteString("  }\n")
	b.WriteString("}\n")

	return types.GeneratedFile{
		Path:     "vcpkg-configuration.json",
		Content:  []byte(b.String()),
		Mode:     0o644,
		Strategy: types.Overwrite,
	}
}

// PreCommitHooks returns pre-commit hook definitions for the C/C++ ecosystem.
func (m *Module) PreCommitHooks(_ ecosystem.ModuleConfig) []ecosystem.HookConfig {
	return []ecosystem.HookConfig{
		{
			ID:            "clang-format",
			Name:          "clang-format",
			Description:   "Format C/C++ source code with clang-format",
			Entry:         "clang-format -i",
			Language:      "system",
			Types:         []string{"c", "c++"},
			Stages:        []string{"pre-commit"},
			Files:         `\.(c|cc|cpp|cxx|h|hh|hpp|hxx)$`,
			PassFilenames: true,
			BuiltIn:       true,
		},
		{
			ID:            "cppcheck",
			Name:          "cppcheck",
			Description:   "Static analysis of C/C++ code with cppcheck",
			Entry:         "cppcheck --error-exitcode=1",
			Language:      "system",
			Types:         []string{"c", "c++"},
			Stages:        []string{"pre-commit"},
			Files:         `\.(c|cc|cpp|cxx|h|hh|hpp|hxx)$`,
			PassFilenames: true,
			BuiltIn:       false,
		},
	}
}

// DenyRules returns Claude Code deny-rule patterns for the C/C++ ecosystem.
// Rules are conditional on the detected package manager; if none is detected,
// both conan and vcpkg rules are included.
func (m *Module) DenyRules(config ecosystem.ModuleConfig) []string {
	pm := config.Extras["package_manager"]

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

	buildSystem := config.Extras["build_system"]

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

	// Conan lock verify if conan is the package manager.
	pm := config.Extras["package_manager"]
	if pm == "conan" {
		cmds = append(cmds, ecosystem.CICommand{
			Name:        "conan-lock-verify",
			Command:     "conan lock create . --lockfile=conan.lock --lockfile-out=/dev/null",
			Description: "Verify Conan lockfile is up to date",
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
			Key:         "cpp_build_system",
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
			Key:         "cpp_package_manager",
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
		{
			Key:         "cpp_standard",
			Label:       "C++ standard",
			Description: "Select the C++ standard version",
			Type:        ecosystem.FieldTypeSelect,
			Options: []ecosystem.WizardOption{
				{Label: "C++17", Value: "17"},
				{Label: "C++20", Value: "20"},
				{Label: "C++23", Value: "23"},
			},
			Default: "17",
		},
	}
}

// VerificationCommands returns build and test commands for C/C++ projects
// using CMake as the default build system.
func (m *Module) VerificationCommands(_ ecosystem.ModuleConfig) ecosystem.VerificationCommands {
	return ecosystem.VerificationCommands{
		Build: []string{"cmake --build build"},
		Test:  []string{"ctest --test-dir build"},
	}
}

// ManifestFiles returns the CMakeLists.txt manifest file for C/C++ projects.
func (m *Module) ManifestFiles(_ ecosystem.ModuleConfig) []ecosystem.ManifestFileInfo {
	return []ecosystem.ManifestFileInfo{{Path: "CMakeLists.txt", Ecosystem: "cmake", LockFilePolicy: ecosystem.LockFilePolicyNone}}
}

