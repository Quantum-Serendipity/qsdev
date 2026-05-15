package toolreg

import "github.com/Quantum-Serendipity/qsdev/pkg/types"

// consultingWorkflowNames lists all consulting workflow skill names in
// a stable order.
var consultingWorkflowNames = []string{
	"review-pr",
	"add-tests",
	"upgrade-dep",
	"onboard-me",
	"write-adr",
	"incident-debug",
	"migration-plan",
	"handoff-doc",
}

func init() {
	r := DefaultRegistry()
	for _, t := range consultingWorkflowTools() {
		_ = r.Register(t)
	}
}

func consultingWorkflowTools() []Tool {
	tools := []Tool{
		consultingWorkflowTool("review-pr", "PR Review (Consulting)", "Comprehensive PR review with security, performance, and quality analysis"),
		consultingWorkflowTool("add-tests", "Add Tests (Consulting)", "Analyze coverage gaps and generate tests following project patterns"),
		consultingWorkflowTool("upgrade-dep", "Upgrade Dependency (Consulting)", "Plan and execute dependency upgrades with verification"),
		consultingWorkflowTool("onboard-me", "Codebase Onboarding (Consulting)", "Systematic codebase onboarding via exploration agent"),
		consultingWorkflowTool("write-adr", "Write ADR (Consulting)", "Generate Architecture Decision Records in MADR format"),
		consultingWorkflowTool("incident-debug", "Incident Debug (Consulting)", "Systematic incident debugging with hypothesis testing"),
		consultingWorkflowTool("migration-plan", "Migration Plan (Consulting)", "Phased migration planning with risk assessment"),
		consultingWorkflowTool("handoff-doc", "Handoff Doc (Consulting)", "Client handoff documentation synthesis"),
	}

	// Special case: review-pr supersedes the basic review-pr skill
	// from the standard skill set.
	for i, t := range tools {
		if t.Name == "consulting-workflow-review-pr" {
			origEnable := tools[i].EnableFunc
			tools[i].EnableFunc = func(a *types.WizardAnswers) {
				if origEnable != nil {
					origEnable(a)
				}
				// Supersede the basic review-pr skill.
				a.Skills = removeStr(a.Skills, "review-pr")
			}
		}
	}

	return tools
}

func consultingWorkflowTool(name, displayName, description string) Tool {
	toolKey := "consulting-workflow-" + name
	return Tool{
		Name:        toolKey,
		DisplayName: displayName,
		Category:    CategoryAIAgent,
		Description: description,
		Default:     OptIn,
		OwnedFiles: []FileOwnership{
			{Path: ".claude/skills/" + name + "/SKILL.md", Ownership: Exclusive},
		},
		EnableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools[toolKey] = true
		},
		DisableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools != nil {
				delete(a.EnabledTools, toolKey)
			}
		},
	}
}
