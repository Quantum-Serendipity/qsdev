package devenv_test

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"

	// Register every ecosystem module with the DefaultRegistry via init().
	_ "github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules"
)

// TestCustomHookNixPackagesResolve validates that every NixPackage attribute a
// custom pre-commit hook declares actually resolves in nixpkgs. It is the guard
// the vacuous non-empty check in TestCustomHooksProvisionTheirBinary could not
// provide: `spotbugs` (no such attribute) and `phpPackages.phpstan` (a removed
// throw-alias) both shipped self-certified because a non-empty string was
// treated as valid, breaking devenv.nix evaluation for Java/PHP at commit time.
//
// `nix eval` against nixpkgs is slow and needs network, so this runs only when
// QSDEV_CHECK_NIX_ATTRS=1 (a dedicated CI job) and nix is on PATH.
func TestCustomHookNixPackagesResolve(t *testing.T) {
	if os.Getenv("QSDEV_CHECK_NIX_ATTRS") != "1" {
		t.Skip("set QSDEV_CHECK_NIX_ATTRS=1 to validate hook NixPackage attributes against nixpkgs")
	}
	nixBin, err := exec.LookPath("nix")
	if err != nil {
		t.Skip("nix not available; skipping NixPackage attribute validation")
	}

	// Collect the unique declared attributes across all modules, including
	// variants toggled by common Extra flags (e.g. Kotlin's ktlint), so
	// conditional hooks are covered too.
	configs := []ecosystem.ModuleConfig{{}, withExtra("kotlin", "true")}
	attrs := map[string]string{} // attr -> "module/hookID" for error context
	for _, mod := range ecosystem.DefaultRegistry().All() {
		for _, cfg := range configs {
			for _, hook := range mod.PreCommitHooks(cfg) {
				if hook.NixPackage == "" {
					continue
				}
				attrs[hook.NixPackage] = mod.Name() + "/" + hook.ID
			}
		}
	}

	for attr, src := range attrs {
		t.Run(attr, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			cmd := exec.CommandContext(ctx, nixBin, "eval", "--raw", "nixpkgs#"+attr+".name")
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Errorf("hook %s declares NixPackage %q which does not resolve in nixpkgs: %v\n%s",
					src, attr, err, out)
			}
		})
	}
}

// withExtra returns a ModuleConfig carrying a single Extra key/value.
func withExtra(key, value string) ecosystem.ModuleConfig {
	return ecosystem.ModuleConfig{Extras: map[string]string{key: value}}
}
