package testutil

import (
	"os"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/procexec"
)

// Switch is an environment variable that turns "tool unavailable" from a
// skip into a failure. A CI job that provisions the tools a switch governs
// sets it to "1", so a missing or broken tool cannot silently drop coverage;
// everywhere else the tests skip, visibly, without it.
type Switch string

// The switches. This file is the only place their names are spelled.
const (
	// RequireNix governs the Nix evaluation checks of rendered output.
	RequireNix Switch = "QSDEV_REQUIRE_NIX"
	// RequireE3 governs the real-sandbox (bubblewrap, Landlock) tests.
	RequireE3 Switch = "QSDEV_REQUIRE_E3"
	// RequireRuleScanner governs the shipped rule-library fixture scans.
	RequireRuleScanner Switch = "QSDEV_REQUIRE_RULE_SCANNER"
	// CheckNixAttrs governs the nixpkgs attribute checks. Unlike the others
	// it is tri-state: "0" (or -short) disables those checks outright.
	CheckNixAttrs Switch = "QSDEV_CHECK_NIX_ATTRS"
	// RequireSecTools governs the tests that run the secret scanners.
	RequireSecTools Switch = "QSDEV_REQUIRE_SECTOOLS"
)

// switchTools is the one table of the tools each switch governs.
var switchTools = map[Switch][]string{
	RequireNix:         {"nix", "nix-instantiate"},
	RequireE3:          {"bwrap", "ll-restrict"},
	RequireRuleScanner: {"opengrep", "semgrep"},
	CheckNixAttrs:      {"nix"},
	RequireSecTools:    {"gitleaks", "ripsecrets"},
}

// toolAliases are the other names a skip message may give a governed tool:
// the project bwrap ships as, and the kernel layer ll-restrict applies.
var toolAliases = map[string][]string{
	"bwrap":       {"bubblewrap"},
	"ll-restrict": {"Landlock"},
}

// Switches returns every switch, in a fixed order.
func Switches() []Switch {
	return []Switch{RequireNix, RequireE3, RequireRuleScanner, CheckNixAttrs, RequireSecTools}
}

// Required reports whether the switch is on: its variable is exactly "1".
func (s Switch) Required() bool {
	return os.Getenv(string(s)) == "1"
}

// Tools returns the tools the switch governs, or nil for an unknown switch.
func (s Switch) Tools() []string {
	return switchTools[s]
}

// Unavailable skips the test with the formatted reason, or fails it, with
// " (<SWITCH>=1)" appended, when s is required. It is for conditions that are
// not a PATH lookup (a kernel probe, an unresolvable nixpkgs); RequireTool
// covers those that are.
func Unavailable(t testing.TB, s Switch, format string, args ...any) {
	t.Helper()
	if s.Required() {
		t.Fatalf(format+" (%s=1)", append(args, string(s))...)
	}
	t.Skipf(format, args...)
}

// RequireTool returns the path of tool on PATH. When it is absent the test
// skips, or fails when s is required.
func RequireTool(t testing.TB, tool string, s Switch) string {
	t.Helper()
	path, err := procexec.LookPath(tool)
	if err != nil {
		Unavailable(t, s, "%s not available: %v", tool, err)
	}
	return path
}

// UnsetEnv unsets each key for the rest of the test and restores its prior
// value (or absence) afterwards. Like t.Setenv, it cannot be used in parallel
// tests.
func UnsetEnv(t testing.TB, keys ...string) {
	t.Helper()
	for _, k := range keys {
		t.Setenv(k, "") // registers the restore of the current value
		if err := os.Unsetenv(k); err != nil {
			t.Fatalf("unsetting %s: %v", k, err)
		}
	}
}
