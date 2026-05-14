package teamreport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadScopeFileValid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scope.json")

	scope := ScopeFile{
		Projects: []ScopeProject{
			{Repo: "org/repo-a", Branch: "main"},
			{Repo: "org/repo-b"},
		},
	}

	data, _ := json.MarshalIndent(scope, "", "  ")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadScopeFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(loaded.Projects) != 2 {
		t.Errorf("expected 2 projects, got %d", len(loaded.Projects))
	}

	if loaded.Projects[0].Repo != "org/repo-a" {
		t.Errorf("expected repo 'org/repo-a', got %q", loaded.Projects[0].Repo)
	}

	if loaded.Projects[0].Branch != "main" {
		t.Errorf("expected branch 'main', got %q", loaded.Projects[0].Branch)
	}

	if loaded.Projects[1].Repo != "org/repo-b" {
		t.Errorf("expected repo 'org/repo-b', got %q", loaded.Projects[1].Repo)
	}
}

func TestLoadScopeFileEmptyProjects(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scope.json")

	scope := ScopeFile{
		Projects: []ScopeProject{},
	}

	data, _ := json.MarshalIndent(scope, "", "  ")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadScopeFile(path)
	if err == nil {
		t.Fatal("expected error for empty projects")
	}

	if !strings.Contains(err.Error(), "no projects defined") {
		t.Errorf("expected 'no projects defined' error, got: %v", err)
	}
}

func TestLoadScopeFileMissingRepo(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scope.json")

	scope := ScopeFile{
		Projects: []ScopeProject{
			{Repo: "org/valid"},
			{Repo: ""},
		},
	}

	data, _ := json.MarshalIndent(scope, "", "  ")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadScopeFile(path)
	if err == nil {
		t.Fatal("expected error for missing repo")
	}

	if !strings.Contains(err.Error(), "empty repo") {
		t.Errorf("expected 'empty repo' error, got: %v", err)
	}
}

func TestLoadScopeFileNotFound(t *testing.T) {
	_, err := LoadScopeFile("/tmp/nonexistent-scope-file.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadScopeFileInvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scope.json")

	if err := os.WriteFile(path, []byte("{invalid}"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadScopeFile(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestSanitizeRepoName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"org/repo", "org-repo"},
		{"my-org/my-repo", "my-org-my-repo"},
		{"simple", "simple"},
	}

	for _, tt := range tests {
		got := sanitizeRepoName(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeRepoName(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
