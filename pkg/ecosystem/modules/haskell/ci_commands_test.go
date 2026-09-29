package haskell_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestCICommands_BuildToolCommands checks each build tool's CI install
// command, in particular that Stack enforces stack.yaml.lock with its real
// lock-file flag (F436: `stack build --locked` is not a Stack flag).
func TestCICommands_BuildToolCommands(t *testing.T) {
	t.Parallel()

	tests := []struct {
		buildTool string
		want      string
	}{
		{buildTool: "stack", want: "stack build --lock-file=error-on-write"},
		{buildTool: "cabal", want: "cabal build"},
		{buildTool: "", want: "cabal build"},
	}
	for _, tt := range tests {
		t.Run("build_tool="+tt.buildTool, func(t *testing.T) {
			t.Parallel()
			config := ecosystem.ModuleConfig{Extras: map[string]string{"build_tool": tt.buildTool}}
			cmds := newModule().CICommands(config)
			if len(cmds) != 1 || cmds[0].Command != tt.want || cmds[0].Phase != ecosystem.CIPhaseInstall {
				t.Errorf("CICommands = %+v, want one install command %q", cmds, tt.want)
			}
		})
	}
}
