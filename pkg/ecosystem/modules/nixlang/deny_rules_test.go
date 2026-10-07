package nixlang_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/nixlang"
)

// TestDenyRules_CommandForms runs real command spellings through the
// project's deny matcher: every nix-env install spelling is denied; routine commands stay allowed.
func TestDenyRules_CommandForms(t *testing.T) {
	t.Parallel()

	rules := (&nixlang.Module{}).DenyRules(ecosystem.ModuleConfig{})
	for _, cmd := range []string{
		"nix-env -i hello",
		"nix-env -iA nixpkgs.hello",
		"nix-env --install hello",
		"nix-env -f . -i hello",
	} {
		if _, ok := denyutil.FirstMatch(rules, "Bash("+cmd+")"); !ok {
			t.Errorf("%q is not denied by %v", cmd, rules)
		}
	}
	for _, cmd := range []string{
		"nix flake check",
		"nix build",
		"nixfmt flake.nix",
	} {
		if rule, ok := denyutil.FirstMatch(rules, "Bash("+cmd+")"); ok {
			t.Errorf("%q is unexpectedly denied by %q", cmd, rule)
		}
	}
}
