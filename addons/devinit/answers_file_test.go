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
		Languages: []types.LanguageChoice{{Name: "go"}},
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
