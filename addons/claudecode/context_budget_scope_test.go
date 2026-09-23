package claudecode_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
)

// writeSized writes a file of n bytes (n/4 estimated tokens) under dir,
// optionally starting with prefix.
func writeSized(t *testing.T, dir, rel, prefix string, n int) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := prefix + strings.Repeat("x", n-len(prefix))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestCalculateContextBudget_FullScope guards F105: imports, nested rules and
// skill descriptions all count toward the budget.
func TestCalculateContextBudget_FullScope(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	outside := t.TempDir()

	claudeMd := strings.Join([]string{
		"# Project",
		"@docs/ref.md",
		"Ignored in code: `@docs/ignored.md` and mail me at dev@docs/ignored.md",
		"```",
		"@docs/ignored.md",
		"```",
		"Not a file: @upstash/context7-mcp",
		"Escapes the project: @" + filepath.ToSlash(filepath.Join("..", filepath.Base(outside), "big.md")),
		"",
	}, "\n")
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), []byte(claudeMd), 0o644); err != nil {
		t.Fatal(err)
	}
	// docs/ref.md imports nested.md relative to itself, and CLAUDE.md again
	// (a cycle that must not be double-counted).
	writeSized(t, dir, "docs/ref.md", "@nested.md @../CLAUDE.md\n", 400)
	writeSized(t, dir, "docs/nested.md", "", 800)
	writeSized(t, dir, "docs/ignored.md", "", 40000)
	writeSized(t, outside, "big.md", "", 40000)

	writeSized(t, dir, ".claude/rules/top.md", "", 400)
	writeSized(t, dir, ".claude/rules/lang/go.md", "", 400)
	writeSized(t, dir, ".claude/rules/notes.txt", "", 4000)

	skill := "---\nname: deploy\ndescription: " + strings.Repeat("d", 393) + "\n---\n" + strings.Repeat("body", 1000)
	writeSized(t, dir, ".claude/skills/deploy/SKILL.md", skill, len(skill))

	budget, err := claudecode.CalculateContextBudget(dir, "opus")
	if err != nil {
		t.Fatalf("CalculateContextBudget: %v", err)
	}

	wantClaude := claudecode.EstimateTokens([]byte(claudeMd)) + 100 + 200
	if budget.ClaudeMdTokens != wantClaude {
		t.Errorf("ClaudeMdTokens = %d, want %d (CLAUDE.md + imports only)", budget.ClaudeMdTokens, wantClaude)
	}
	if budget.RulesTokens != 200 {
		t.Errorf("RulesTokens = %d, want 200 (nested .md rules, no .txt)", budget.RulesTokens)
	}
	// "deploy\n" + 393 chars = 400 bytes = 100 tokens.
	if budget.SkillDescTokens != 100 {
		t.Errorf("SkillDescTokens = %d, want 100", budget.SkillDescTokens)
	}
	if want := budget.ClaudeMdTokens + budget.RulesTokens + budget.SkillDescTokens; budget.TotalTokens != want {
		t.Errorf("TotalTokens = %d, want %d", budget.TotalTokens, want)
	}
}

// TestCalculateContextBudget_ReadErrors guards F105: an unreadable context file
// is reported, not silently counted as zero.
func TestCalculateContextBudget_ReadErrors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(t *testing.T, dir string)
	}{
		{"CLAUDE.md is a directory", func(t *testing.T, dir string) {
			if err := os.Mkdir(filepath.Join(dir, "CLAUDE.md"), 0o755); err != nil {
				t.Fatal(err)
			}
		}},
		{"skills path is a file", func(t *testing.T, dir string) {
			writeSized(t, dir, ".claude/skills", "", 4)
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			tc.setup(t, dir)
			if _, err := claudecode.CalculateContextBudget(dir, "sonnet"); err == nil {
				t.Error("expected an error")
			}
		})
	}
}
