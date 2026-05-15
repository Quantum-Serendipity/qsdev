package claudecode_test

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
)

func TestResolveModelSize_Sonnet(t *testing.T) {
	got := claudecode.ResolveModelSize("sonnet")
	if got != "sonnet" {
		t.Errorf("ResolveModelSize(\"sonnet\") = %q, want %q", got, "sonnet")
	}
}

func TestResolveModelSize_Opus(t *testing.T) {
	got := claudecode.ResolveModelSize("opus")
	if got != "opus" {
		t.Errorf("ResolveModelSize(\"opus\") = %q, want %q", got, "opus")
	}
}

func TestResolveModelSize_AutoDefaultsSonnet(t *testing.T) {
	for _, input := range []string{"auto", "", "unknown", "haiku"} {
		got := claudecode.ResolveModelSize(input)
		if got != "sonnet" {
			t.Errorf("ResolveModelSize(%q) = %q, want %q", input, got, "sonnet")
		}
	}
}

func TestMaxTokensForModel(t *testing.T) {
	tests := []struct {
		model string
		want  int
	}{
		{"opus", 1_000_000},
		{"sonnet", 200_000},
		{"unknown", 200_000},
	}
	for _, tt := range tests {
		got := claudecode.MaxTokensForModel(tt.model)
		if got != tt.want {
			t.Errorf("MaxTokensForModel(%q) = %d, want %d", tt.model, got, tt.want)
		}
	}
}

func TestEstimateTokens(t *testing.T) {
	// 400 bytes -> 100 tokens at 4 bytes/token
	content := bytes.Repeat([]byte("x"), 400)
	got := claudecode.EstimateTokens(content)
	if got != 100 {
		t.Errorf("EstimateTokens(400 bytes) = %d, want 100", got)
	}

	// Empty content -> 0
	got = claudecode.EstimateTokens(nil)
	if got != 0 {
		t.Errorf("EstimateTokens(nil) = %d, want 0", got)
	}
}

func TestContextBudget_Validate_UnderBudget(t *testing.T) {
	b := claudecode.ContextBudget{
		ModelSize:   "opus",
		MaxTokens:   1_000_000,
		TotalTokens: 10_000,
		BudgetPct:   1.0,
	}
	if err := b.Validate(); err != nil {
		t.Errorf("Validate() returned error for under-budget: %v", err)
	}
}

func TestContextBudget_Validate_OverBudget(t *testing.T) {
	b := claudecode.ContextBudget{
		ModelSize:   "sonnet",
		MaxTokens:   200_000,
		TotalTokens: 20_000,
		BudgetPct:   10.0,
	}
	err := b.Validate()
	if err == nil {
		t.Fatal("Validate() should return error for over-budget")
	}
	if !strings.Contains(err.Error(), "5%") {
		t.Errorf("error should mention 5%% threshold, got: %v", err)
	}
	if !strings.Contains(err.Error(), "sonnet") {
		t.Errorf("error should mention model name, got: %v", err)
	}
}

func TestContextBudget_FormatReport(t *testing.T) {
	b := claudecode.ContextBudget{
		ModelSize:       "opus",
		MaxTokens:       1_000_000,
		ClaudeMdTokens:  500,
		RulesTokens:     300,
		SkillDescTokens: 200,
		TotalTokens:     1000,
		BudgetPct:       0.1,
	}

	var buf bytes.Buffer
	b.FormatReport(&buf)
	output := buf.String()

	requireContains(t, output, "opus")
	requireContains(t, output, "1000000")
	requireContains(t, output, "500")
	requireContains(t, output, "300")
	requireContains(t, output, "200")
	requireContains(t, output, "1000")
	requireContains(t, output, "Within budget")

	// Over budget report
	b.BudgetPct = 7.0
	buf.Reset()
	b.FormatReport(&buf)
	output = buf.String()
	requireContains(t, output, "OVER BUDGET")
}

func TestCalculateContextBudget(t *testing.T) {
	dir := t.TempDir()

	// Write a CLAUDE.md
	claudeContent := bytes.Repeat([]byte("x"), 800)
	if err := os.WriteFile(filepath.Join(dir, "CLAUDE.md"), claudeContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Write rules directory with a file
	rulesDir := filepath.Join(dir, ".claude", "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	ruleContent := bytes.Repeat([]byte("y"), 400)
	if err := os.WriteFile(filepath.Join(rulesDir, "test.md"), ruleContent, 0o644); err != nil {
		t.Fatal(err)
	}

	budget, err := claudecode.CalculateContextBudget(dir, "opus")
	if err != nil {
		t.Fatalf("CalculateContextBudget: %v", err)
	}

	if budget.ModelSize != "opus" {
		t.Errorf("ModelSize = %q, want %q", budget.ModelSize, "opus")
	}
	if budget.MaxTokens != 1_000_000 {
		t.Errorf("MaxTokens = %d, want 1000000", budget.MaxTokens)
	}
	// 800/4 = 200 tokens for CLAUDE.md
	if budget.ClaudeMdTokens != 200 {
		t.Errorf("ClaudeMdTokens = %d, want 200", budget.ClaudeMdTokens)
	}
	// 400/4 = 100 tokens for rules
	if budget.RulesTokens != 100 {
		t.Errorf("RulesTokens = %d, want 100", budget.RulesTokens)
	}
	if budget.TotalTokens != 300 {
		t.Errorf("TotalTokens = %d, want 300", budget.TotalTokens)
	}
}
