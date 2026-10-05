package haskell_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/shelltest"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// cabalInstall is the Cabal CI install command: it requires the committed
// freeze file, refreshes the package index the freeze file's index-state
// pins, then builds only the dependencies (the test phase builds the rest).
const cabalInstall = `test -f cabal.project.freeze || { echo "cabal.project.freeze missing: run cabal freeze" >&2; exit 1; }` +
	" && cabal update && cabal build --only-dependencies"

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
		{buildTool: "cabal", want: cabalInstall},
		{buildTool: "", want: cabalInstall},
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

// TestCICommands_CabalRequiresFreeze runs the Cabal install command with a
// stub cabal: a project without cabal.project.freeze fails before cabal runs,
// and otherwise `cabal update` then `cabal build --only-dependencies` run in
// order, the first failure stopping the step (U10-11).
func TestCICommands_CabalRequiresFreeze(t *testing.T) {
	t.Parallel()

	cmds := newModule().CICommands(ecosystem.ModuleConfig{})
	if len(cmds) != 1 || cmds[0].Phase != ecosystem.CIPhaseInstall {
		t.Fatalf("CICommands = %+v, want one install command", cmds)
	}

	freeze := map[string]string{"x.cabal": "", "cabal.project.freeze": "index-state: hackage.haskell.org 2026-01-01T00:00:00Z\n"}
	tests := []struct {
		name       string
		files      map[string]string
		stub       shelltest.Stub
		wantFail   bool
		wantOutput string
		wantCalls  []string
	}{
		{
			name:      "freeze file present",
			files:     freeze,
			wantCalls: []string{"cabal update", "cabal build --only-dependencies"},
		},
		{
			name:       "freeze file missing",
			files:      map[string]string{"x.cabal": ""},
			wantFail:   true,
			wantOutput: "cabal.project.freeze missing: run cabal freeze",
		},
		{
			name:      "update failure stops",
			files:     freeze,
			stub:      shelltest.Stub{Script: `[ "$1" = update ] && exit 1`},
			wantFail:  true,
			wantCalls: []string{"cabal update"},
		},
		{
			name:      "build failure fails",
			files:     freeze,
			stub:      shelltest.Stub{Script: `[ "$1" = build ] && exit 1`},
			wantFail:  true,
			wantCalls: []string{"cabal update", "cabal build --only-dependencies"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := shelltest.WriteTree(t, t.TempDir(), tt.files)
			res := shelltest.Run(t, dir, cmds[0].Command, map[string]shelltest.Stub{"cabal": tt.stub})
			if (res.Exit != 0) != tt.wantFail {
				t.Errorf("exit = %d, want failure %v; output:\n%s", res.Exit, tt.wantFail, res.Output)
			}
			if !strings.Contains(res.Output, tt.wantOutput) {
				t.Errorf("output = %q, want it to contain %q", res.Output, tt.wantOutput)
			}
			if strings.Join(res.Calls, "\n") != strings.Join(tt.wantCalls, "\n") {
				t.Errorf("calls = %q, want %q", res.Calls, tt.wantCalls)
			}
		})
	}
}
