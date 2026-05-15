package toolreg

import "github.com/Quantum-Serendipity/qsdev/pkg/types"

func init() {
	r := DefaultRegistry()
	for _, t := range consultingAgentTools() {
		_ = r.Register(t)
	}
}

func consultingAgentTools() []Tool {
	return []Tool{
		consultingAgentTool("security-reviewer", "Security Reviewer Agent", "Deep security analysis with OWASP methodology"),
		consultingAgentTool("codebase-explorer", "Codebase Explorer Agent", "Fast codebase navigation and architecture analysis"),
		consultingAgentTool("test-gap-analyzer", "Test Gap Analyzer Agent", "Identify untested code paths and coverage gaps"),
		consultingAgentTool("onboarding-guide", "Onboarding Guide Agent", "Interactive codebase mentoring for new engineers"),
		consultingAgentTool("migration-planner", "Migration Planner Agent", "Framework upgrade risk assessment and planning"),
		consultingAgentTool("handoff-doc-generator", "Handoff Doc Generator Agent", "Client handoff documentation synthesis"),
		consultingAgentTool("incident-debugger", "Incident Debugger Agent", "Systematic incident debugging with hypothesis testing"),
	}
}

func consultingAgentTool(name, displayName, description string) Tool {
	return Tool{
		Name:        "consulting-agent-" + name,
		DisplayName: displayName,
		Category:    CategoryAIAgent,
		Description: description,
		Default:     OptIn,
		OwnedFiles: []FileOwnership{
			{Path: ".claude/agents/" + name + ".md", Ownership: Exclusive},
		},
		EnableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools["consulting-agent-"+name] = true
		},
		DisableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools != nil {
				a.EnabledTools["consulting-agent-"+name] = false
			}
		},
	}
}
