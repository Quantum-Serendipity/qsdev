package zig_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/zig"
)

// TestDenyRules_CommandForms runs real command spellings through the
// project's deny matcher: `zig fetch --save` pins whatever an arbitrary URL serves (trust on first use); routine commands stay allowed.
func TestDenyRules_CommandForms(t *testing.T) {
	t.Parallel()

	rules := (&zig.Module{}).DenyRules(ecosystem.ModuleConfig{})
	for _, cmd := range []string{
		"zig fetch --save https://example.com/dep.tar.gz",
		"zig fetch https://example.com/dep.tar.gz",
	} {
		if _, ok := denyutil.FirstMatch(rules, "Bash("+cmd+")"); !ok {
			t.Errorf("%q is not denied by %v", cmd, rules)
		}
	}
	for _, cmd := range []string{
		"zig build",
		"zig fmt --check .",
		"zig test src/main.zig",
	} {
		if rule, ok := denyutil.FirstMatch(rules, "Bash("+cmd+")"); ok {
			t.Errorf("%q is unexpectedly denied by %q", cmd, rule)
		}
	}
}
