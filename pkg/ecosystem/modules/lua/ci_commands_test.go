package lua_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/shelltest"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/lua"
)

// TestCICommands_GatedOnLuaRocks checks only LuaRocks projects get the
// LuaRocks dependency install (F436).
func TestCICommands_GatedOnLuaRocks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		pm   string
		want int
	}{
		{pm: "luarocks", want: 1},
		{pm: "lux", want: 0},
		{pm: "", want: 0},
	}
	for _, tt := range tests {
		t.Run("pm="+tt.pm, func(t *testing.T) {
			t.Parallel()
			cmds := (&lua.Module{}).CICommands(ecosystem.ModuleConfig{PackageManager: tt.pm})
			if len(cmds) != tt.want {
				t.Errorf("CICommands(%q) = %+v, want %d commands", tt.pm, cmds, tt.want)
			}
		})
	}
}

// TestCICommands_InstallsEveryRockspec runs the install command with a stub
// luarocks: each root rockspec is passed as the operand `--only-deps` needs,
// and a failing install or a missing rockspec fails the step (F436).
func TestCICommands_InstallsEveryRockspec(t *testing.T) {
	t.Parallel()

	cmds := (&lua.Module{}).CICommands(ecosystem.ModuleConfig{PackageManager: "luarocks"})
	if len(cmds) != 1 || cmds[0].Phase != ecosystem.CIPhaseInstall {
		t.Fatalf("CICommands = %+v, want one install command", cmds)
	}

	tests := []struct {
		name      string
		files     map[string]string
		exit      int
		wantFail  bool
		wantCalls []string
	}{
		{
			name:  "each rockspec",
			files: map[string]string{"app-1.0-1.rockspec": "", "app-scm-1.rockspec": ""},
			wantCalls: []string{
				"luarocks install --local --only-deps ./app-1.0-1.rockspec",
				"luarocks install --local --only-deps ./app-scm-1.rockspec",
			},
		},
		{
			name:      "install failure",
			files:     map[string]string{"app-1.0-1.rockspec": ""},
			exit:      1,
			wantFail:  true,
			wantCalls: []string{"luarocks install --local --only-deps ./app-1.0-1.rockspec"},
		},
		{
			name:     "no rockspec",
			files:    map[string]string{"main.lua": ""},
			wantFail: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := shelltest.WriteTree(t, t.TempDir(), tt.files)
			res := shelltest.Run(t, dir, cmds[0].Command, map[string]shelltest.Stub{"luarocks": {Exit: tt.exit}})
			if (res.Exit != 0) != tt.wantFail {
				t.Errorf("exit = %d, want failure %v; output:\n%s", res.Exit, tt.wantFail, res.Output)
			}
			if strings.Join(res.Calls, "\n") != strings.Join(tt.wantCalls, "\n") {
				t.Errorf("calls = %q, want %q", res.Calls, tt.wantCalls)
			}
		})
	}
}
