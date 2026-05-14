package toolreg

import (
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

var gdevOpsSkillNames = []string{
	"gdev-init",
	"gdev-onboard",
	"gdev-setup",
	"gdev-enable",
	"gdev-disable",
	"gdev-update",
	"gdev-doctor",
	"gdev-status",
	"gdev-tools",
	"gdev-detect",
}

func TestGdevOpsToolsRegistered(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range gdevOpsSkillNames {
		tool, ok := reg.ByName(name)
		if !ok {
			t.Errorf("gdev-ops tool %q not found in DefaultRegistry", name)
			continue
		}
		if tool.DisplayName == "" {
			t.Errorf("gdev-ops tool %q has empty DisplayName", name)
		}
		if tool.Description == "" {
			t.Errorf("gdev-ops tool %q has empty Description", name)
		}
		if tool.Category != CategoryAIAgent {
			t.Errorf("gdev-ops tool %q has category %q, want %q", name, tool.Category, CategoryAIAgent)
		}
		if len(tool.OwnedFiles) == 0 {
			t.Errorf("gdev-ops tool %q has no OwnedFiles", name)
		}
	}
}

func TestGdevOpsToolsAlwaysOn(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range gdevOpsSkillNames {
		tool, ok := reg.ByName(name)
		if !ok {
			t.Errorf("gdev-ops tool %q not found in DefaultRegistry", name)
			continue
		}
		if tool.Default != AlwaysOn {
			t.Errorf("gdev-ops tool %q has Default %v, want AlwaysOn", name, tool.Default)
		}
	}
}

func TestGdevOpsToolEnableDisable(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range gdevOpsSkillNames {
		t.Run(name, func(t *testing.T) {
			tool, ok := reg.ByName(name)
			if !ok {
				t.Fatalf("gdev-ops tool %q not found in DefaultRegistry", name)
			}

			if tool.EnableFunc == nil {
				t.Fatalf("gdev-ops tool %q has nil EnableFunc", name)
			}
			if tool.DisableFunc == nil {
				t.Fatalf("gdev-ops tool %q has nil DisableFunc", name)
			}

			// Test enable.
			answers := &types.WizardAnswers{
				EnabledTools: make(map[string]bool),
			}
			tool.EnableFunc(answers)
			if !answers.EnabledTools[name] {
				t.Errorf("after EnableFunc, EnabledTools[%q] should be true", name)
			}

			// Test disable.
			tool.DisableFunc(answers)
			if answers.EnabledTools[name] {
				t.Errorf("after DisableFunc, EnabledTools[%q] should be false", name)
			}
		})
	}
}

func TestGdevOpsToolEnableFunc_NilMap(t *testing.T) {
	reg := DefaultRegistry()

	// Verify EnableFunc initializes the map when nil.
	tool, ok := reg.ByName("gdev-init")
	if !ok {
		t.Fatal("gdev-init not found in DefaultRegistry")
	}

	answers := &types.WizardAnswers{}
	tool.EnableFunc(answers)
	if answers.EnabledTools == nil {
		t.Fatal("EnableFunc should initialize EnabledTools map when nil")
	}
	if !answers.EnabledTools["gdev-init"] {
		t.Error("EnableFunc should set gdev-init to true")
	}
}

func TestGdevOpsToolDisableFunc_NilMap(t *testing.T) {
	reg := DefaultRegistry()

	// Verify DisableFunc initializes the map when nil.
	tool, ok := reg.ByName("gdev-init")
	if !ok {
		t.Fatal("gdev-init not found in DefaultRegistry")
	}

	answers := &types.WizardAnswers{}
	tool.DisableFunc(answers)
	if answers.EnabledTools == nil {
		t.Fatal("DisableFunc should initialize EnabledTools map when nil")
	}
	if answers.EnabledTools["gdev-init"] {
		t.Error("DisableFunc should set gdev-init to false")
	}
}

func TestGdevOpsToolOwnedFiles(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range gdevOpsSkillNames {
		tool, ok := reg.ByName(name)
		if !ok {
			t.Errorf("gdev-ops tool %q not found", name)
			continue
		}

		expectedPath := ".claude/skills/" + name + "/SKILL.md"
		found := false
		for _, f := range tool.OwnedFiles {
			if f.Path == expectedPath {
				found = true
				if f.Ownership != Exclusive {
					t.Errorf("tool %q SKILL.md should be Exclusive, got %v", name, f.Ownership)
				}
			}
		}
		if !found {
			t.Errorf("tool %q does not own %q", name, expectedPath)
		}
	}
}
