package modules

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestEveryDenyRuleValidates checks every deny rule each registered module
// contributes against denyutil.Validate, so no module ships a malformed rule
// or a Bash rule ending in the legacy ":*" marker, which Claude Code reads as
// a trailing " *".
func TestEveryDenyRuleValidates(t *testing.T) {
	t.Parallel()
	checked := 0
	for _, mod := range ecosystem.DefaultRegistry().All() {
		drp, ok := mod.(ecosystem.DenyRuleProvider)
		if !ok {
			continue
		}
		for _, rule := range drp.DenyRules(ecosystem.ModuleConfig{}) {
			checked++
			if err := denyutil.Validate(rule); err != nil {
				t.Errorf("module %s: %v", mod.Name(), err)
			}
		}
	}
	if checked == 0 {
		t.Fatal("no module contributed a deny rule; the registry walk is broken")
	}
}
