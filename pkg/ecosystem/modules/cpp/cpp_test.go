package cpp_test

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/cpp"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*cpp.Module)(nil)
var _ ecosystem.PackageProvider = (*cpp.Module)(nil)

func TestModuleIdentity(t *testing.T) {
	ecosystem.AssertModuleIdentity(t, &cpp.Module{}, "cpp", "C/C++", 2)
}

func TestDetect_CMakeListsPresent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "CMakeLists.txt"), []byte("cmake_minimum_required(VERSION 3.20)\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &cpp.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when CMakeLists.txt is present")
	}
	if result.Confidence != ecosystem.ConfidenceCertain {
		t.Errorf("Confidence = %v, want ConfidenceCertain", result.Confidence)
	}
	if len(result.Evidence) < 1 {
		t.Fatal("expected at least one evidence entry")
	}
	found := false
	for _, e := range result.Evidence {
		if strings.Contains(e, "CMakeLists.txt") {
			found = true
		}
	}
	if !found {
		t.Error("evidence should mention CMakeLists.txt")
	}
	if result.SuggestedConfig.Extras["build_system"] != "cmake" {
		t.Errorf("build_system = %q, want %q", result.SuggestedConfig.Extras["build_system"], "cmake")
	}
}

// TestDetect_Makefile verifies a Makefile only indicates C/C++ when C/C++
// sources accompany it. A bare Makefile fronts Go/Python/Node/container
// projects too, and detecting it as C/C++ enabled clang, cppcheck and
// cmake build/test tasks for those projects.
func TestDetect_Makefile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		files     map[string]string
		wantFound bool
	}{
		{"Makefile alone (Go service)", map[string]string{"Makefile": "build:\n\tgo build ./...\n", "go.mod": "module x\n"}, false},
		{"Makefile with root C source", map[string]string{"Makefile": "all:\n\tcc main.c\n", "main.c": "int main(void){return 0;}\n"}, true},
		{"Makefile with src/ C++ source", map[string]string{"Makefile": "all:\n", "src/app.cpp": "int main(){}\n"}, true},
		{"Makefile with include/ header", map[string]string{"Makefile": "all:\n", "include/app.h": "#pragma once\n"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			for rel, content := range tt.files {
				full := filepath.Join(dir, filepath.FromSlash(rel))
				if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}

			result := (&cpp.Module{}).Detect(dir)
			if result.Detected != tt.wantFound {
				t.Fatalf("Detected = %v, want %v (evidence %v)", result.Detected, tt.wantFound, result.Evidence)
			}
			if !tt.wantFound {
				return
			}
			if result.Confidence != ecosystem.ConfidenceProbable {
				t.Errorf("Confidence = %v, want ConfidenceProbable", result.Confidence)
			}
			if got := result.SuggestedConfig.Extras["build_system"]; got != "make" {
				t.Errorf("build_system = %q, want make", got)
			}
		})
	}
}

// TestVerificationCommands_ConfigureOnFreshCheckout verifies the build
// commands configure the build directory first, so the generated
// qsdev-build task works on a clean clone (no "could not load cache").
func TestVerificationCommands_ConfigureOnFreshCheckout(t *testing.T) {
	t.Parallel()
	tests := []struct {
		buildSystem string
		wantBuild   string
	}{
		{"cmake", "cmake -B build && cmake --build build"},
		{"meson", "(test -d build || meson setup build) && meson compile -C build"},
		{"make", "make"},
	}
	for _, tt := range tests {
		t.Run(tt.buildSystem, func(t *testing.T) {
			t.Parallel()
			vc := (&cpp.Module{}).VerificationCommands(ecosystem.ModuleConfig{Extras: map[string]string{"build_system": tt.buildSystem}})
			if len(vc.Build) != 1 || vc.Build[0] != tt.wantBuild {
				t.Errorf("Build = %v, want [%s]", vc.Build, tt.wantBuild)
			}
		})
	}
}

func TestDetect_EmptyDir(t *testing.T) {
	dir := t.TempDir()

	m := &cpp.Module{}
	result := m.Detect(dir)

	if result.Detected {
		t.Error("expected Detected=false when no C/C++ indicators present")
	}
}

func TestDevenvNixFragment(t *testing.T) {
	m := &cpp.Module{}
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{
		Extras: map[string]string{"build_system": "cmake"},
	})
	if err != nil {
		t.Fatalf("DevenvNixFragment() returned error: %v", err)
	}

	if fragment == "" {
		t.Fatal("DevenvNixFragment() returned empty string")
	}

	if !strings.Contains(fragment, "enable = true") {
		t.Errorf("DevenvNixFragment() missing %q\ngot:\n%s", "enable = true", fragment)
	}
	if !strings.Contains(fragment, "languages.cplusplus") {
		t.Errorf("DevenvNixFragment() missing %q\ngot:\n%s", "languages.cplusplus", fragment)
	}
	// Fragment should not contain packages — those are in DevenvPackages.
	if strings.Contains(fragment, "packages") {
		t.Errorf("DevenvNixFragment() should not contain packages block\ngot:\n%s", fragment)
	}
}

// --- DevenvPackages tests ---

