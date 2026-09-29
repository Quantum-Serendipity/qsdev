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
	denied := func(cmd string) bool {
		for _, r := range rules {
			if denyutil.MatchesDenyRule(r, "Bash("+cmd+")") {
				return true
			}
		}
		return false
	}

	for _, cmd := range []string{
		"swift package update",
		"swift package update Alamofire",
	} {
		if !denied(cmd) {
			t.Errorf("%q is not denied by %v", cmd, rules)
		}
	}
	for _, cmd := range []string{
		"swift package resolve",
		"swift build",
		"swift test",
	} {
		if denied(cmd) {
			t.Errorf("%q is unexpectedly denied by %v", cmd, rules)
		}
	}
}
