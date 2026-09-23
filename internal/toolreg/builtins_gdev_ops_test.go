package toolreg

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestNewOperationSkillTool(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name            string
		description     string
		wantDisplayName string
	}{
		{"qsdev-init", "Initialize qsdev project configuration", "qsdev init"},
		{"qsdev-add-dep", "Add a dependency or package safely", "qsdev add-dep"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tool := NewOperationSkillTool(tt.name, tt.description)

			if tool.Name != tt.name {
				t.Errorf("Name = %q, want %q", tool.Name, tt.name)
			}
			if tool.DisplayName != tt.wantDisplayName {
				t.Errorf("DisplayName = %q, want %q", tool.DisplayName, tt.wantDisplayName)
			}
			if tool.Description != tt.description {
				t.Errorf("Description = %q, want %q", tool.Description, tt.description)
			}
			if tool.Category != CategoryAIAgent {
				t.Errorf("Category = %q, want %q", tool.Category, CategoryAIAgent)
			}
			if tool.Default != AlwaysOn {
				t.Errorf("Default = %v, want AlwaysOn", tool.Default)
			}
			if tool.EnableFunc != nil || tool.DisableFunc != nil {
				t.Error("operation skill tools need only lifecycle bookkeeping; want nil Enable/DisableFunc")
			}

			wantPath := ".claude/skills/" + tt.name + "/SKILL.md"
			if len(tool.OwnedFiles) != 1 || tool.OwnedFiles[0].Path != wantPath || tool.OwnedFiles[0].Ownership != Exclusive {
				t.Errorf("OwnedFiles = %+v, want one exclusive %q", tool.OwnedFiles, wantPath)
			}
		})
	}
}

// TestOperationSkillToolIsEnabledByDefault proves an operation skill tool is
// switched on by MergeInferredTools, which gates whether its SKILL.md is
// deployed.
func TestOperationSkillToolIsEnabledByDefault(t *testing.T) {
	t.Parallel()
	reg := newTestRegistry(NewOperationSkillTool("qsdev-add-dep", "Add a dependency or package safely"))

	answers := types.WizardAnswers{EnabledTools: map[string]bool{"qsdev-init": true}}
	MergeInferredTools(&answers, reg)
	if !answers.EnabledTools["qsdev-add-dep"] {
		t.Errorf("EnabledTools[qsdev-add-dep] = false, want true (always-on)")
	}
}
