package perl_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/perl"
)

// TestDenyRules_CommandForms runs real command spellings through the
// project's deny matcher: every cpan/cpanm spelling can install an unvetted module; routine commands stay allowed.
func TestDenyRules_CommandForms(t *testing.T) {
	t.Parallel()

	rules := (&perl.Module{}).DenyRules(ecosystem.ModuleConfig{})
	for _, cmd := range []string{
		"cpan Foo::Bar",
		"cpan -i Foo::Bar",
		"cpan install Foo::Bar",
		"cpanm Foo::Bar",
	} {
		if _, ok := denyutil.FirstMatch(rules, "Bash("+cmd+")"); !ok {
			t.Errorf("%q is not denied by %v", cmd, rules)
		}
	}
	for _, cmd := range []string{
		"carton install --deployment",
		"perl -c lib/Foo.pm",
		"prove -lr t",
	} {
		if rule, ok := denyutil.FirstMatch(rules, "Bash("+cmd+")"); ok {
			t.Errorf("%q is unexpectedly denied by %q", cmd, rule)
		}
	}
}
