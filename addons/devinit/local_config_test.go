package devinit

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestGenerateLocalConfigTemplate_GoProject(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go", Version: "1.24"},
		},
		ClaudeCode: true,
	}
	detected := types.DetectedProject{}

	content := GenerateLocalConfigTemplate(answers, detected)
	s := string(content)

	if !strings.Contains(s, "go") {
		t.Error("Go project template should mention go")
	}
	if !strings.Contains(s, "1.24") {
		t.Error("Go project template should include go version")
	}
}

func TestGenerateLocalConfigTemplate_MultiLanguage(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go", Version: "1.24"},
			{Name: "javascript", Version: "22"},
		},
		ClaudeCode: true,
	}
	detected := types.DetectedProject{}

	content := GenerateLocalConfigTemplate(answers, detected)
	s := string(content)

	if !strings.Contains(s, "go") {
		t.Error("Multi-lang template should mention go")
	}
	if !strings.Contains(s, "javascript") {
		t.Error("Multi-lang template should mention javascript")
	}
}

func TestGenerateLocalConfigTemplate_ClaudeEnabled(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go"},
		},
		ClaudeCode: true,
	}
	detected := types.DetectedProject{}

	content := GenerateLocalConfigTemplate(answers, detected)
	s := string(content)

	if !strings.Contains(s, "claude_code:") {
		t.Error("Claude-enabled template should contain claude_code section")
	}
	if !strings.Contains(s, "permission_level") {
		t.Error("Claude-enabled template should contain permission_level")
	}
}

func TestGenerateLocalConfigTemplate_ClaudeDisabled(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go"},
		},
		ClaudeCode: false,
	}
	detected := types.DetectedProject{}

	content := GenerateLocalConfigTemplate(answers, detected)
	s := string(content)

	if strings.Contains(s, "claude_code:") {
		t.Error("Claude-disabled template should not contain claude_code section")
	}
}

func TestGenerateLocalConfigTemplate_AllLinesCommented(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go", Version: "1.24"},
			{Name: "python", Version: "3.12"},
		},
		ClaudeCode: true,
	}
	detected := types.DetectedProject{}

	content := GenerateLocalConfigTemplate(answers, detected)
	lines := strings.Split(string(content), "\n")

	for i, line := range lines {
		if line == "" {
			continue // empty lines are fine
		}
		if !strings.HasPrefix(line, "#") {
			t.Errorf("line %d is not commented: %q", i+1, line)
		}
	}
}

func TestGenerateLocalConfigTemplate_DefaultVersions(t *testing.T) {
	// Test that languages without explicit versions get default examples.
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "python"}, // no version set
		},
		ClaudeCode: false,
	}
	detected := types.DetectedProject{}

	content := GenerateLocalConfigTemplate(answers, detected)
	s := string(content)

	if !strings.Contains(s, "python") {
		t.Error("template should mention python")
	}
	if !strings.Contains(s, "3.12") {
		t.Error("template should include default python version example")
	}
}
