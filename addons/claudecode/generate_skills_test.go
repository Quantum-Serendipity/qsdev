package claudecode_test

import (
	"strings"
	"testing"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/addons/claudecode"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// ---------------------------------------------------------------------------
// loadManifest tests
// ---------------------------------------------------------------------------

func TestLoadManifest_Valid(t *testing.T) {
	manifest, err := claudecode.ExportLoadManifest()
	if err != nil {
		t.Fatalf("loadManifest returned error: %v", err)
	}

	if len(manifest.Skills) != 6 {
		t.Errorf("expected 6 skills in manifest, got %d", len(manifest.Skills))
	}

	for i, s := range manifest.Skills {
		if s.Name == "" {
			t.Errorf("skill %d has empty name", i)
		}
		if s.Description == "" {
			t.Errorf("skill %d (%s) has empty description", i, s.Name)
		}
	}
}

func TestLoadManifest_AllSkillFilesExist(t *testing.T) {
	manifest, err := claudecode.ExportLoadManifest()
	if err != nil {
		t.Fatalf("loadManifest returned error: %v", err)
	}

	for _, s := range manifest.Skills {
		// Deploy each skill individually to verify the embedded file exists.
		answers := types.WizardAnswers{Skills: []string{s.Name}}
		files, err := claudecode.ExportDeploySkills(answers)
		if err != nil {
			t.Errorf("skill %q: file not found in embed: %v", s.Name, err)
			continue
		}
		if len(files) != 1 {
			t.Errorf("skill %q: expected 1 file, got %d", s.Name, len(files))
		}
	}
}

func TestLoadManifest_NoDuplicateNames(t *testing.T) {
	manifest, err := claudecode.ExportLoadManifest()
	if err != nil {
		t.Fatalf("loadManifest returned error: %v", err)
	}

	seen := make(map[string]bool)
	for _, s := range manifest.Skills {
		if seen[s.Name] {
			t.Errorf("duplicate skill name: %q", s.Name)
		}
		seen[s.Name] = true
	}
}

// ---------------------------------------------------------------------------
// deploySkills tests
// ---------------------------------------------------------------------------

func TestDeploySkills_SelectedOnly(t *testing.T) {
	answers := types.WizardAnswers{
		Skills: []string{"deploy", "review-pr"},
	}

	files, err := claudecode.ExportDeploySkills(answers)
	if err != nil {
		t.Fatalf("deploySkills returned error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}

	names := make(map[string]bool)
	for _, f := range files {
		names[f.Path] = true
	}
	if !names[".claude/skills/deploy.md"] {
		t.Error("missing .claude/skills/deploy.md")
	}
	if !names[".claude/skills/review-pr.md"] {
		t.Error("missing .claude/skills/review-pr.md")
	}
}

func TestDeploySkills_EmptySelection(t *testing.T) {
	answers := types.WizardAnswers{
		Skills: []string{},
	}

	files, err := claudecode.ExportDeploySkills(answers)
	if err != nil {
		t.Fatalf("deploySkills returned error: %v", err)
	}

	if len(files) != 0 {
		t.Errorf("expected 0 files for empty selection, got %d", len(files))
	}
}

func TestDeploySkills_UnknownSkill(t *testing.T) {
	answers := types.WizardAnswers{
		Skills: []string{"nonexistent-skill"},
	}

	_, err := claudecode.ExportDeploySkills(answers)
	if err == nil {
		t.Fatal("expected error for unknown skill, got nil")
	}

	if !strings.Contains(err.Error(), "nonexistent-skill") {
		t.Errorf("error should mention the unknown skill name, got: %v", err)
	}
}

func TestDeploySkills_ContentNotEmpty(t *testing.T) {
	manifest, err := claudecode.ExportLoadManifest()
	if err != nil {
		t.Fatalf("loadManifest returned error: %v", err)
	}

	allNames := make([]string, 0, len(manifest.Skills))
	for _, s := range manifest.Skills {
		allNames = append(allNames, s.Name)
	}

	answers := types.WizardAnswers{Skills: allNames}
	files, err := claudecode.ExportDeploySkills(answers)
	if err != nil {
		t.Fatalf("deploySkills returned error: %v", err)
	}

	for _, f := range files {
		if len(f.Content) == 0 {
			t.Errorf("skill file %q has empty content", f.Path)
		}
	}
}

func TestDeploySkills_FileMetadata(t *testing.T) {
	answers := types.WizardAnswers{
		Skills: []string{"deploy", "security-review"},
	}

	files, err := claudecode.ExportDeploySkills(answers)
	if err != nil {
		t.Fatalf("deploySkills returned error: %v", err)
	}

	for _, f := range files {
		if !strings.HasPrefix(f.Path, ".claude/skills/") {
			t.Errorf("path %q does not start with .claude/skills/", f.Path)
		}
		if !strings.HasSuffix(f.Path, ".md") {
			t.Errorf("path %q does not end with .md", f.Path)
		}
		if f.Mode != 0o644 {
			t.Errorf("file %q has mode %o, want 0644", f.Path, f.Mode)
		}
		if f.Strategy != types.LibraryManaged {
			t.Errorf("file %q has strategy %v, want LibraryManaged", f.Path, f.Strategy)
		}
	}
}

// ---------------------------------------------------------------------------
// deployRules tests
// ---------------------------------------------------------------------------

func TestDeployRules_GoProject(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go"},
		},
	}

	files, err := claudecode.ExportDeployRules(answers)
	if err != nil {
		t.Fatalf("deployRules returned error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files (go-conventions + security-rules), got %d", len(files))
	}

	paths := make(map[string]bool)
	for _, f := range files {
		paths[f.Path] = true
	}
	if !paths[".claude/rules/go-conventions.md"] {
		t.Error("missing go-conventions.md")
	}
	if !paths[".claude/rules/security-rules.md"] {
		t.Error("missing security-rules.md")
	}
}

func TestDeployRules_MultiLanguage(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go"},
			{Name: "javascript"},
		},
	}

	files, err := claudecode.ExportDeployRules(answers)
	if err != nil {
		t.Fatalf("deployRules returned error: %v", err)
	}

	if len(files) != 3 {
		t.Fatalf("expected 3 files (go-conventions + typescript-conventions + security-rules), got %d", len(files))
	}

	paths := make(map[string]bool)
	for _, f := range files {
		paths[f.Path] = true
	}
	if !paths[".claude/rules/go-conventions.md"] {
		t.Error("missing go-conventions.md")
	}
	if !paths[".claude/rules/typescript-conventions.md"] {
		t.Error("missing typescript-conventions.md")
	}
	if !paths[".claude/rules/security-rules.md"] {
		t.Error("missing security-rules.md")
	}
}

