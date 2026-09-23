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
// It runs whenever nix is on PATH and can resolve the nixpkgs flake (about a
// second with a warm cache), so every `go test ./...` on a Nix machine checks
// it. QSDEV_CHECK_NIX_ATTRS=1 makes it mandatory (fail instead of skip when
// nix or nixpkgs is unavailable); QSDEV_CHECK_NIX_ATTRS=0 or -short skips it.
func TestCustomHookNixPackagesResolve(t *testing.T) {
	nixBin := requireNixpkgs(t)

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

// requireNixpkgs returns the nix binary when the nixpkgs flake is resolvable,
// honouring QSDEV_CHECK_NIX_ATTRS (see TestCustomHookNixPackagesResolve).
func requireNixpkgs(t *testing.T) string {
	t.Helper()
	mode := os.Getenv("QSDEV_CHECK_NIX_ATTRS")
	required := mode == "1"
	unavailable := func(format string, args ...any) {
		t.Helper()
		if required {
			t.Fatalf(format, args...)
		}
		t.Skipf(format, args...)
	}
	if !required && (mode == "0" || testing.Short()) {
		t.Skip("NixPackage attribute validation disabled (QSDEV_CHECK_NIX_ATTRS=0 or -short)")
	}
	nixBin, err := exec.LookPath("nix")
	if err != nil {
		unavailable("nix not available; cannot validate hook NixPackage attributes")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	if out, err := exec.CommandContext(ctx, nixBin, "eval", "--raw", "nixpkgs#lib.version").CombinedOutput(); err != nil {
		unavailable("nixpkgs flake not resolvable (offline?): %v\n%s", err, out)
	}
	return nixBin
}

// withExtra returns a ModuleConfig carrying a single Extra key/value.
func withExtra(key, value string) ecosystem.ModuleConfig {
	return ecosystem.ModuleConfig{Extras: map[string]string{key: value}}
}

// TestNixPkgsAttrOr_Evaluates evaluates ecosystem.NixPkgsAttrOr against real
// nixpkgs for a present attribute, a missing one and a removed-version throw
// alias (bazel_6, zig_0_12). A plain `pkgs.<attr> or <fallback>` does not
// catch the throw, which failed evaluation of every Bazel 6 devenv.
func TestNixPkgsAttrOr_Evaluates(t *testing.T) {
	nixBin := requireNixpkgs(t)
	tests := []struct {
		attr, fallback, want string
	}{
		{"hello", "null", "hello"},
		{"bazel_6", `{ pname = "fallback"; }`, "fallback"},
		{"zig_0_12", `{ pname = "fallback"; }`, "fallback"},
		{"zig_0_9", `{ pname = "fallback"; }`, "fallback"},
	}
	for _, tt := range tests {
		t.Run(tt.attr, func(t *testing.T) {
			t.Parallel()
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()
			expr := `let pkgs = import (builtins.getFlake "nixpkgs") { }; lib = pkgs.lib; in (` +
				ecosystem.NixPkgsAttrOr(tt.attr, tt.fallback, "test fallback") + `).pname`
			out, err := exec.CommandContext(ctx, nixBin, "eval", "--impure", "--raw", "--expr", expr).Output()
			if err != nil {
				t.Fatalf("evaluating %s: %v", expr, err)
			}
			if string(out) != tt.want {
				t.Errorf("pname = %q, want %q", out, tt.want)
			}
		})
	}
}
