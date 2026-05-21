package catalog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadWithOrgOverride_AddsTier(t *testing.T) {
	t.Parallel()

	f := writeUnifiedFile(t, `
tiers:
  custom-tier:
    order: 10
    description: "Custom org tier"
    default_permission_preset: standard
`)

	cat, err := Load(WithOrgConfigFile(f))
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

	f := writeUnifiedFile(t, `
tools:
  custom-scanner:
    display_name: "Custom Scanner"
    category: security
    description: "Org-specific security scanner"
    default_policy: opt-in
`)

	cat, err := Load(WithOrgConfigFile(f))
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

	f := writeUnifiedFile(t, `
project_profiles:
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

	cat, err := Load(WithProjectConfigFile(f))
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

func TestLoadWithOrgOverride_MissingFileIgnored(t *testing.T) {
	t.Parallel()

	cat, err := Load(WithOrgConfigFile("/nonexistent/path/defaults.yaml"))
	if err != nil {
		t.Fatalf("Load() should not error for missing org file: %v", err)
	}
	if cat == nil {
		t.Fatal("catalog should not be nil")
	}
}

func TestProjectConfigFile_ExistingFile(t *testing.T) {
	dir := t.TempDir()
	qsdevDir := filepath.Join(dir, ".qsdev")
	if err := os.MkdirAll(qsdevDir, 0o755); err != nil {
		t.Fatal(err)
	}
	defaultsFile := filepath.Join(qsdevDir, "defaults.yaml")
	if err := os.WriteFile(defaultsFile, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := ProjectConfigFile(dir)
	if result != defaultsFile {
		t.Errorf("ProjectConfigFile() = %q, want %q", result, defaultsFile)
	}
}

func TestProjectConfigFile_MissingFile(t *testing.T) {
	result := ProjectConfigFile(t.TempDir())
	if result != "" {
		t.Errorf("ProjectConfigFile() = %q, want empty", result)
	}
}

func TestProjectConfigFile_EmptyRoot(t *testing.T) {
	result := ProjectConfigFile("")
	if result != "" {
		t.Errorf("ProjectConfigFile() = %q, want empty", result)
	}
}

func writeUnifiedFile(t *testing.T, content string) string {
	t.Helper()
	f := filepath.Join(t.TempDir(), "defaults.yaml")
	if err := os.WriteFile(f, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return f
}

func TestLoadWithOrgOverride_MalformedYAML(t *testing.T) {
	t.Parallel()

	f := writeUnifiedFile(t, `
tiers:
  broken:
    - this: [is not valid
    yaml because the bracket is unclosed
`)

	_, err := Load(WithOrgConfigFile(f))
	if err == nil {
		t.Fatal("expected error for malformed YAML overlay, got nil")
	}
}

func TestLoadWithCombinedOrgAndProject(t *testing.T) {
	t.Parallel()

	orgFile := writeUnifiedFile(t, `
tiers:
  org-tier:
    order: 10
    description: "Org tier original"
    default_permission_preset: standard
`)

	projFile := writeUnifiedFile(t, `
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

	cat, err := Load(WithOrgConfigFile(orgFile), WithProjectConfigFile(projFile))
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

	f := writeUnifiedFile(t, `
profiles:
  broken-profile:
    tier: nonexistent-tier
    description: "This profile references a bad tier"
`)

	_, err := Load(WithOrgConfigFile(f))
	if err == nil {
		t.Fatal("expected validation error for overlay with bad tier reference")
	}
	if !strings.Contains(err.Error(), "nonexistent-tier") {
		t.Errorf("error should mention nonexistent-tier, got: %v", err)
	}
}
