package claudecode_test

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// rulePaths collects the generated rule paths into a set for easy assertions.
func rulePaths(t *testing.T, answers types.WizardAnswers) map[string]types.GeneratedFile {
	t.Helper()
	files, err := claudecode.ExportDeployRules(answers)
	if err != nil {
		t.Fatalf("deployRules returned error: %v", err)
	}
	byPath := make(map[string]types.GeneratedFile, len(files))
	for _, f := range files {
		byPath[f.Path] = f
	}
	return byPath
}

func TestDeployLSPRules_PresenceByLanguage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		languages []types.LanguageChoice
		present   []string
		absent    []string
	}{
		{
			name:      "go only",
			languages: []types.LanguageChoice{{Name: "go"}},
			present: []string{
				".claude/rules/lsp-navigation.md",
				".claude/rules/nix-lsp.md",
				".claude/rules/go-lsp.md",
			},
			absent: []string{".claude/rules/typescript-lsp.md"},
		},
		{
			name:      "empty languages",
			languages: nil,
			present: []string{
				".claude/rules/lsp-navigation.md",
				".claude/rules/nix-lsp.md",
			},
			absent: []string{
				".claude/rules/go-lsp.md",
				".claude/rules/typescript-lsp.md",
			},
		},
		{
			name:      "python only excludes go-lsp",
			languages: []types.LanguageChoice{{Name: "python"}},
			present: []string{
				".claude/rules/lsp-navigation.md",
				".claude/rules/nix-lsp.md",
				".claude/rules/python-lsp.md",
			},
			absent: []string{".claude/rules/go-lsp.md"},
		},
		{
			name:      "javascript gets typescript-lsp",
			languages: []types.LanguageChoice{{Name: "javascript"}},
			present: []string{
				".claude/rules/lsp-navigation.md",
				".claude/rules/typescript-lsp.md",
			},
			absent: []string{".claude/rules/go-lsp.md"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			paths := rulePaths(t, types.WizardAnswers{Languages: tc.languages})
			for _, p := range tc.present {
				if _, ok := paths[p]; !ok {
					t.Errorf("expected rule %q to be present", p)
				}
			}
			for _, p := range tc.absent {
				if _, ok := paths[p]; ok {
					t.Errorf("expected rule %q to be absent", p)
				}
			}
		})
	}
}

func TestDeployLSPRules_NavigationAlwaysPresent(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		languages []types.LanguageChoice
	}{
		{"go only", []types.LanguageChoice{{Name: "go"}}},
		{"empty", nil},
		{"unmapped language", []types.LanguageChoice{{Name: "ruby"}}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			paths := rulePaths(t, types.WizardAnswers{Languages: tc.languages})
			if _, ok := paths[".claude/rules/lsp-navigation.md"]; !ok {
				t.Error("lsp-navigation.md missing")
			}
			if _, ok := paths[".claude/rules/nix-lsp.md"]; !ok {
				t.Error("nix-lsp.md missing")
			}
		})
	}
}

func TestDeployLSPRules_PerLanguageFrontmatter(t *testing.T) {
	t.Parallel()

	// Detect every mapped per-language ecosystem at once so all per-language
	// LSP rules are emitted. nix-lsp.md is always present too.
	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{
			{Name: "go"},
			{Name: "javascript"},
			{Name: "rust"},
			{Name: "python"},
			{Name: "java"},
		},
	}
	paths := rulePaths(t, answers)

	perLanguage := []string{
		".claude/rules/go-lsp.md",
		".claude/rules/typescript-lsp.md",
		".claude/rules/rust-lsp.md",
		".claude/rules/python-lsp.md",
		".claude/rules/java-lsp.md",
		".claude/rules/nix-lsp.md",
	}

	for _, p := range perLanguage {
		f, ok := paths[p]
		if !ok {
			t.Errorf("expected per-language rule %q to be present", p)
			continue
		}
		content := string(f.Content)
		if !strings.HasPrefix(content, "---") {
			t.Errorf("%s: content should begin with frontmatter delimiter ---, got %.10q", p, content)
		}
		if !strings.Contains(content, "paths:") {
			t.Errorf("%s: content should contain a paths: frontmatter key", p)
		}
	}
}

func TestDeployLSPRules_Owner(t *testing.T) {
	t.Parallel()

	answers := types.WizardAnswers{
		Languages: []types.LanguageChoice{{Name: "go"}, {Name: "javascript"}},
	}
	files, err := claudecode.ExportDeployRules(answers)
	if err != nil {
		t.Fatalf("deployRules returned error: %v", err)
	}

	lspRules := map[string]bool{
		".claude/rules/lsp-navigation.md": true,
		".claude/rules/nix-lsp.md":        true,
		".claude/rules/go-lsp.md":         true,
		".claude/rules/typescript-lsp.md": true,
	}
	conventionRules := map[string]bool{
		".claude/rules/go-conventions.md":         true,
		".claude/rules/typescript-conventions.md": true,
		".claude/rules/security-rules.md":         true,
	}

	for _, f := range files {
		switch {
		case lspRules[f.Path]:
			if f.Owner != "lsp-rules" {
				t.Errorf("%s: Owner = %q, want %q", f.Path, f.Owner, "lsp-rules")
			}
			if f.Strategy != types.LibraryManaged {
				t.Errorf("%s: Strategy = %v, want LibraryManaged", f.Path, f.Strategy)
			}
		case conventionRules[f.Path]:
			// Convention rules intentionally set no Owner.
			if f.Owner != "" {
				t.Errorf("%s: convention rule should have empty Owner, got %q", f.Path, f.Owner)
			}
		}
	}
}

func TestGenerateClaudeMd_CodeNavigationSection(t *testing.T) {
	t.Parallel()

	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		ProjectName: "navtest",
		Languages:   []types.LanguageChoice{{Name: "go"}},
	}

	got, err := claudecode.GenerateClaudeMd(answers, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := string(got.Content)
	requireContains(t, content, "Code Navigation")
	requireContains(t, content, "lsp-navigation.md")
}
