package lua_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/lua"
)

// TestDenyRules_CommandForms runs real command spellings through the
// project's deny matcher: flags before the subcommand must not bypass the rule; routine commands stay allowed.
func TestDenyRules_CommandForms(t *testing.T) {
	t.Parallel()

	rules := (&lua.Module{}).DenyRules(ecosystem.ModuleConfig{})
	for _, cmd := range []string{
		"luarocks install foo",
		"luarocks --local install foo",
		"luarocks install",
	} {
		if _, ok := denyutil.FirstMatch(rules, "Bash("+cmd+")"); !ok {
			t.Errorf("%q is not denied by %v", cmd, rules)
		}
	}
	for _, cmd := range []string{
		"luarocks list",
		"busted",
		"luacheck .",
	} {
		if rule, ok := denyutil.FirstMatch(rules, "Bash("+cmd+")"); ok {
			t.Errorf("%q is unexpectedly denied by %q", cmd, rule)
		}
	}
}
