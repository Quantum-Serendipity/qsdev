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

func TestDevExToolsHaveGenerateFunc(t *testing.T) {
	reg := DefaultRegistry()

	for _, name := range []string{"changelog", "commitlint", "secretspec"} {
		tool, ok := reg.ByName(name)
		if !ok {
			t.Errorf("tool %q not found", name)
			continue
		}
		if tool.GenerateFunc == nil {
			t.Errorf("tool %q has nil GenerateFunc", name)
		}
	}
}

func TestDevExTool_Changelog_Generate(t *testing.T) {
	reg := DefaultRegistry()
	tool, ok := reg.ByName("changelog")
	if !ok {
		t.Fatal("changelog tool not found")
	}

	files, err := tool.GenerateFunc(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateFunc returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 generated file, got %d", len(files))
	}
	if files[0].Path != "cliff.toml" {
		t.Errorf("generated file path = %q, want cliff.toml", files[0].Path)
	}
	if len(files[0].Content) == 0 {
		t.Error("generated file has empty content")
	}
}

func TestDevExTool_Commitlint_Generate(t *testing.T) {
	reg := DefaultRegistry()
	tool, ok := reg.ByName("commitlint")
	if !ok {
		t.Fatal("commitlint tool not found")
	}

	files, err := tool.GenerateFunc(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateFunc returned error: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 generated file, got %d", len(files))
	}
	if files[0].Path != ".commitlintrc.yml" {
		t.Errorf("generated file path = %q, want .commitlintrc.yml", files[0].Path)
	}
	if len(files[0].Content) == 0 {
		t.Error("generated file has empty content")
	}
}

func TestDevExTool_Secretspec_GenerateNil(t *testing.T) {
	reg := DefaultRegistry()
	tool, ok := reg.ByName("secretspec")
	if !ok {
		t.Fatal("secretspec tool not found")
	}

	files, err := tool.GenerateFunc(types.WizardAnswers{})
	if err != nil {
		t.Fatalf("GenerateFunc returned error: %v", err)
	}
	if files != nil {
		t.Errorf("expected nil files for no-services answers, got %d files", len(files))
	}
}
