package swift_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/swift"
)

// TestDenyRules_CommandForms runs real command spellings through the
// project's deny matcher: bare `swift package update` bumps every dependency past Package.resolved; routine commands stay allowed.
func TestDenyRules_CommandForms(t *testing.T) {
	t.Parallel()

	rules := (&swift.Module{}).DenyRules(ecosystem.ModuleConfig{})
	for _, cmd := range []string{
		"swift package update",
		"swift package update Alamofire",
	} {
		if _, ok := denyutil.FirstMatch(rules, "Bash("+cmd+")"); !ok {
			t.Errorf("%q is not denied by %v", cmd, rules)
		}
	}
	for _, cmd := range []string{
		"swift package resolve",
		"swift build",
		"swift test",
	} {
		if rule, ok := denyutil.FirstMatch(rules, "Bash("+cmd+")"); ok {
			t.Errorf("%q is unexpectedly denied by %q", cmd, rule)
		}
	}
}
