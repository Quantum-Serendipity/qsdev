package devenv_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/addons/devenv"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

func TestSaveAndLoadAnswers_RoundTrip(t *testing.T) {
	tmpDir := t.TempDir()

	original := types.WizardAnswers{
		ProjectName: "test-project",
		ProjectRoot: tmpDir,
		Languages: []types.LanguageChoice{
			{Name: "go", Version: "1.24", PackageManager: "gomod"},
			{Name: "javascript", Version: "22", PackageManager: "npm"},
		},
		Services: []types.ServiceChoice{
			{Name: "postgres", Version: "16"},
			{Name: "redis"},
		},
		Direnv:   true,
		GitHooks: []string{"ripsecrets"},
		EnvVars:  map[string]string{"FOO": "bar"},
	}

	if err := devenv.ExportSaveAnswers(tmpDir, original); err != nil {
		t.Fatalf("saveAnswers failed: %v", err)
	}

	loaded, err := devenv.ExportLoadAnswers(tmpDir)
	if err != nil {
		t.Fatalf("loadAnswers failed: %v", err)
	}

	// Verify key fields match.
	if loaded.ProjectName != original.ProjectName {
		t.Errorf("ProjectName = %q, want %q", loaded.ProjectName, original.ProjectName)
	}
	if len(loaded.Languages) != len(original.Languages) {
		t.Fatalf("Languages count = %d, want %d", len(loaded.Languages), len(original.Languages))
	}
	for i, lang := range loaded.Languages {
		if lang.Name != original.Languages[i].Name {
			t.Errorf("Languages[%d].Name = %q, want %q", i, lang.Name, original.Languages[i].Name)
		}
		if lang.Version != original.Languages[i].Version {
			t.Errorf("Languages[%d].Version = %q, want %q", i, lang.Version, original.Languages[i].Version)
		}
	}
	if len(loaded.Services) != len(original.Services) {
		t.Fatalf("Services count = %d, want %d", len(loaded.Services), len(original.Services))
	}
	for i, svc := range loaded.Services {
		if svc.Name != original.Services[i].Name {
			t.Errorf("Services[%d].Name = %q, want %q", i, svc.Name, original.Services[i].Name)
		}
	}
	if loaded.Direnv != original.Direnv {
		t.Errorf("Direnv = %v, want %v", loaded.Direnv, original.Direnv)
	}
	if len(loaded.GitHooks) != len(original.GitHooks) {
		t.Errorf("GitHooks count = %d, want %d", len(loaded.GitHooks), len(original.GitHooks))
	}
	if loaded.EnvVars["FOO"] != "bar" {
		t.Errorf("EnvVars[FOO] = %q, want %q", loaded.EnvVars["FOO"], "bar")
	}
}

func TestLoadAnswers_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	_, err := devenv.ExportLoadAnswers(tmpDir)
	if err == nil {
		t.Fatal("expected error when answers file does not exist, got nil")
	}
}

func TestSaveAnswers_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	answers := types.WizardAnswers{
		ProjectName: "test",
	}

	if err := devenv.ExportSaveAnswers(tmpDir, answers); err != nil {
		t.Fatalf("saveAnswers failed: %v", err)
	}

	// Verify .devenv/ directory was created.
	devenvDir := filepath.Join(tmpDir, ".devenv")
	info, err := os.Stat(devenvDir)
	if err != nil {
		t.Fatalf(".devenv directory not created: %v", err)
	}
	if !info.IsDir() {
		t.Error(".devenv should be a directory")
	}

	// Verify the answers file exists.
	answersFile := filepath.Join(devenvDir, ".gdev-answers.yaml")
	if _, err := os.Stat(answersFile); err != nil {
		t.Fatalf("answers file not created: %v", err)
	}
}
