package catalog

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadWithOrgOverride_AddsTier(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "tiers.yaml"), `
tiers:
  custom-tier:
    order: 10
    description: "Custom org tier"
    default_permission_preset: standard
`)

	cat, err := Load(WithOrgConfig(dir))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if _, ok := cat.TierDef("custom-tier"); !ok {
		t.Error("custom-tier should be present after org overlay")
	}
	if _, ok := cat.TierDef("full"); !ok {
		t.Error("full tier should still be present after org overlay")
	}
}

func TestLoadWithOrgOverride_AddsTool(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "tools.yaml"), `
tools:
  custom-scanner:
    display_name: "Custom Scanner"
    category: security
    description: "Org-specific security scanner"
    default_policy: opt-in
`)

	cat, err := Load(WithOrgConfig(dir))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if _, ok := cat.Tool("custom-scanner"); !ok {
		t.Error("custom-scanner should be present after org overlay")
	}
	if _, ok := cat.Tool("semgrep"); !ok {
		t.Error("semgrep should still be present after org overlay")
	}
}

func TestLoadWithProjectOverride_AddsProjectProfile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "project_profiles.yaml"), `
profiles:
  custom-web:
    description: "Custom project profile"
    tier: full
    languages:
      - name: go
    services: []
    direnv: true
    claude_code: true
    permission_level: standard
`)

	cat, err := Load(WithProjectConfig(dir))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if _, ok := cat.ProjectProfile("custom-web"); !ok {
		t.Error("custom-web should be present after project overlay")
	}
	if _, ok := cat.ProjectProfile("go-web"); !ok {
		t.Error("go-web should still be present")
	}
}

func TestLoadWithOrgOverride_MissingDirIgnored(t *testing.T) {
	t.Parallel()

	cat, err := Load(WithOrgConfig("/nonexistent/path"))
	if err != nil {
		t.Fatalf("Load() should not error for missing org dir: %v", err)
	}
	if cat == nil {
		t.Fatal("catalog should not be nil")
	}
}

func TestProjectConfigDir_ExistingDir(t *testing.T) {
	dir := t.TempDir()
	catalogDir := filepath.Join(dir, ".qsdev", "catalog")
	if err := os.MkdirAll(catalogDir, 0o755); err != nil {
		t.Fatal(err)
	}

	result := ProjectConfigDir(dir)
	if result != catalogDir {
		t.Errorf("ProjectConfigDir() = %q, want %q", result, catalogDir)
	}
}

func TestProjectConfigDir_MissingDir(t *testing.T) {
	result := ProjectConfigDir(t.TempDir())
	if result != "" {
		t.Errorf("ProjectConfigDir() = %q, want empty", result)
	}
}

func TestProjectConfigDir_EmptyRoot(t *testing.T) {
	result := ProjectConfigDir("")
	if result != "" {
		t.Errorf("ProjectConfigDir() = %q, want empty", result)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
