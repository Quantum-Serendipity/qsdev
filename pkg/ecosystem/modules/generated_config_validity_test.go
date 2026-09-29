package modules

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestCpp_NoInvalidPackageManagerConfigs verifies the C/C++ module writes no
// Conan profile (Conan never loads project profiles, and
// tools.graph:lockfile_policy is not a Conan conf, so passing it errors) and
// no vcpkg-configuration.json with a placeholder baseline (vcpkg rejects a
// baseline that is not a commit SHA). Pinning is enforced in CI instead.
func TestCpp_NoInvalidPackageManagerConfigs(t *testing.T) {
	t.Parallel()
	mod, _ := ecosystem.DefaultRegistry().ByName("cpp")
	for _, pm := range []string{"", "conan", "vcpkg", "meson-wrap"} {
		cfg := ecosystem.ModuleConfig{Extras: map[string]string{"package_manager": pm}}
		for _, f := range mod.SecurityConfigs(cfg) {
			if strings.Contains(string(f.Content), "lockfile_policy") || strings.Contains(string(f.Content), "REPLACE_WITH") {
				t.Errorf("package manager %q: %s contains an invalid setting:\n%s", pm, f.Path, f.Content)
			}
		}
	}

	cmds := mod.CICommands(ecosystem.ModuleConfig{Extras: map[string]string{"package_manager": "vcpkg"}})
	if !slices.ContainsFunc(cmds, func(c ecosystem.CICommand) bool { return c.Name == "vcpkg-baseline-verify" }) {
		t.Errorf("vcpkg CI commands %v lack a baseline check", cmds)
	}
	cmds = mod.CICommands(ecosystem.ModuleConfig{Extras: map[string]string{"package_manager": "conan"}})
	if !slices.ContainsFunc(cmds, func(c ecosystem.CICommand) bool { return strings.Contains(c.Command, "--lockfile=conan.lock") }) {
		t.Errorf("conan CI commands %v do not enforce the lockfile", cmds)
	}
}

// TestRuby_GemrcIsLoaded verifies the project .gemrc is exported through
// GEMRC: RubyGems never reads a .gemrc from the working directory.
func TestRuby_GemrcIsLoaded(t *testing.T) {
	t.Parallel()
	mod, _ := ecosystem.DefaultRegistry().ByName("ruby")
	files := mod.SecurityConfigs(ecosystem.ModuleConfig{})
	if !slices.ContainsFunc(files, func(f types.GeneratedFile) bool { return f.Path == ".gemrc" }) {
		t.Fatalf("SecurityConfigs = %v, want .gemrc", files)
	}
	frag, err := mod.DevenvNixFragment(ecosystem.ModuleConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(frag, `env.GEMRC = "${config.devenv.root}/.gemrc";`) {
		t.Errorf("fragment does not export GEMRC:\n%s", frag)
	}
}
