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

	// Gate 2: tier >= Full for MCP, agents, skills, workflows, AlwaysOn tools
	if t < tier.Full {
		return files, nil
	}

	// 5. Skills
	skillFiles, err := deploySkills(answers)
	if err != nil {
		return nil, fmt.Errorf("generating skills: %w", err)
	}
	files = append(files, skillFiles...)

	// 6. Inject semble into MCP servers if enabled.
	if answers.AgentTools.SembleEnabled && (answers.AgentTools.SembleMode == "mcp" || answers.AgentTools.SembleMode == "both" || answers.AgentTools.SembleMode == "") {
		if !sliceutil.Contains(answers.MCPServers, "semble") {
			answers.MCPServers = append(answers.MCPServers, "semble")
		}
	}

	// 7. MCP config
	mcpFile, err := GenerateMcpJson(answers, g.cfg)
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

	// 10. Semble sub-agent
	if answers.AgentTools.SembleEnabled && (answers.AgentTools.SembleMode == "subagent" || answers.AgentTools.SembleMode == "both") {
		sembleFiles, err := generateSembleFiles(answers)
		if err != nil {
			return nil, fmt.Errorf("generating semble config: %w", err)
		}
		for i := range sembleFiles {
			sembleFiles[i].Owner = "semble"
		}
		files = append(files, sembleFiles...)
	}

	// 11. qsdev operation skills
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

	// 15. Tool generation: AlwaysOn + explicitly enabled tools.
	reg := toolreg.DefaultRegistry()
	alreadyHandled := map[string]bool{
		"attach-guard":         true,
		"agent-postmortem":     true,
		"version-sentinel":     true,
		"semble":               true,
		"trail-of-bits-skills": true,
	}
	for _, tool := range reg.All() {
		if alreadyHandled[tool.Name] {
			continue
		}
		if strings.HasPrefix(tool.Name, "consulting-agent-") ||
			strings.HasPrefix(tool.Name, "consulting-workflow-") {
			continue
		}
		if tool.GenerateFunc == nil {
			continue
		}
		shouldGenerate := tool.Default == toolreg.AlwaysOn || answers.EnabledTools[tool.Name]
		if !shouldGenerate {
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
