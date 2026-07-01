package claudecode

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/tier"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/lsp"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface check.
var _ types.Generator = (*ClaudeCodeGenerator)(nil)

// ClaudeCodeGenerator orchestrates all Claude Code sub-generators to produce
// the complete set of files for a security-hardened Claude Code configuration.
type ClaudeCodeGenerator struct {
	registry    *ecosystem.Registry
	cfg         Config
	lspRegistry *lsp.LSPRegistry
}

// NewClaudeCodeGenerator creates a ClaudeCodeGenerator backed by the given
// ecosystem module registry and addon configuration. The LSP registry is pure
// static data, so it is built internally rather than injected.
func NewClaudeCodeGenerator(registry *ecosystem.Registry, cfg Config) *ClaudeCodeGenerator {
	return &ClaudeCodeGenerator{registry: registry, cfg: cfg, lspRegistry: lsp.NewRegistry()}
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

	// Reconcile the claudecode Config LSP enforcement override into answers so
	// downstream hook generation observes it. answers is a value parameter, so
	// this mutates only the local copy used by the rest of Generate.
	if g.cfg.LSPEnforcement != "" {
		answers.LSP.Enforcement = g.cfg.LSPEnforcement
	}

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

	// 4a. Consolidated LSP plugin (Standard+): auto-loads LSP servers for the
	// detected ecosystems (nixd always included).
	lspFiles, err := GenerateLspPlugin(answers, g.lspRegistry)
	if err != nil {
		return nil, fmt.Errorf("generating LSP plugin: %w", err)
	}
	files = append(files, lspFiles...)

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

	// 5. Skills (Standard+): deploying an explicitly requested skill is a
	// deliberate choice and must not be gated behind Full. deploySkills
	// self-guards on an empty Skills list, so this is a no-op when none are
	// configured.
	skillFiles, err := deploySkills(answers)
	if err != nil {
		return nil, fmt.Errorf("generating skills: %w", err)
	}
	files = append(files, skillFiles...)

	// 6/7. MCP (Standard+ when servers are configured; always at Full, where
	// enabled tools may auto-inject their own servers). The guard is a strict
	// superset of the previous Full-only behavior, so Full-tier output is
	// unchanged, while a standard-tier project that configured mcp_servers now
	// gets its .mcp.json instead of silently nothing.
	if t >= tier.Full || len(answers.MCPServers) > 0 {
		// 6. Auto-inject MCP servers for enabled tools that declare mcp_server_name.
		cat := catalog.MustDefault()
		for name, def := range cat.Tools() {
			if def.MCPServerName == "" {
				continue
			}
			if !answers.EnabledTools[name] {
				continue
			}
			if !slices.Contains(answers.MCPServers, def.MCPServerName) {
				answers.MCPServers = append(answers.MCPServers, def.MCPServerName)
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
					if !slices.Contains(answers.MCPServers, name) {
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
	}

	// Gate 2: tier >= Full for consulting agents/workflows, operation skills,
	// and the qsdev reference doc.
	if t < tier.Full {
		return dedupeFilesByPath(files), nil
	}

	// 8. qsdev operation skills
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

	return dedupeFilesByPath(files), nil
}

// dedupeFilesByPath drops duplicate GeneratedFile entries that target the same
// path, keeping the LAST occurrence. Later deployers intentionally supersede
// earlier ones for the same path — e.g. review-pr is emitted both as a library
// skill (deploySkills) and, at Full tier, as the richer consulting workflow
// skill (deployWorkflowSkills); keeping the last occurrence lets the workflow
// version win.
func dedupeFilesByPath(files []types.GeneratedFile) []types.GeneratedFile {
	lastIdx := make(map[string]int, len(files))
	for i, f := range files {
		lastIdx[f.Path] = i
	}
	out := make([]types.GeneratedFile, 0, len(files))
	for i, f := range files {
		if lastIdx[f.Path] == i {
			out = append(out, f)
		}
	}
	return out
}

// SuppressedConfigWarnings returns human-readable warnings for skills or MCP
// servers that the resolved tier will NOT emit, so callers can avoid reporting
// an unqualified success. After the standard-tier fix (skills + configured MCP
// now emit at Standard), suppression only happens below Standard.
func SuppressedConfigWarnings(answers types.WizardAnswers) []string {
	if resolveTier(answers) >= tier.Standard {
		return nil
	}
	var warnings []string
	if len(answers.Skills) > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"%d configured skill(s) will not be generated below the standard tier; raise the tier to standard or higher",
			len(answers.Skills)))
	}
	if len(answers.MCPServers) > 0 {
		warnings = append(warnings, fmt.Sprintf(
			"%d configured MCP server(s) will not be generated below the standard tier; raise the tier to standard or higher",
			len(answers.MCPServers)))
	}
	return warnings
}