func TestDevenvPackages(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		cfg  ecosystem.ModuleConfig
		want []string
	}{
		// cppcheck backs the CI scan, so it is provisioned for every
		// configuration, not only at the hook tiers its hook runs at.
		{name: "no build system", want: []string{"cppcheck"}},
		{name: "cmake", cfg: ecosystem.ModuleConfig{Extras: map[string]string{"build_system": "cmake"}}, want: []string{"cppcheck", "cmake", "gnumake"}},
		{name: "meson", cfg: ecosystem.ModuleConfig{Extras: map[string]string{"build_system": "meson"}}, want: []string{"cppcheck", "meson", "ninja"}},
		{name: "make", cfg: ecosystem.ModuleConfig{Extras: map[string]string{"build_system": "make"}}, want: []string{"cppcheck", "gnumake"}},
		{name: "cmake with sccache", cfg: ecosystem.ModuleConfig{Extras: map[string]string{"build_system": "cmake", "build_cache": "sccache"}}, want: []string{"cppcheck", "cmake", "gnumake", "sccache"}},
		{name: "conan wizard answer", cfg: ecosystem.ModuleConfig{PackageManager: "conan"}, want: []string{"cppcheck", "conan"}},
		{name: "conan detected", cfg: ecosystem.ModuleConfig{Extras: map[string]string{"package_manager": "conan", "build_system": "cmake"}}, want: []string{"cppcheck", "cmake", "gnumake", "conan"}},
		{name: "vcpkg", cfg: ecosystem.ModuleConfig{PackageManager: "vcpkg"}, want: []string{"cppcheck"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := (&cpp.Module{}).DevenvPackages(tt.cfg); !slices.Equal(got, tt.want) {
				t.Errorf("DevenvPackages(%+v) = %q, want %q", tt.cfg, got, tt.want)
			}
		})
	}
}

func TestDenyRules(t *testing.T) {
	m := &cpp.Module{}
	rules := m.DenyRules(ecosystem.ModuleConfig{})

	if len(rules) != 2 {
		t.Fatalf("DenyRules() with no PM returned %d rules, want 2", len(rules))
	}

	// Default (no PM) should include both conan and vcpkg rules.
	expectedConan := "Bash(conan install * --update)"
	expectedVcpkg := "Bash(vcpkg install *)"
	if rules[0] != expectedConan {
		t.Errorf("rules[0] = %q, want %q", rules[0], expectedConan)
	}
	if rules[1] != expectedVcpkg {
		t.Errorf("rules[1] = %q, want %q", rules[1], expectedVcpkg)
	}
}

func TestDenyRules_ConanOnly(t *testing.T) {
	m := &cpp.Module{}
	rules := m.DenyRules(ecosystem.ModuleConfig{
		Extras: map[string]string{"package_manager": "conan"},
	})

	if len(rules) != 1 {
		t.Fatalf("DenyRules(conan) returned %d rules, want 1", len(rules))
	}
	if rules[0] != "Bash(conan install * --update)" {
		t.Errorf("rules[0] = %q, want %q", rules[0], "Bash(conan install * --update)")
	}
}

func TestPreCommitHooks(t *testing.T) {
	m := &cpp.Module{}
	hooks := m.PreCommitHooks(ecosystem.ModuleConfig{})

	if len(hooks) != 2 {
		t.Fatalf("PreCommitHooks() returned %d hooks, want 2", len(hooks))
	}
	if hooks[0].ID != "clang-format" {
		t.Errorf("hooks[0].ID = %q, want %q", hooks[0].ID, "clang-format")
	}
	if hooks[1].ID != "cppcheck" {
		t.Errorf("hooks[1].ID = %q, want %q", hooks[1].ID, "cppcheck")
	}
}

func TestRegistration(t *testing.T) {
	reg := ecosystem.DefaultRegistry()
	mod, ok := reg.ByName("cpp")
	if !ok {
		t.Fatal("expected module 'cpp' to be registered in DefaultRegistry")
	}
	if mod.Name() != "cpp" {
		t.Errorf("registered module Name() = %q, want %q", mod.Name(), "cpp")
	}
}

// TestCICommands_ConanDetectsProfile checks the Conan lockfile check first
// creates the default profile a clean CI runner lacks, so `conan lock create`
// does not fail on the missing profile (U10-05).
func TestCICommands_ConanDetectsProfile(t *testing.T) {
	t.Parallel()

	const want = "conan profile detect --exist-ok && conan lock create . --lockfile=conan.lock --lockfile-out=/dev/null"
	for _, cfg := range []ecosystem.ModuleConfig{
		{PackageManager: "conan"},
		{Extras: map[string]string{"package_manager": "conan"}},
	} {
		var found bool
		for _, c := range (&cpp.Module{}).CICommands(cfg) {
			if c.Name == "conan-lock-verify" {
				found = true
				if c.Command != want || c.Phase != ecosystem.CIPhaseInstall {
					t.Errorf("conan-lock-verify = %q (phase %v), want install step %q", c.Command, c.Phase, want)
				}
			}
		}
		if !found {
			t.Errorf("CICommands(%+v) has no conan-lock-verify step", cfg)
		}
	}
}
