package catalog

import (
	"os"
	"path/filepath"
	"strings"
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

func TestLoadWithOrgOverride_MalformedYAML(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "tiers.yaml"), `
tiers:
  broken:
    - this: [is not valid
    yaml because the bracket is unclosed
`)

	_, err := Load(WithOrgConfig(dir))
	if err == nil {
		t.Fatal("expected error for malformed YAML overlay, got nil")
	}
}

func TestLoadWithCombinedOrgAndProject(t *testing.T) {
	t.Parallel()

	orgDir := t.TempDir()
	writeFile(t, filepath.Join(orgDir, "tiers.yaml"), `
tiers:
  org-tier:
    order: 10
    description: "Org tier original"
    default_permission_preset: standard
`)

	projDir := t.TempDir()
	writeFile(t, filepath.Join(projDir, "tiers.yaml"), `
tiers:
  org-tier:
    order: 10
    description: "Org tier overridden by project"
    default_permission_preset: standard
  proj-tier:
    order: 11
    description: "Project-only tier"
    default_permission_preset: standard
`)

	cat, err := Load(WithOrgConfig(orgDir), WithProjectConfig(projDir))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if _, ok := cat.TierDef("full"); !ok {
		t.Error("base tier 'full' missing")
	}

	orgTier, ok := cat.TierDef("org-tier")
	if !ok {
		t.Fatal("org-tier missing")
	}
	if orgTier.Description != "Org tier overridden by project" {
		t.Errorf("org-tier description = %q, want project override", orgTier.Description)
	}

	if _, ok := cat.TierDef("proj-tier"); !ok {
		t.Error("proj-tier missing")
	}
}

func TestLoadWithOverlay_BreaksValidation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "profiles.yaml"), `
profiles:
  broken-profile:
    tier: nonexistent-tier
    description: "This profile references a bad tier"
`)

	_, err := Load(WithOrgConfig(dir))
	if err == nil {
		t.Fatal("expected validation error for overlay with bad tier reference")
	}
	if !strings.Contains(err.Error(), "nonexistent-tier") {
		t.Errorf("error should mention nonexistent-tier, got: %v", err)
	}
}
