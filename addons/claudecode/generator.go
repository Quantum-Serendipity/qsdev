package claudecode

import (
	"fmt"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
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
//  7. .claude/skills/agent-postmortem/SKILL.md — postmortem verification skill
//  8. Version-Sentinel config (.version-sentinel/ignore, recovery skill)
//  9. Semble sub-agent (.claude/agents/semble-search.md)
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

	// 3. Hook files (package guard, audit log, etc.)
	hookFiles, err := GenerateHookFiles(answers)
	if err != nil {
		return nil, fmt.Errorf("generating hook files: %w", err)
	}
	files = append(files, hookFiles...)

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

	// 5b. Inject semble into MCP servers if enabled in MCP or both mode.
	if answers.AgentTools.SembleEnabled && (answers.AgentTools.SembleMode == "mcp" || answers.AgentTools.SembleMode == "both" || answers.AgentTools.SembleMode == "") {
		if !contains(answers.MCPServers, "semble") {
			answers.MCPServers = append(answers.MCPServers, "semble")
		}
	}

	// 6. MCP config (only when MCP servers are configured)
	mcpFile, err := GenerateMcpJson(answers, g.cfg)
	if err != nil {
		return nil, fmt.Errorf("generating MCP config: %w", err)
	}
	if mcpFile != nil {
		files = append(files, *mcpFile)
	}

	// 7. Agent postmortem skill
	if answers.AgentTools.PostmortemEnabled {
		postmortemFile, err := generatePostmortemSkill(answers, g.registry)
		if err != nil {
			return nil, fmt.Errorf("generating postmortem skill: %w", err)
		}
		if postmortemFile != nil {
			files = append(files, *postmortemFile)
		}
	}

	// 8. Version-Sentinel config
	if answers.AgentTools.VersionSentinel {
		vsFiles, err := generateVersionSentinelFiles(answers, g.registry)
		if err != nil {
			return nil, fmt.Errorf("generating version-sentinel config: %w", err)
		}
		files = append(files, vsFiles...)
	}

	// 9. Semble sub-agent (MCP handled in step 5b+6)
	if answers.AgentTools.SembleEnabled && (answers.AgentTools.SembleMode == "subagent" || answers.AgentTools.SembleMode == "both") {
		sembleFiles, err := generateSembleFiles(answers)
		if err != nil {
			return nil, fmt.Errorf("generating semble config: %w", err)
		}
		files = append(files, sembleFiles...)
	}

	return files, nil
}