func TestDeployRules_NoLanguages(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{},
	}

	files, err := claudecode.ExportDeployRules(answers)
	if err != nil {
		t.Fatalf("deployRules returned error: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file (security-rules only), got %d", len(files))
	}

	if files[0].Path != ".claude/rules/security-rules.md" {
		t.Errorf("expected security-rules.md, got %q", files[0].Path)
	}
}

func TestDeployRules_SecurityAlwaysPresent(t *testing.T) {
	testCases := []struct {
		name      string
		languages []types.LanguageChoice
	}{
		{"no languages", nil},
		{"go only", []types.LanguageChoice{{Name: "go"}}},
		{"python only", []types.LanguageChoice{{Name: "python"}}},
		{"rust only", []types.LanguageChoice{{Name: "rust"}}},
		{"javascript only", []types.LanguageChoice{{Name: "javascript"}}},
		{"all languages", []types.LanguageChoice{
			{Name: "go"}, {Name: "python"}, {Name: "rust"}, {Name: "javascript"},
		}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			answers := types.WizardAnswers{Languages: tc.languages}
			files, err := claudecode.ExportDeployRules(answers)
			if err != nil {
				t.Fatalf("deployRules returned error: %v", err)
			}

			found := false
			for _, f := range files {
				if f.Path == ".claude/rules/security-rules.md" {
					found = true
					break
				}
			}
			if !found {
				t.Error("security-rules.md not found in output")
			}
		})
	}
}

func TestDeployRules_RustProject(t *testing.T) {
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "rust"},
		},
	}

	files, err := claudecode.ExportDeployRules(answers)
	if err != nil {
		t.Fatalf("deployRules returned error: %v", err)
	}

	if len(files) != 2 {
		t.Fatalf("expected 2 files (rust-conventions + security-rules), got %d", len(files))
	}

	paths := make(map[string]bool)
	for _, f := range files {
		paths[f.Path] = true
	}
	if !paths[".claude/rules/rust-conventions.md"] {
		t.Error("missing rust-conventions.md")
	}
	if !paths[".claude/rules/security-rules.md"] {
		t.Error("missing security-rules.md")
	}
}
