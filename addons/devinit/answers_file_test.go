package devinit_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/devinit"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestAnswersFile_FromReader(t *testing.T) {
	yaml := `
languages:
  - name: go
    version: "1.24"
claude_code: true
permission_level: standard
direnv: true
`
	r := strings.NewReader(yaml)
	answers, err := devinit.ExportLoadAnswersFromReader(r, "test")
	if err != nil {
		t.Fatalf("LoadAnswersFromReader: %v", err)
	}
	if len(answers.Languages) != 1 || answers.Languages[0].Name != "go" {
		t.Errorf("languages = %+v, want [{go 1.24}]", answers.Languages)
	}
	if !answers.ClaudeCode {
		t.Error("expected claude_code = true")
	}
	if answers.PermissionLevel != "standard" {
		t.Errorf("permission_level = %q, want %q", answers.PermissionLevel, "standard")
	}
}

func TestAnswersFile_LoadFromFile(t *testing.T) {
	dir := t.TempDir()
	content := []byte(`
languages:
  - name: python
    version: "3.12"
direnv: true
`)
	path := filepath.Join(dir, "answers.yaml")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	answers, err := devinit.ExportLoadAnswersFile(path)
	if err != nil {
		t.Fatalf("LoadAnswersFile: %v", err)
	}
	if len(answers.Languages) != 1 || answers.Languages[0].Name != "python" {
		t.Errorf("languages = %+v, want [{python 3.12}]", answers.Languages)
	}
}

func TestAnswersFile_EmptyFileError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(path, nil, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := devinit.ExportLoadAnswersFile(path)
	if err == nil {
		t.Fatal("expected error for empty file")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("error = %q, want it to contain 'empty'", err.Error())
	}
}

func TestAnswersFile_InvalidYAMLError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte("languages:\n  - [invalid\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := devinit.ExportLoadAnswersFile(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
	if !strings.Contains(err.Error(), "parsing") {
		t.Errorf("error = %q, want it to contain 'parsing'", err.Error())
	}
}

func TestAnswersFile_NonexistentFile(t *testing.T) {
	_, err := devinit.ExportLoadAnswersFile("/nonexistent/path/answers.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "/nonexistent/path/answers.yaml") {
		t.Errorf("error = %q, want it to contain the path", err.Error())
	}
}

func TestAnswersFile_MissingRequiredFields(t *testing.T) {
	answers := types.WizardAnswers{
		Direnv: true,
	}
	err := devinit.ExportValidateAnswersFileCompleteness(answers)
	if err == nil {
		t.Fatal("expected error for missing languages")
	}
	if !strings.Contains(err.Error(), "languages") {
		t.Errorf("error = %q, want it to mention 'languages'", err.Error())
	}
}

func TestAnswersFile_MissingPermissionLevel(t *testing.T) {
	answers := types.WizardAnswers{
		Languages:  []types.LanguageChoice{{Name: "go"}},
		ClaudeCode: true,
	}
	err := devinit.ExportValidateAnswersFileCompleteness(answers)
	if err == nil {
		t.Fatal("expected error for missing permission_level")
	}
	if !strings.Contains(err.Error(), "permission_level") {
		t.Errorf("error = %q, want it to mention 'permission_level'", err.Error())
	}
}

func TestAnswersFile_ValidComplete(t *testing.T) {
	answers := types.WizardAnswers{
		Languages:       []types.LanguageChoice{{Name: "go"}},
		ClaudeCode:      true,
		PermissionLevel: "standard",
	}
	if err := devinit.ExportValidateAnswersFileCompleteness(answers); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestAnswersFile_CompletenessMatchesIsComplete locks the single definition of
// answer completeness: the answers-file check must agree with
// WizardAnswers.IsComplete, which accepts a tier in place of a permission level.
func TestAnswersFile_CompletenessMatchesIsComplete(t *testing.T) {
	t.Parallel()
	goLang := []types.LanguageChoice{{Name: "go"}}
	tests := []struct {
		name    string
		answers types.WizardAnswers
	}{
		{"tier without permission level", types.WizardAnswers{Languages: goLang, ClaudeCode: true, Tier: "full"}},
		{"permission level", types.WizardAnswers{Languages: goLang, ClaudeCode: true, PermissionLevel: "standard"}},
		{"claude without permission or tier", types.WizardAnswers{Languages: goLang, ClaudeCode: true}},
		{"claude disabled", types.WizardAnswers{Languages: goLang}},
		{"no languages", types.WizardAnswers{ClaudeCode: true, Tier: "full"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			a := tt.answers
			a.Confirmed = true
			complete := a.IsComplete()
			err := devinit.ExportValidateAnswersFileCompleteness(tt.answers)
			if (err == nil) != complete {
				t.Errorf("ValidateAnswersFileCompleteness err = %v, but IsComplete = %v", err, complete)
			}
		})
	}
}

// TestAnswersFile_StrictDecoding is the regression test for silently dropped
// keys: a misspelled security setting must fail instead of defaulting to off.
func TestAnswersFile_StrictDecoding(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		wantErr string
	}{
		{"typo in nested hook key", "languages:\n  - name: go\nhooks:\n  safty_block: true\n", "safty_block"},
		{"typo in top-level key", "languages:\n  - name: go\npermision_level: minimal\n", "permision_level"},
		{"comment-only document", "# nothing here\n", "empty"},
		{"oversized input", "project_name: " + strings.Repeat("a", 2<<20) + "\n", "limit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := devinit.LoadAnswersFromReader(strings.NewReader(tt.input), "test.yaml")
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want it to mention %q", err, tt.wantErr)
			}
		})
	}
}
