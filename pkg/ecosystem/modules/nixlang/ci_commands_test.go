package nixlang_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestCICommands_LockCheckEnforcesNotUpdates guards the flake.lock CI step:
// it must fail when flake.lock does not lock what flake.nix declares, and
// must never update an input, which would fail on every upstream commit.
func TestCICommands_LockCheckEnforcesNotUpdates(t *testing.T) {
	t.Parallel()

	cmds := newModule().CICommands(ecosystem.ModuleConfig{})

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
