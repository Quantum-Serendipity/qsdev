package claudecode

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
	"github.com/Quantum-Serendipity/qsdev/internal/tier"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
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

// resolveTier determines the effective tier from wizard answers, falling back
// to inference from legacy fields when the explicit tier is not set.
func resolveTier(answers types.WizardAnswers) tier.Tier {
	return tier.Resolve(answers.Tier, answers.PermissionLevel, answers.MCPServers)
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
	t := resolveTier(answers)

	// 1. settings.json (all tiers)
	settingsFile, err := GenerateSettings(answers, g.registry, g.cfg)
	if err != nil {
		return nil, fmt.Errorf("generating settings: %w", err)
	}
	if settingsFile != nil {
		files = append(files, *settingsFile)
	}

	// 2. Hook files (all tiers)
	hookFiles, err := GenerateHookFiles(answers)
	if err != nil {
		return nil, fmt.Errorf("generating hook files: %w", err)
	}
	files = append(files, hookFiles...)

	// Gate 1: tier >= Standard for CLAUDE.md, skills, rules
	if t < tier.Standard {
		return files, nil
	}

	// 3. CLAUDE.md
	claudeMdFile, err := GenerateClaudeMd(answers, g.registry)
	if err != nil {
		return nil, fmt.Errorf("generating CLAUDE.md: %w", err)
	}
	if claudeMdFile != nil {
		files = append(files, *claudeMdFile)
	}

	// 4. Rules
	ruleFiles, err := deployRules(answers)
	if err != nil {
		return nil, fmt.Errorf("generating rules: %w", err)
	}
	files = append(files, ruleFiles...)

	// 4b. AlwaysOn tool configs (Standard+): security tools that must be
	// present regardless of tier.
	reg := toolreg.DefaultRegistry()
	for _, tool := range reg.All() {
		if tool.Default != toolreg.AlwaysOn {
			continue
		}
		if tool.GenerateFunc == nil {
			continue
		}
		toolFiles, err := tool.GenerateFunc(answers)
		if err != nil {
			return nil, fmt.Errorf("generating %s files: %w", tool.Name, err)
		}
		for i := range toolFiles {
			toolFiles[i].Owner = tool.Name
		}
		files = append(files, toolFiles...)
	}

	// Gate 2: tier >= Full for MCP, agents, skills, workflows
	if t < tier.Full {
		return files, nil
	}

	// 5. Skills
	skillFiles, err := deploySkills(answers)
	if err != nil {
		return nil, fmt.Errorf("generating skills: %w", err)
	}
	files = append(files, skillFiles...)

	// 6. Inject agent-tool MCP servers when enabled.
	if answers.AgentTools.PostmortemEnabled {
		if !sliceutil.Contains(answers.MCPServers, "agent-postmortem") {
			answers.MCPServers = append(answers.MCPServers, "agent-postmortem")
		}
	}
	if answers.AgentTools.VersionSentinel {
		if !sliceutil.Contains(answers.MCPServers, "version-sentinel") {
			answers.MCPServers = append(answers.MCPServers, "version-sentinel")
		}
	}

	var sembleOverride *MCPServerConfig
	if answers.AgentTools.SembleEnabled {
		sr, err := generateSembleConfig(answers)
		if err != nil {
			return nil, fmt.Errorf("generating semble config: %w", err)
		}
		if sr != nil {
			for _, name := range sr.MCPServers {
				if !sliceutil.Contains(answers.MCPServers, name) {
					answers.MCPServers = append(answers.MCPServers, name)
				}
			}
			for i := range sr.Files {
				sr.Files[i].Owner = "semble"
			}
			files = append(files, sr.Files...)
			sembleOverride = sr.Override
		}
	}

	// 7. MCP config
	mcpCfg := g.cfg
	if sembleOverride != nil {
		mcpCfg.MCPServers = append(append([]MCPServerConfig{}, mcpCfg.MCPServers...), *sembleOverride)
	}
	mcpFile, err := GenerateMcpJson(answers, mcpCfg)
	if err != nil {
		return nil, fmt.Errorf("generating MCP config: %w", err)
	}
	if mcpFile != nil {
		files = append(files, *mcpFile)
	}

	// 8. Agent postmortem skill
	if answers.AgentTools.PostmortemEnabled {
		postmortemFile, err := generatePostmortemSkill(answers, g.registry)
		if err != nil {
			return nil, fmt.Errorf("generating postmortem skill: %w", err)
		}
		if postmortemFile != nil {
			postmortemFile.Owner = "agent-postmortem"
			files = append(files, *postmortemFile)
		}
	}

	// 9. Version-Sentinel config
	if answers.AgentTools.VersionSentinel {
		vsFiles, err := generateVersionSentinelFiles(answers, g.registry)
		if err != nil {
			return nil, fmt.Errorf("generating version-sentinel config: %w", err)
		}
		for i := range vsFiles {
			vsFiles[i].Owner = "version-sentinel"
		}
		files = append(files, vsFiles...)
	}

	// 10. qsdev operation skills
	qsdevOpsFiles, err := deployOperationSkills(answers)
	if err != nil {
		return nil, fmt.Errorf("generating qsdev-ops skills: %w", err)
	}
	files = append(files, qsdevOpsFiles...)

	// 12. Consulting workflow agents
	agentFiles, err := deployAgents(answers)
	if err != nil {
		return nil, fmt.Errorf("generating consulting agents: %w", err)
	}
	files = append(files, agentFiles...)

	// 13. Consulting workflow skills
	workflowFiles, err := deployWorkflowSkills(answers, g.registry)
	if err != nil {
		return nil, fmt.Errorf("generating workflow skills: %w", err)
	}
	files = append(files, workflowFiles...)

	// 14. qsdev reference doc
	refFile, err := GenerateQsdevReference(answers, g.registry)
	if err != nil {
		return nil, fmt.Errorf("generating qsdev reference: %w", err)
	}
	if refFile != nil {
		files = append(files, *refFile)
	}

	// 15. Tool generation: explicitly enabled (opt-in) tools.
	// Derive already-generated set from Owner fields to avoid re-generating
	// tools handled by specialized code above.
	alreadyGenerated := make(map[string]bool, len(files))
	for _, f := range files {
		if f.Owner != "" {
			alreadyGenerated[f.Owner] = true
		}
	}
	for _, tool := range reg.All() {
		if alreadyGenerated[tool.Name] {
			continue
		}
		if tool.Default == toolreg.AlwaysOn {
			continue
		}
		if strings.HasPrefix(tool.Name, "consulting-agent-") ||
			strings.HasPrefix(tool.Name, "consulting-workflow-") {
			continue
		}
		if tool.GenerateFunc == nil {
			continue
		}
		if !answers.EnabledTools[tool.Name] {
			continue
		}
		toolFiles, err := tool.GenerateFunc(answers)
		if err != nil {
			return nil, fmt.Errorf("generating %s files: %w", tool.Name, err)
		}
		for i := range toolFiles {
			toolFiles[i].Owner = tool.Name
		}
		files = append(files, toolFiles...)
	}

	return files, nil
}
