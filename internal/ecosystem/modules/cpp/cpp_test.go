package cpp_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem/modules/cpp"
)

// Compile-time interface compliance check.
var _ ecosystem.EcosystemModule = (*cpp.Module)(nil)

func TestName(t *testing.T) {
	m := &cpp.Module{}
	if got := m.Name(); got != "cpp" {
		t.Errorf("Name() = %q, want %q", got, "cpp")
	}
}

func TestDisplayName(t *testing.T) {
	m := &cpp.Module{}
	got := m.DisplayName()
	if !strings.Contains(got, "C/C++") {
		t.Errorf("DisplayName() = %q, want it to contain %q", got, "C/C++")
	}
}

func TestTier(t *testing.T) {
	m := &cpp.Module{}
	if got := m.Tier(); got != 2 {
		t.Errorf("Tier() = %d, want %d", got, 2)
	}
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

func TestDetect_MakefileProbable(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Makefile"), []byte("all:\n\techo hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	m := &cpp.Module{}
	result := m.Detect(dir)

	if !result.Detected {
		t.Fatal("expected Detected=true when Makefile is present")
	}
	if result.Confidence != ecosystem.ConfidenceProbable {
		t.Errorf("Confidence = %v, want ConfidenceProbable", result.Confidence)
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
}

func TestDevenvNixFragment_CMakePackages(t *testing.T) {
	m := &cpp.Module{}
	fragment, err := m.DevenvNixFragment(ecosystem.ModuleConfig{
		Extras: map[string]string{"build_system": "cmake"},
	})
	if err != nil {
		t.Fatalf("DevenvNixFragment() returned error: %v", err)
	}
	if !strings.Contains(fragment, "pkgs.cmake") {
		t.Errorf("DevenvNixFragment(cmake) should contain pkgs.cmake\ngot:\n%s", fragment)
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
