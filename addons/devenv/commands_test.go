package devenv_test

import (
	"testing"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/addons/devenv"
)

func TestBuildAnswersFromFlags(t *testing.T) {
	projectRoot := "/tmp/test-project"
	langs := []string{"go", "javascript"}
	services := []string{"postgres", "redis"}
	direnvEnabled := true

	answers := devenv.ExportBuildAnswersFromFlags(projectRoot, langs, services, direnvEnabled)

	if answers.ProjectRoot != projectRoot {
		t.Errorf("ProjectRoot = %q, want %q", answers.ProjectRoot, projectRoot)
	}
	if answers.ProjectName != "test-project" {
		t.Errorf("ProjectName = %q, want %q", answers.ProjectName, "test-project")
	}
	if answers.Direnv != direnvEnabled {
		t.Errorf("Direnv = %v, want %v", answers.Direnv, direnvEnabled)
	}
	if len(answers.Languages) != 2 {
		t.Fatalf("Languages count = %d, want 2", len(answers.Languages))
	}
	if answers.Languages[0].Name != "go" {
		t.Errorf("Languages[0].Name = %q, want %q", answers.Languages[0].Name, "go")
	}
	if answers.Languages[1].Name != "javascript" {
		t.Errorf("Languages[1].Name = %q, want %q", answers.Languages[1].Name, "javascript")
	}
	if len(answers.Services) != 2 {
		t.Fatalf("Services count = %d, want 2", len(answers.Services))
	}
	if answers.Services[0].Name != "postgres" {
		t.Errorf("Services[0].Name = %q, want %q", answers.Services[0].Name, "postgres")
	}
	if answers.Services[1].Name != "redis" {
		t.Errorf("Services[1].Name = %q, want %q", answers.Services[1].Name, "redis")
	}
}

func TestBuildAnswersFromFlags_Empty(t *testing.T) {
	answers := devenv.ExportBuildAnswersFromFlags("/tmp/empty", nil, nil, false)

	if answers.Direnv != false {
		t.Errorf("Direnv = %v, want false", answers.Direnv)
	}
	if len(answers.Languages) != 0 {
		t.Errorf("Languages count = %d, want 0", len(answers.Languages))
	}
	if len(answers.Services) != 0 {
		t.Errorf("Services count = %d, want 0", len(answers.Services))
	}
}

func TestValidServices(t *testing.T) {
	expected := map[string]bool{
		"postgres":      true,
		"redis":         true,
		"mysql":         true,
		"mongodb":       true,
		"elasticsearch": true,
		"rabbitmq":      true,
	}

	services := devenv.ExportValidServices
	if len(services) != len(expected) {
		t.Fatalf("validServices has %d entries, want %d", len(services), len(expected))
	}

	for _, svc := range services {
		if !expected[svc] {
			t.Errorf("unexpected service in validServices: %q", svc)
		}
	}
}

func TestValidLanguages(t *testing.T) {
	expected := map[string]bool{
		"go":         true,
		"javascript": true,
		"python":     true,
		"rust":       true,
		"java":       true,
		"dotnet":     true,
		"docker":     true,
		"terraform":  true,
	}

	languages := devenv.ExportValidLanguages
	if len(languages) != len(expected) {
		t.Fatalf("validLanguages has %d entries, want %d", len(languages), len(expected))
	}

	for _, lang := range languages {
		if !expected[lang] {
			t.Errorf("unexpected language in validLanguages: %q", lang)
		}
	}
}
