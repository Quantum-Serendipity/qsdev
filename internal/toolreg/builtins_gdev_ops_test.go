package toolreg

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

var qsdevOpsSkillNames = []string{
	"qsdev-init",
	"qsdev-onboard",
	"qsdev-setup",
	"qsdev-enable",
	"qsdev-disable",
	"qsdev-update",
	"qsdev-doctor",
	"qsdev-status",
	"qsdev-tools",
	"qsdev-detect",
}

func TestQsdevOpsToolsRegistered(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range qsdevOpsSkillNames {
		tool, ok := reg.ByName(name)
		if !ok {
			t.Errorf("qsdev-ops tool %q not found in DefaultRegistry", name)
			continue
		}
		if tool.DisplayName == "" {
			t.Errorf("qsdev-ops tool %q has empty DisplayName", name)
		}
		if tool.Description == "" {
			t.Errorf("qsdev-ops tool %q has empty Description", name)
		}
		if tool.Category != CategoryAIAgent {
			t.Errorf("qsdev-ops tool %q has category %q, want %q", name, tool.Category, CategoryAIAgent)
		}
		if len(tool.OwnedFiles) == 0 {
			t.Errorf("qsdev-ops tool %q has no OwnedFiles", name)
		}
	}
}

func TestQsdevOpsToolsAlwaysOn(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range qsdevOpsSkillNames {
		tool, ok := reg.ByName(name)
		if !ok {
			t.Errorf("qsdev-ops tool %q not found in DefaultRegistry", name)
			continue
		}
		if tool.Default != AlwaysOn {
			t.Errorf("qsdev-ops tool %q has Default %v, want AlwaysOn", name, tool.Default)
		}
	}
}

func TestQsdevOpsToolEnableDisable(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range qsdevOpsSkillNames {
		t.Run(name, func(t *testing.T) {
			tool, ok := reg.ByName(name)
			if !ok {
				t.Fatalf("qsdev-ops tool %q not found in DefaultRegistry", name)
			}

			if tool.EnableFunc == nil {
				t.Fatalf("qsdev-ops tool %q has nil EnableFunc", name)
			}
			if tool.DisableFunc == nil {
				t.Fatalf("qsdev-ops tool %q has nil DisableFunc", name)
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

func TestQsdevOpsToolEnableFunc_NilMap(t *testing.T) {
	reg := DefaultRegistry()

	// Verify EnableFunc initializes the map when nil.
	tool, ok := reg.ByName("qsdev-init")
	if !ok {
		t.Fatal("qsdev-init not found in DefaultRegistry")
	}

	answers := &types.WizardAnswers{}
	tool.EnableFunc(answers)
	if answers.EnabledTools == nil {
		t.Fatal("EnableFunc should initialize EnabledTools map when nil")
	}
	if !answers.EnabledTools["qsdev-init"] {
		t.Error("EnableFunc should set qsdev-init to true")
	}
}

func TestQsdevOpsToolDisableFunc_NilMap(t *testing.T) {
	reg := DefaultRegistry()

	// Verify DisableFunc initializes the map when nil.
	tool, ok := reg.ByName("qsdev-init")
	if !ok {
		t.Fatal("qsdev-init not found in DefaultRegistry")
	}

	answers := &types.WizardAnswers{}
	tool.DisableFunc(answers)
	if answers.EnabledTools == nil {
		t.Fatal("DisableFunc should initialize EnabledTools map when nil")
	}
	if answers.EnabledTools["qsdev-init"] {
		t.Error("DisableFunc should set qsdev-init to false")
	}
}

func TestQsdevOpsToolOwnedFiles(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range qsdevOpsSkillNames {
		tool, ok := reg.ByName(name)
		if !ok {
			t.Errorf("qsdev-ops tool %q not found", name)
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
