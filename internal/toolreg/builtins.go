package toolreg

import (
	"github.com/Quantum-Serendipity/qsdev/internal/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/internal/sectools"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func init() {
	r := DefaultRegistry()
	for _, t := range builtinTools() {
		_ = r.Register(t)
	}
}

func builtinTools() []Tool {
	return []Tool{
		attachGuardTool(),
		agentPostmortemTool(),
		versionSentinelTool(),
		sembleTool(),
		trailOfBitsSkillsTool(),
		secretspecTool(),
		context7Tool(),
		githubMCPTool(),
		socketDevMCPTool(),
		postgresMCPTool(),
		changelogTool(),
		semgrepTool(),
		gitleaksTool(),
		containerSecurityTool(),
		licenseComplianceTool(),
		commitlintTool(),
	}
}

func attachGuardTool() Tool {
	return Tool{
		Name:        "attach-guard",
		DisplayName: "Package Install Guard",
		Category:    CategorySecurity,
		Description: "PreToolUse hook blocking unverified package installs with age-gating and vulnerability checks",
		Default:     AlwaysOn,
		OwnedFiles: []FileOwnership{
			{Path: ".claude/hooks/package-guard.py", Ownership: Exclusive},
			{Path: ".claude/settings.json", Ownership: Shared, SectionID: "attach-guard"},
			{Path: "CLAUDE.md", Ownership: Shared, SectionID: "attach-guard"},
		},
		EnableFunc: func(a *types.WizardAnswers) {
			a.Hooks.SafetyBlock = true
		},
		DisableFunc: func(a *types.WizardAnswers) {
			a.Hooks.SafetyBlock = false
		},
	}
}

func agentPostmortemTool() Tool {
	return Tool{
		Name:        "agent-postmortem",
		DisplayName: "Agent Postmortem Protocol",
		Category:    CategoryAIAgent,
		Description: "Verification skill requiring agents to validate changes against build/test/lint after implementation",
		Default:     AlwaysOn,
		OwnedFiles: []FileOwnership{
			{Path: ".claude/skills/agent-postmortem/SKILL.md", Ownership: Exclusive},
			{Path: "CLAUDE.md", Ownership: Shared, SectionID: "agent-postmortem"},
		},
		EnableFunc: func(a *types.WizardAnswers) {
			a.AgentTools.PostmortemEnabled = true
		},
		DisableFunc: func(a *types.WizardAnswers) {
			a.AgentTools.PostmortemEnabled = false
		},
	}
}

func versionSentinelTool() Tool {
	return Tool{
		Name:        "version-sentinel",
		DisplayName: "Version Sentinel",
		Category:    CategoryAIAgent,
		Description: "Guards dependency version changes in manifest files, requiring agents to justify bumps",
		Default:     AlwaysOn,
		OwnedFiles: []FileOwnership{
			{Path: ".claude/skills/version-sentinel/SKILL.md", Ownership: Exclusive},
			{Path: ".version-sentinel/ignore", Ownership: Exclusive},
			{Path: "CLAUDE.md", Ownership: Shared, SectionID: "version-sentinel"},
		},
		EnableFunc: func(a *types.WizardAnswers) {
			a.AgentTools.VersionSentinel = true
			if a.AgentTools.VersionSentinelHours == 0 {
				a.AgentTools.VersionSentinelHours = 24
			}
		},
		DisableFunc: func(a *types.WizardAnswers) {
			a.AgentTools.VersionSentinel = false
		},
	}
}

func sembleTool() Tool {
	return Tool{
		Name:        "semble",
		DisplayName: "Semble Semantic Code Search",
		Category:    CategoryAIAgent,
		Description: "Semantic code search via MCP server or sub-agent for codebase navigation",
		Default:     AlwaysOn,
		Prerequisites: nil,
		OwnedFiles: []FileOwnership{
			{Path: ".mcp.json", Ownership: Shared, SectionID: "semble"},
			{Path: ".claude/agents/semble-search.md", Ownership: Exclusive},
			{Path: "CLAUDE.md", Ownership: Shared, SectionID: "semble"},
		},
		EnableFunc: func(a *types.WizardAnswers) {
			a.AgentTools.SembleEnabled = true
			if a.AgentTools.SembleMode == "" {
				a.AgentTools.SembleMode = "mcp"
			}
		},
		DisableFunc: func(a *types.WizardAnswers) {
			a.AgentTools.SembleEnabled = false
			a.MCPServers = removeStr(a.MCPServers, "semble")
		},
	}
}

func trailOfBitsSkillsTool() Tool {
	return Tool{
		Name:        "trail-of-bits-skills",
		DisplayName: "Trail of Bits Security Skills",
		Category:    CategoryAIAgent,
		Description: "Security-focused code review skills based on Trail of Bits audit methodology",
		Default:     AlwaysOn,
		OwnedFiles: []FileOwnership{
			{Path: ".claude/skills/security-review.md", Ownership: Exclusive},
			{Path: "CLAUDE.md", Ownership: Shared, SectionID: "trail-of-bits-skills"},
		},
		EnableFunc: func(a *types.WizardAnswers) {
			if !containsStr(a.Skills, "security-review") {
				a.Skills = append(a.Skills, "security-review")
			}
		},
		DisableFunc: func(a *types.WizardAnswers) {
			a.Skills = removeStr(a.Skills, "security-review")
		},
	}
}

func secretspecTool() Tool {
	return Tool{
		Name:        "secretspec",
		DisplayName: "Secret Specification",
		Category:    CategoryDevEx,
		Description: "Generates secretspec.toml declaring required secrets for services and ecosystem modules",
		Default: OptIn,
		OwnedFiles: []FileOwnership{
			{Path: "secretspec.toml", Ownership: Exclusive},
			{Path: "devenv.nix", Ownership: Shared, SectionID: "secretspec"},
			{Path: "CLAUDE.md", Ownership: Shared, SectionID: "secretspec"},
		},
	}
}

func context7Tool() Tool {
	return Tool{
		Name:        "context7",
		DisplayName: "Context7 MCP",
		Category:    CategoryAIAgent,
		Description: "Context7 MCP server providing library documentation lookups for AI agents",
		Default:     AlwaysOn,
		OwnedFiles: []FileOwnership{
			{Path: ".mcp.json", Ownership: Shared, SectionID: "context7"},
			{Path: "CLAUDE.md", Ownership: Shared, SectionID: "context7"},
		},
		EnableFunc: func(a *types.WizardAnswers) {
			if !containsStr(a.MCPServers, "context7") {
				a.MCPServers = append(a.MCPServers, "context7")
			}
		},
		DisableFunc: func(a *types.WizardAnswers) {
			a.MCPServers = removeStr(a.MCPServers, "context7")
		},
	}
}

func githubMCPTool() Tool {
	return Tool{
		Name:        "github-mcp",
		DisplayName: "GitHub MCP",
		Category:    CategoryAIAgent,
		Description: "GitHub MCP server for repository and issue management by AI agents",
		Default:     AlwaysOn,
		OwnedFiles: []FileOwnership{
			{Path: ".mcp.json", Ownership: Shared, SectionID: "github-mcp"},
			{Path: "CLAUDE.md", Ownership: Shared, SectionID: "github-mcp"},
		},
		EnableFunc: func(a *types.WizardAnswers) {
			if !containsStr(a.MCPServers, "github") {
				a.MCPServers = append(a.MCPServers, "github")
			}
		},
		DisableFunc: func(a *types.WizardAnswers) {
			a.MCPServers = removeStr(a.MCPServers, "github")
		},
	}
}

func socketDevMCPTool() Tool {
	return Tool{
		Name:        "socket-dev-mcp",
		DisplayName: "Socket.dev MCP",
		Category:    CategoryAIAgent,
		Description: "Socket.dev MCP server for supply chain security analysis of dependencies",
		Default:     AlwaysOn,
		OwnedFiles: []FileOwnership{
			{Path: ".mcp.json", Ownership: Shared, SectionID: "socket-dev-mcp"},
			{Path: "CLAUDE.md", Ownership: Shared, SectionID: "socket-dev-mcp"},
		},
		EnableFunc: func(a *types.WizardAnswers) {
			if !containsStr(a.MCPServers, "socket") {
				a.MCPServers = append(a.MCPServers, "socket")
			}
		},
		DisableFunc: func(a *types.WizardAnswers) {
			a.MCPServers = removeStr(a.MCPServers, "socket")
		},
	}
}

func postgresMCPTool() Tool {
	// TODO: promote to OnWhenDetected once DetectedProject gains a HasPostgres
	// field (or equivalent service-presence detection).
	return Tool{
		Name:        "postgres-mcp",
		DisplayName: "PostgreSQL MCP",
		Category:    CategoryAIAgent,
		Description: "PostgreSQL MCP server for database schema inspection and query execution by AI agents",
		Default:     OptIn,
		OwnedFiles: []FileOwnership{
			{Path: ".mcp.json", Ownership: Shared, SectionID: "postgres-mcp"},
			{Path: "CLAUDE.md", Ownership: Shared, SectionID: "postgres-mcp"},
		},
		EnableFunc: func(a *types.WizardAnswers) {
			if !containsStr(a.MCPServers, "postgres") {
				a.MCPServers = append(a.MCPServers, "postgres")
			}
		},
		DisableFunc: func(a *types.WizardAnswers) {
			a.MCPServers = removeStr(a.MCPServers, "postgres")
		},
	}
}

func changelogTool() Tool {
	return Tool{
		Name:        "changelog",
		DisplayName: "git-cliff Changelog",
		Category:    CategoryDevEx,
		Description: "Generates cliff.toml configuration for git-cliff conventional changelog generation",
		Default:     OptIn,
		OwnedFiles: []FileOwnership{
			{Path: "cliff.toml", Ownership: Exclusive},
			{Path: "devenv.nix", Ownership: Shared, SectionID: "changelog"},
			{Path: "CLAUDE.md", Ownership: Shared, SectionID: "changelog"},
		},
	}
}

func semgrepTool() Tool {
	return Tool{
		Name:        "semgrep",
		DisplayName: "Semgrep SAST",
		Category:    CategorySecurity,
		Description: "Static analysis via Semgrep with ecosystem-aware rule sets",
		Default:     AlwaysOn,
		OwnedFiles: []FileOwnership{
			{Path: ".semgrep.yml", Ownership: Exclusive},
			{Path: "devenv.nix", Ownership: Shared, SectionID: "semgrep"},
			{Path: "CLAUDE.md", Ownership: Shared, SectionID: "semgrep"},
		},
		GenerateFunc: func(a types.WizardAnswers) ([]types.GeneratedFile, error) {
			f, err := sectools.GenerateSemgrepYml(a, ecosystem.DefaultRegistry())
			if err != nil {
				return nil, err
			}
			return []types.GeneratedFile{*f}, nil
		},
	}
}

func gitleaksTool() Tool {
	return Tool{
		Name:        "gitleaks",
		DisplayName: "Gitleaks Secret Scanner",
		Category:    CategorySecurity,
		Description: "Secret detection via Gitleaks with project-specific allowlists",
		Default:     AlwaysOn,
		OwnedFiles: []FileOwnership{
			{Path: ".gitleaks.toml", Ownership: Exclusive},
			{Path: "devenv.nix", Ownership: Shared, SectionID: "gitleaks"},
			{Path: "CLAUDE.md", Ownership: Shared, SectionID: "gitleaks"},
		},
		GenerateFunc: func(a types.WizardAnswers) ([]types.GeneratedFile, error) {
			f, err := sectools.GenerateGitleaksToml(a)
			if err != nil {
				return nil, err
			}
			return []types.GeneratedFile{*f}, nil
		},
	}
}

func containerSecurityTool() Tool {
	return Tool{
		Name:        "container-security",
		DisplayName: "Container Security (Grype + Cosign)",
		Category:    CategorySecurity,
		Description: "Container vulnerability scanning with Grype and image signing with Cosign",
		Default:     OnWhenDetected,
		DetectFunc: func(d types.DetectedProject) bool {
			return d.HasDockerfile
		},
		OwnedFiles: []FileOwnership{
			{Path: ".grype.yaml", Ownership: Exclusive},
			{Path: ".cosign/policy.yaml", Ownership: Exclusive},
			{Path: "devenv.nix", Ownership: Shared, SectionID: "container-security"},
			{Path: "CLAUDE.md", Ownership: Shared, SectionID: "container-security"},
		},
		GenerateFunc: func(a types.WizardAnswers) ([]types.GeneratedFile, error) {
			grype, err := sectools.GenerateGrypeYaml(a)
			if err != nil {
				return nil, err
			}
			cosign, err := sectools.GenerateCosignPolicy(a)
			if err != nil {
				return nil, err
			}
			return []types.GeneratedFile{*grype, *cosign}, nil
		},
	}
}

func licenseComplianceTool() Tool {
	return Tool{
		Name:        "license-compliance",
		DisplayName: "License Compliance (ScanCode)",
		Category:    CategorySecurity,
		Description: "License compliance checking via ScanCode with SPDX allowlist/blocklist policies",
		Default:     OptIn,
		OwnedFiles: []FileOwnership{
			{Path: ".scancode.yml", Ownership: Exclusive},
			{Path: ".license-exceptions.yml", Ownership: Exclusive},
			{Path: "CLAUDE.md", Ownership: Shared, SectionID: "license-compliance"},
		},
		GenerateFunc: func(a types.WizardAnswers) ([]types.GeneratedFile, error) {
			scancode, err := sectools.GenerateScancodeYml(a)
			if err != nil {
				return nil, err
			}
			exceptions, err := sectools.GenerateLicenseExceptionsYml()
			if err != nil {
				return nil, err
			}
			return []types.GeneratedFile{*scancode, *exceptions}, nil
		},
	}
}

func commitlintTool() Tool {
	return Tool{
		Name:        "commitlint",
		DisplayName: "Commitlint",
		Category:    CategoryDevEx,
		Description: "Conventional commit message linting via commitlint with standard type enforcement",
		Default:     OptIn,
		OwnedFiles: []FileOwnership{
			{Path: ".commitlintrc.yml", Ownership: Exclusive},
			{Path: "CLAUDE.md", Ownership: Shared, SectionID: "commitlint"},
		},
	}
}

func removeStr(ss []string, s string) []string {
	var result []string
	for _, v := range ss {
		if v != s {
			result = append(result, v)
		}
	}
	return result
}
