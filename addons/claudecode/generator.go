package claudecode

import (
	"fmt"

	"fastcat.org/go/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"fastcat.org/go/gdev-secure-devenv-bootstrap/pkg/types"
)

// Compile-time interface check.
var _ types.Generator = (*ClaudeCodeGenerator)(nil)

// ClaudeCodeGenerator orchestrates all Claude Code sub-generators to produce
// the complete set of files for a security-hardened Claude Code configuration.
type ClaudeCodeGenerator struct {
	registry *ecosystem.Registry
	cfg      Config
}

// NewClaudeCodeGenerator creates a ClaudeCodeGenerator backed by the given
// ecosystem module registry and addon configuration.
func NewClaudeCodeGenerator(registry *ecosystem.Registry, cfg Config) *ClaudeCodeGenerator {
	return &ClaudeCodeGenerator{registry: registry, cfg: cfg}
}

// Generate produces the full set of generated files from wizard answers:
//  1. .claude/settings.json — permission rules, deny rules, hooks
//  2. CLAUDE.md             — project documentation for Claude
//  3. .claude/hooks/package-guard.py — PreToolUse guard hook (when enabled)
//  4. .claude/skills/*.md   — selected skill files
//  5. .claude/rules/*.md    — convention rule files based on languages
//  6. .mcp.json             — MCP server configuration (when servers configured)
func (g *ClaudeCodeGenerator) Generate(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	var files []types.GeneratedFile

	// 1. settings.json
	settingsFile, err := GenerateSettings(answers, g.registry, g.cfg)
	if err != nil {
		return nil, fmt.Errorf("generating settings: %w", err)
	}
	if settingsFile != nil {
		files = append(files, *settingsFile)
	}

	// 2. CLAUDE.md
	claudeMdFile, err := GenerateClaudeMd(answers, g.registry)
	if err != nil {
		return nil, fmt.Errorf("generating CLAUDE.md: %w", err)
	}
	if claudeMdFile != nil {
		files = append(files, *claudeMdFile)
	}

	// 3. Package guard hook (only when SafetyBlock is enabled)
	hookFile, err := GeneratePackageGuardHook(answers)
	if err != nil {
		return nil, fmt.Errorf("generating package guard hook: %w", err)
	}
	if hookFile != nil {
		files = append(files, *hookFile)
	}

	// 4. Skills
	skillFiles, err := deploySkills(answers)
	if err != nil {
		return nil, fmt.Errorf("generating skills: %w", err)
	}
	files = append(files, skillFiles...)

	// 5. Rules
	ruleFiles, err := deployRules(answers)
	if err != nil {
		return nil, fmt.Errorf("generating rules: %w", err)
	}
	files = append(files, ruleFiles...)

	// 6. MCP config (only when MCP servers are configured)
	mcpFile, err := GenerateMcpJson(answers, g.cfg)
	if err != nil {
		return nil, fmt.Errorf("generating MCP config: %w", err)
	}
	if mcpFile != nil {
		files = append(files, *mcpFile)
	}

	return files, nil
}
