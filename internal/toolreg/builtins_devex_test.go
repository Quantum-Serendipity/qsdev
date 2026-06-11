package toolreg

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestDevExToolsRegistered(t *testing.T) {
	reg := DefaultRegistry()

	tools := []struct {
		name     string
		category ToolCategory
	}{
		{"changelog", CategoryDevEx},
		{"commitlint", CategoryDevEx},
		{"secretspec", CategoryDevEx},
	}

	for _, tt := range tools {
		tool, ok := reg.ByName(tt.name)
		if !ok {
			t.Errorf("devex tool %q not found in registry", tt.name)
			continue
		}
		if tool.Category != tt.category {
			t.Errorf("tool %q has category %v, want %v", tt.name, tool.Category, tt.category)
		}
		if tool.DisplayName == "" {
			t.Errorf("tool %q has empty DisplayName", tt.name)
		}
		if tool.Description == "" {
			t.Errorf("tool %q has empty Description", tt.name)
		}
		if tool.Default != OptIn {
			t.Errorf("tool %q has default %v, want OptIn", tt.name, tool.Default)
		}
	}
}

func TestDevExToolEnableDisable(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range []string{"changelog", "commitlint", "secretspec"} {
		tool, ok := reg.ByName(name)
		if !ok {
			t.Errorf("tool %q not found", name)
			continue
		}

		t.Run(name, func(t *testing.T) {
			answers := types.WizardAnswers{}

			if tool.EnableFunc == nil {
				t.Fatal("EnableFunc is nil")
			}
			tool.EnableFunc(&answers)

			if answers.EnabledTools == nil {
				t.Fatal("EnabledTools should be initialized after Enable")
			}
			if !answers.EnabledTools[name] {
				t.Errorf("after Enable, EnabledTools[%q] should be true", name)
			}

			if tool.DisableFunc == nil {
				t.Fatal("DisableFunc is nil")
			}
			tool.DisableFunc(&answers)

			if answers.EnabledTools[name] {
				t.Errorf("after Disable, EnabledTools[%q] should be false", name)
			}
		})
	}
}
