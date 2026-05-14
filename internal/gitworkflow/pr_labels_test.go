package gitworkflow

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/cigeneration"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestGenerateLabelerConfig_ReturnsTwoFiles(t *testing.T) {
	answers := types.WizardAnswers{}
	files, err := GenerateLabelerConfig(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	if files[0].Path != ".github/labeler.yml" {
		t.Errorf("files[0].Path = %q, want .github/labeler.yml", files[0].Path)
	}
	if files[1].Path != ".github/workflows/labeler.yml" {
		t.Errorf("files[1].Path = %q, want .github/workflows/labeler.yml", files[1].Path)
	}

	for i, f := range files {
		if f.Mode != 0o644 {
			t.Errorf("files[%d].Mode = %o, want 644", i, f.Mode)
		}
		if f.Strategy != types.Overwrite {
			t.Errorf("files[%d].Strategy = %v, want Overwrite", i, f.Strategy)
		}
	}
}

func TestGenerateLabelerConfig_StandardLabels(t *testing.T) {
	answers := types.WizardAnswers{}
	files, err := GenerateLabelerConfig(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	labelerContent := string(files[0].Content)

	for _, label := range []string{
		"documentation:",
		"infrastructure:",
		"security:",
		"dependencies:",
	} {
		if !strings.Contains(labelerContent, label) {
			t.Errorf("labeler.yml missing standard label %q", label)
		}
	}
}

func TestGenerateLabelerConfig_EcosystemLabels(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go"},
			{Name: "python"},
			{Name: "rust"},
		},
	}

	files, err := GenerateLabelerConfig(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	labelerContent := string(files[0].Content)

	expectations := map[string]string{
		"go":     "'**/*.go'",
		"python": "'**/*.py'",
		"rust":   "'**/*.rs'",
	}

	for lang, glob := range expectations {
		if !strings.Contains(labelerContent, lang+":") {
			t.Errorf("labeler.yml missing ecosystem label %q", lang)
		}
		if !strings.Contains(labelerContent, glob) {
			t.Errorf("labeler.yml missing glob %q for %s", glob, lang)
		}
	}
}

func TestGenerateLabelerConfig_UnknownEcosystemSkipped(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "haskell"},
		},
	}

	files, err := GenerateLabelerConfig(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	labelerContent := string(files[0].Content)
	if strings.Contains(labelerContent, "haskell:") {
		t.Error("labeler.yml should not contain label for unknown ecosystem 'haskell'")
	}
}

func TestGenerateLabelerConfig_WorkflowSHAPinned(t *testing.T) {
	answers := types.WizardAnswers{}
	files, err := GenerateLabelerConfig(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	workflowContent := string(files[1].Content)

	// Must use SHA-pinned action reference.
	if !strings.Contains(workflowContent, cigeneration.ActionLabeler.SHA) {
		t.Error("workflow should use SHA-pinned action reference")
	}
	if !strings.Contains(workflowContent, cigeneration.ActionLabeler.Tag) {
		t.Error("workflow should include tag as comment")
	}
}

func TestGenerateLabelerConfig_WorkflowPermissions(t *testing.T) {
	answers := types.WizardAnswers{}
	files, err := GenerateLabelerConfig(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	workflowContent := string(files[1].Content)

	if !strings.Contains(workflowContent, "contents: read") {
		t.Error("workflow missing contents: read permission")
	}
	if !strings.Contains(workflowContent, "pull-requests: write") {
		t.Error("workflow missing pull-requests: write permission")
	}
}

func TestGenerateLabelerConfig_JavascriptGlob(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "javascript"},
		},
	}

	files, err := GenerateLabelerConfig(answers)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	labelerContent := string(files[0].Content)
	if !strings.Contains(labelerContent, "'**/*.{js,jsx,ts,tsx}'") {
		t.Error("labeler.yml missing JavaScript/TypeScript glob pattern")
	}
}
