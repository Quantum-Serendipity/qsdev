package toolreg

import (
	"testing"
)

// assertLifecycleOnly checks that each named tool has no Enable/DisableFunc.
// These tools have no effect on WizardAnswers beyond their EnabledTools
// entry, which the enable and disable commands record themselves; a
// closure that only repeats that bookkeeping invites a mistyped key.
func assertLifecycleOnly(t *testing.T, reg *Registry, names ...string) {
	t.Helper()
	for _, name := range names {
		tool, ok := reg.ByName(name)
		if !ok {
			t.Errorf("tool %q not found in registry", name)
			continue
		}
		if tool.EnableFunc != nil {
			t.Errorf("tool %q has an EnableFunc; it only needs lifecycle bookkeeping", name)
		}
		if tool.DisableFunc != nil {
			t.Errorf("tool %q has a DisableFunc; it only needs lifecycle bookkeeping", name)
		}
	}
}

func TestBuiltinBehaviorsTargetCatalogTools(t *testing.T) {
	t.Parallel()
	reg, err := BuildFromCatalogE()
	if err != nil {
		t.Fatalf("BuildFromCatalogE: %v", err)
	}
	for name := range builtinBehaviors() {
		if _, ok := reg.ByName(name); !ok {
			t.Errorf("builtin behavior attached to %q, which is not a catalog tool", name)
		}
	}
}
