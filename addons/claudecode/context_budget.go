package claudecode

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

const (
	// ModelSonnet represents the Sonnet model with a 200k token context window.
	ModelSonnet = "sonnet"
	// ModelOpus represents the Opus model with a 1M token context window.
	ModelOpus = "opus"
	// ModelAuto indicates automatic model selection (defaults to sonnet).
	ModelAuto = "auto"
)

// ContextBudget holds the measured token usage of all generated context files
// and compares it against the model's context window.
type ContextBudget struct {
	ModelSize       string
	MaxTokens       int
	ClaudeMdTokens  int
	RulesTokens     int
	SkillDescTokens int
	TotalTokens     int
	BudgetPct       float64
}

// ResolveModelSize normalizes a raw model size string to a known value.
// Unknown values default to sonnet (conservative).
func ResolveModelSize(raw string) string {
	switch raw {
	case ModelSonnet, ModelOpus:
		return raw
	default:
		return ModelSonnet
	}
}

// MaxTokensForModel returns the context window size for a given model.
func MaxTokensForModel(model string) int {
	switch model {
	case ModelOpus:
		return 1_000_000
	default:
		return 200_000
	}
}

// EstimateTokens provides a rough token estimate from byte content.
// Uses the standard ~4 chars per token heuristic for English text.
func EstimateTokens(content []byte) int {
	return len(content) / 4
}

// CalculateContextBudget measures all generated context files in the project
// and returns a budget report.
func CalculateContextBudget(projectRoot, modelSize string) (ContextBudget, error) {
	model := ResolveModelSize(modelSize)
	budget := ContextBudget{
		ModelSize: model,
		MaxTokens: MaxTokensForModel(model),
	}

	// Measure CLAUDE.md.
	claudeMd, err := os.ReadFile(filepath.Join(projectRoot, "CLAUDE.md"))
	if err == nil {
		budget.ClaudeMdTokens = EstimateTokens(claudeMd)
	}

	// Measure .claude/rules/*.md.
	rulesDir := filepath.Join(projectRoot, ".claude", "rules")
	entries, _ := os.ReadDir(rulesDir)
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(rulesDir, e.Name()))
		if readErr == nil {
			budget.RulesTokens += EstimateTokens(data)
		}
	}

	budget.TotalTokens = budget.ClaudeMdTokens + budget.RulesTokens + budget.SkillDescTokens
	if budget.MaxTokens > 0 {
		budget.BudgetPct = float64(budget.TotalTokens) / float64(budget.MaxTokens) * 100
	}

	return budget, nil
}

// Validate checks whether the context budget is within the 5% threshold.
func (b ContextBudget) Validate() error {
	if b.BudgetPct > 5.0 {
		return fmt.Errorf("context budget %.1f%% exceeds 5%% threshold (model: %s, %d/%d tokens)",
			b.BudgetPct, b.ModelSize, b.TotalTokens, b.MaxTokens)
	}
	return nil
}

// FormatReport writes a human-readable budget report to the given writer.
func (b ContextBudget) FormatReport(w io.Writer) {
	fmt.Fprintf(w, "Context Budget Report (model: %s, window: %d tokens)\n", b.ModelSize, b.MaxTokens)
	fmt.Fprintf(w, "  CLAUDE.md:     %6d tokens\n", b.ClaudeMdTokens)
	fmt.Fprintf(w, "  Rules:         %6d tokens\n", b.RulesTokens)
	fmt.Fprintf(w, "  Skill descs:   %6d tokens\n", b.SkillDescTokens)
	fmt.Fprintf(w, "  Total:         %6d tokens (%.1f%%)\n", b.TotalTokens, b.BudgetPct)
	if b.BudgetPct > 5.0 {
		fmt.Fprintf(w, "  WARNING: OVER BUDGET (target: <5%%)\n")
	} else {
		fmt.Fprintf(w, "  OK: Within budget\n")
	}
}
