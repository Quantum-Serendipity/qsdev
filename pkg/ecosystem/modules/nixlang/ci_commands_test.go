package nixlang_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/nixlang"
)

// flakeConfig is the module config detection suggests for a flake project.
var flakeConfig = ecosystem.ModuleConfig{Extras: map[string]string{nixlang.ExtraFlake: "true"}}

// TestCICommands_LockCheckEnforcesNotUpdates guards the flake.lock CI step:
// it must fail when flake.lock does not lock what flake.nix declares, and
// must never update an input, which would fail on every upstream commit.
func TestCICommands_LockCheckEnforcesNotUpdates(t *testing.T) {
	t.Parallel()

	cmds := newModule().CICommands(flakeConfig)

	var lockCheck *ecosystem.CICommand
	for i := range cmds {
		if strings.Contains(cmds[i].Command, "--update-input") || strings.Contains(cmds[i].Command, "flake update") {
			t.Errorf("CI command %q updates a flake input", cmds[i].Command)
		}
		if cmds[i].Name == "nix-flake-lock-check" {
			lockCheck = &cmds[i]
		}
	}

	if lockCheck == nil {
		t.Fatal("no nix-flake-lock-check CI command")
	}
	if !strings.Contains(lockCheck.Command, "--no-update-lock-file") {
		t.Errorf("lock check %q must refuse to modify flake.lock", lockCheck.Command)
	}
	if lockCheck.Phase != ecosystem.CIPhaseInstall {
		t.Errorf("lock check phase = %v, want install", lockCheck.Phase)
	}
}

// TestCICommands_OnlyForFlakes checks the flake commands are emitted only for
// a flake project: `nix flake check` and `nix flake metadata` fail in a
// default.nix or shell.nix project (F436).
func TestCICommands_OnlyForFlakes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config ecosystem.ModuleConfig
		want   []string
	}{
		{name: "flake", config: flakeConfig, want: []string{"nix-flake-check", "nix-flake-lock-check"}},
		{name: "no flake recorded", config: ecosystem.ModuleConfig{}},
		{name: "flake false", config: ecosystem.ModuleConfig{Extras: map[string]string{nixlang.ExtraFlake: "false"}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got []string
			for _, c := range newModule().CICommands(tt.config) {
				got = append(got, c.Name)
			}
			if strings.Join(got, ",") != strings.Join(tt.want, ",") {
				t.Errorf("CI commands = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestDetect_RecordsFlake checks detection records a root flake.nix, which
// gates the flake CI commands.
func TestDetect_RecordsFlake(t *testing.T) {
	t.Parallel()

	tests := []struct {
		file string
		want string
	}{
		{file: "flake.nix", want: "true"},
		{file: "default.nix", want: ""},
		{file: "shell.nix", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if err := os.WriteFile(filepath.Join(dir, tt.file), []byte("{ }\n"), 0o600); err != nil {
				t.Fatal(err)
			}
			result := newModule().Detect(dir)
			if got := result.SuggestedConfig.Extra(nixlang.ExtraFlake, ""); got != tt.want {
				t.Errorf("%s: extra %s = %q, want %q", tt.file, nixlang.ExtraFlake, got, tt.want)
			}
		})
	}
}
