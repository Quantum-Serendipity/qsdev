package catalog

import (
	"os"
	"path/filepath"
	"slices"
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

	result, err := ProjectConfigFile(dir)
	if err != nil {
		t.Fatalf("ProjectConfigFile() error: %v", err)
	}
	if result != defaultsFile {
		t.Errorf("ProjectConfigFile() = %q, want %q", result, defaultsFile)
	}
}

func TestProjectConfigFile_MissingFile(t *testing.T) {
	result, err := ProjectConfigFile(t.TempDir())
	if err != nil || result != "" {
		t.Errorf("ProjectConfigFile() = (%q, %v), want empty", result, err)
	}
}

func TestProjectConfigFile_EmptyRoot(t *testing.T) {
	result, err := ProjectConfigFile("")
	if err != nil || result != "" {
		t.Errorf("ProjectConfigFile() = (%q, %v), want empty", result, err)
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

// The committed project layer applies after the org layer (the developer's
// own defaults) and may only add: where both set the same deny set, the
// result is the org file's list plus the project's additions, so the org
// file cannot erase a rule the project committed (G-V02).
func TestLoadWithCombinedOrgAndProject(t *testing.T) {
	t.Parallel()

	orgFile := writeUnifiedFile(t, `
permission_deny_rules:
  npx:
    - Bash(org-deny *)
`)

	projFile := writeUnifiedFile(t, `
permission_deny_rules:
  npx:
    - Bash(project-deny *)
  project_extra:
    - Bash(project-extra *)
permission_all_deny_sets:
  - project_extra
`)

	cat, err := Load(WithOrgConfigFile(orgFile), WithProjectConfigFile(projFile))
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}

	if got, want := cat.PermissionDenyRules("npx"), []string{"Bash(org-deny *)", "Bash(project-deny *)"}; !slices.Equal(got, want) {
		t.Errorf("npx deny rules = %v, want org then project %v", got, want)
	}
	if got := cat.PermissionDenyRules("project_extra"); !slices.Equal(got, []string{"Bash(project-extra *)"}) {
		t.Errorf("project_extra deny rules = %v, want the project's rule", got)
	}
	if !slices.Contains(cat.AllPermissionDenyRules(), "Bash(project-extra *)") {
		t.Error("project deny set missing from AllPermissionDenyRules()")
	}
}

func TestLoadWithOverlay_BreaksValidation(t *testing.T) {
	t.Parallel()

	f := writeUnifiedFile(t, `
project_profiles:
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
