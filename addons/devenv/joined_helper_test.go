package devenv_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// markJoined records the local init state that `qsdev init` writes, so a
// fixture that commits a .qsdev.yaml by hand is a joined checkout rather
// than a fresh clone (which the mutating commands refuse).
func markJoined(t *testing.T, dir string) {
	t.Helper()
	if err := state.SaveProjectState(dir, state.InitStateFile(), types.GeneratedState{}); err != nil {
		t.Fatal(err)
	}
}
