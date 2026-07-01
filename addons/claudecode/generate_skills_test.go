package claudecode_test

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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
	// Skills must be written as <name>/SKILL.md so Claude Code can load them.
	if !names[".claude/skills/deploy/SKILL.md"] {
		t.Error("missing .claude/skills/deploy/SKILL.md")
	}
	if !names[".claude/skills/review-pr/SKILL.md"] {
		t.Error("missing .claude/skills/review-pr/SKILL.md")
	}
}

// TestDeploySkills_SynthesizesFrontMatter guards DEFECT-8: every deployed skill
// must be a <name>/SKILL.md with valid YAML front-matter (name + description),
// or Claude Code cannot load it.
func TestDeploySkills_SynthesizesFrontMatter(t *testing.T) {
	answers := types.WizardAnswers{Skills: []string{"deploy", "security-review-owasp"}}
	files, err := claudecode.ExportDeploySkills(answers)
	if err != nil {
		t.Fatalf("deploySkills returned error: %v", err)
	}
	for _, f := range files {
		if !strings.HasSuffix(f.Path, "/SKILL.md") {
			t.Errorf("path %q should end with /SKILL.md", f.Path)
		}
		if !strings.HasPrefix(string(f.Content), "---\n") {
			t.Errorf("%s should begin with YAML front-matter delimiter", f.Path)
		}
		var fm struct {
			Name        string `yaml:"name"`
			Description string `yaml:"description"`
		}
		// Extract the front-matter block (between the first two --- lines).
		parts := strings.SplitN(string(f.Content), "---\n", 3)
		if len(parts) < 3 {
			t.Fatalf("%s has no closed front-matter block", f.Path)
		}
		if err := yaml.Unmarshal([]byte(parts[1]), &fm); err != nil {
			t.Errorf("%s front-matter is not valid YAML: %v", f.Path, err)
		}
		if fm.Name == "" || fm.Description == "" {
			t.Errorf("%s front-matter missing name/description: %+v", f.Path, fm)
		}
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
		Skills: []string{"deploy", "security-review-owasp"},
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
			t.Errorf("file %q has mode %o, want 0o644", f.Path, f.Mode)
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

	// go-conventions + security-rules, plus the always-on LSP rules
	// (lsp-navigation + nix-lsp) and the per-language go-lsp rule = 5.
	if len(files) != 5 {
		t.Fatalf("expected 5 files (go-conventions + security-rules + lsp-navigation + nix-lsp + go-lsp), got %d", len(files))
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

	// go-conventions + typescript-conventions + security-rules, plus the
	// always-on LSP rules (lsp-navigation + nix-lsp) and the per-language
	// go-lsp + typescript-lsp rules = 7.
	if len(files) != 7 {
		t.Fatalf("expected 7 files (go-conventions + typescript-conventions + security-rules + lsp-navigation + nix-lsp + go-lsp + typescript-lsp), got %d", len(files))
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

	// security-rules only for conventions, plus the always-on LSP rules
	// (lsp-navigation + nix-lsp) = 3.
	if len(files) != 3 {
		t.Fatalf("expected 3 files (security-rules + lsp-navigation + nix-lsp), got %d", len(files))
	}

	paths := make(map[string]bool)
	for _, f := range files {
		paths[f.Path] = true
	}
	if !paths[".claude/rules/security-rules.md"] {
		t.Error("missing security-rules.md")
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

	// rust-conventions + security-rules, plus the always-on LSP rules
	// (lsp-navigation + nix-lsp) and the per-language rust-lsp rule = 5.
	if len(files) != 5 {
		t.Fatalf("expected 5 files (rust-conventions + security-rules + lsp-navigation + nix-lsp + rust-lsp), got %d", len(files))
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
