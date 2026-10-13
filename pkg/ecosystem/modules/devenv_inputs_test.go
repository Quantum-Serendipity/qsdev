package modules_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// lockableInputRe matches a GitHub flake reference without a ref or query:
// devenv.lock pins its revision, and nothing in devenv.yaml overrides it.
var lockableInputRe = regexp.MustCompile(`^github:[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+$`)

// inputConfigs are module configurations that exercise both sides of each
// module's input decision: unset, and a version each version-pinning module
// accepts.
var inputConfigs = []ecosystem.ModuleConfig{
	{},
	{Version: "1.26.7"},
	{Version: "1.24"},
	{Version: "3.3.0"},
	{Version: "3.12"},
	{Version: "1.8.5"},
	{Version: "stable"},
	{Extras: map[string]string{"toolchain": "1.26.8"}},
	{Extras: map[string]string{"variant": "opentofu"}, Version: "1.8.5"},
}

// TestDevenvYamlInputs_ReferencedAndLocked is the contract between a module's
// devenv.yaml inputs and its devenv.nix fragment: every input the module
// contributes is read by an option its fragment sets (so devenv.yaml never
// declares a flake input the generated devenv.nix does not use), and every
// input is lockable in devenv.lock and follows the project's nixpkgs (so it
// never brings in a second, separately pinned nixpkgs).
func TestDevenvYamlInputs_ReferencedAndLocked(t *testing.T) {
	t.Parallel()
	providers := 0
	for _, mod := range ecosystem.DefaultRegistry().All() {
		yip, ok := mod.(ecosystem.DevenvYamlInputProvider)
		if !ok {
			continue
		}
		providers++
		for _, cfg := range inputConfigs {
			fragment, err := mod.DevenvNixFragment(cfg)
			if err != nil {
				continue // not a configuration this module generates
			}
			for _, in := range yip.DevenvYamlInputs(cfg) {
				if !lockableInputRe.MatchString(in.URL) {
					t.Errorf("%s %+v: input %q is not a github:owner/repo reference devenv.lock pins", mod.Name(), cfg, in.URL)
				}
				if in.Follows != "nixpkgs" {
					t.Errorf("%s %+v: input %q follows %q, want nixpkgs", mod.Name(), cfg, in.URL, in.Follows)
				}
				if len(in.Options) == 0 {
					t.Errorf("%s: input %q names no devenv option that reads it", mod.Name(), in.URL)
				} else if !setsAnyOption(fragment, in.Options) {
					t.Errorf("%s %+v: input %q is contributed but the fragment sets none of %v:\n%s",
						mod.Name(), cfg, in.URL, in.Options, fragment)
				}
			}
		}
	}
	if providers == 0 {
		t.Fatal("no registered module implements ecosystem.DevenvYamlInputProvider; the contract checks nothing")
	}
}

// setsAnyOption reports whether fragment assigns one of the devenv options
// ("languages.go.version"), written either flat or inside the language block
// BuildLanguageFragment renders ("languages.go = { ... version = ...; }").
func setsAnyOption(fragment string, options []string) bool {
	for _, opt := range options {
		i := strings.LastIndex(opt, ".")
		block, leaf := opt[:i], opt[i+1:]
		if strings.Contains(fragment, opt+" = ") ||
			strings.Contains(fragment, block+" = {") && strings.Contains(fragment, " "+leaf+" = ") {
			return true
		}
	}
	return false
}
