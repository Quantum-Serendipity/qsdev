package claudecode

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
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
	ModelSize string
	MaxTokens int
	// ClaudeMdTokens covers CLAUDE.md plus every project file it pulls in via
	// @imports (followed recursively, as Claude Code does).
	ClaudeMdTokens int
	// RulesTokens covers every .md file under .claude/rules/, recursively.
	RulesTokens int
	// SkillDescTokens covers the name and description front matter of each
	// .claude/skills/<name>/SKILL.md, which Claude Code loads up front.
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
// and returns a budget report. Missing files count as zero; any other I/O
// error is returned, since an unmeasured file would understate the budget.
func CalculateContextBudget(projectRoot, modelSize string) (ContextBudget, error) {
	model := ResolveModelSize(modelSize)
	budget := ContextBudget{
		ModelSize: model,
		MaxTokens: MaxTokensForModel(model),
	}

	var err error
	if budget.ClaudeMdTokens, err = memoryFileTokens(projectRoot, filepath.Join(projectRoot, "CLAUDE.md")); err != nil {
		return ContextBudget{}, err
	}
	if budget.RulesTokens, err = rulesTokens(filepath.Join(projectRoot, ".claude", "rules")); err != nil {
		return ContextBudget{}, err
	}
	if budget.SkillDescTokens, err = skillDescTokens(filepath.Join(projectRoot, ".claude", "skills")); err != nil {
		return ContextBudget{}, err
	}

	budget.TotalTokens = budget.ClaudeMdTokens + budget.RulesTokens + budget.SkillDescTokens
	if budget.MaxTokens > 0 {
		budget.BudgetPct = float64(budget.TotalTokens) / float64(budget.MaxTokens) * 100
	}

	return budget, nil
}

// maxImportDepth is how many @import hops Claude Code follows from CLAUDE.md.
const maxImportDepth = 5

var (
	// importRef matches an @path reference at the start of a line or after
	// whitespace (so e-mail addresses are not imports).
	importRef = regexp.MustCompile(`(?:^|\s)@(\S+)`)
	// inlineCode matches a markdown code span; imports inside it are ignored.
	inlineCode = regexp.MustCompile("`[^`]*`")
)

// memoryFileTokens estimates the tokens of a memory file plus the project files
// it imports, recursively up to maxImportDepth. A missing top-level file counts
// as zero.
func memoryFileTokens(projectRoot, path string) (int, error) {
	return importTokens(projectRoot, path, 0, map[string]bool{})
}

func importTokens(projectRoot, path string, depth int, seen map[string]bool) (int, error) {
	seen[path] = true
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("reading %s: %w", path, err)
	}
	tokens := EstimateTokens(data)
	if depth >= maxImportDepth {
		return tokens, nil
	}
	for _, ref := range parseImports(data) {
		target, ok := resolveImport(projectRoot, filepath.Dir(path), ref)
		if !ok || seen[target] {
			continue
		}
		t, err := importTokens(projectRoot, target, depth+1, seen)
		if err != nil {
			return 0, err
		}
		tokens += t
	}
	return tokens, nil
}

// parseImports returns the @path references in markdown content, skipping
// fenced code blocks and inline code spans.
func parseImports(data []byte) []string {
	var refs []string
	inFence := false
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		for _, m := range importRef.FindAllStringSubmatch(inlineCode.ReplaceAllString(line, ""), -1) {
			refs = append(refs, strings.TrimRight(m[1], ".,;:!?)"))
		}
	}
	return refs
}

// resolveImport resolves an @import reference relative to the importing
// file's directory. Only regular files inside the project count: home (~/)
// and absolute imports are machine-specific, and a reference that names no
// file (e.g. an npm scope such as @upstash/pkg in prose) is not an import.
func resolveImport(projectRoot, dir, ref string) (string, bool) {
	if ref == "" || strings.HasPrefix(ref, "~") || filepath.IsAbs(ref) {
		return "", false
	}
	target := filepath.Join(dir, filepath.FromSlash(ref))
	rel, err := filepath.Rel(projectRoot, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", false
	}
	info, err := os.Stat(target)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	return target, true
}

// rulesTokens sums every .md file under the rules directory, recursively.
func rulesTokens(rulesDir string) (int, error) {
	total := 0
	err := filepath.WalkDir(rulesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) && path == rulesDir {
				return fs.SkipAll
			}
			return err
		}
		if d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".md") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		total += EstimateTokens(data)
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("measuring rules in %s: %w", rulesDir, err)
	}
	return total, nil
}

// skillDescTokens sums the name and description front matter of every
// <skillsDir>/<name>/SKILL.md.
func skillDescTokens(skillsDir string) (int, error) {
	entries, err := os.ReadDir(skillsDir)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("reading skills directory %s: %w", skillsDir, err)
	}
	total := 0
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(skillsDir, e.Name(), "SKILL.md"))
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}
		if err != nil {
			return 0, fmt.Errorf("reading skill %s: %w", e.Name(), err)
		}
		total += EstimateTokens(skillDescription(data))
	}
	return total, nil
}

// skillDescription returns the name and description a SKILL.md front matter
// contributes to the context. Unparseable front matter is counted whole, so a
// malformed skill never shrinks the estimate.
func skillDescription(data []byte) []byte {
	fm, ok := frontMatter(data)
	if !ok {
		return nil
	}
	var meta struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	if err := yaml.Unmarshal(fm, &meta); err != nil {
		return fm
	}
	return []byte(meta.Name + "\n" + meta.Description)
}

// frontMatter extracts the YAML between a leading "---" line and the next one.
func frontMatter(data []byte) ([]byte, bool) {
	rest, ok := bytes.CutPrefix(bytes.TrimLeft(data, " \t\r\n"), []byte("---"))
	if !ok {
		return nil, false
	}
	body, _, found := bytes.Cut(rest, []byte("\n---"))
	if !found {
		return nil, false
	}
	return body, true
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
