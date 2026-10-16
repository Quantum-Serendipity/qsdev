package toolreg

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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
	cat, err := catalog.Default()
	if err != nil {
		t.Fatalf("catalog.Default: %v", err)
	}
	reg, err := BuildFromCatalogE()
	if err != nil {
		t.Fatalf("BuildFromCatalogE: %v", err)
	}
	for name := range builtinBehaviors(cat) {
		if _, ok := reg.ByName(name); !ok {
			t.Errorf("builtin behavior attached to %q, which is not a catalog tool", name)
		}
	}
}

// TestSembleEnableFunc_ModeDefaults checks that enabling semble resolves an
// empty mode to the catalog default (.qsdev.yaml does not record it) and
// keeps a mode the answers already carry.
func TestSembleEnableFunc_ModeDefaults(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatalf("catalog.Default: %v", err)
	}
	enable := builtinBehaviors(cat)[ToolSemble].EnableFunc
	tests := []struct {
		name, mode, want string
	}{
		{"empty resolves to the catalog default", "", cat.DefaultSembleMode()},
		{"explicit mode kept", "subagent", "subagent"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := types.WizardAnswers{AgentTools: types.AgentToolsAnswers{SembleMode: tt.mode}}
			enable(&a)
			if !a.AgentTools.SembleEnabled || a.AgentTools.SembleMode != tt.want {
				t.Errorf("after enable: enabled=%v mode=%q, want enabled with mode %q",
					a.AgentTools.SembleEnabled, a.AgentTools.SembleMode, tt.want)
			}
		})
	}
}
