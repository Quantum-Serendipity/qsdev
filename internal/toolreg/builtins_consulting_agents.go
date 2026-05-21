package toolreg

import "github.com/Quantum-Serendipity/qsdev/pkg/types"

func init() {
	r := DefaultRegistry()
	for _, name := range []string{
		"security-reviewer", "codebase-explorer", "test-gap-analyzer",
		"onboarding-guide", "migration-planner", "handoff-doc-generator",
		"incident-debugger",
	} {
		toolName := "consulting-agent-" + name
		r.AttachBehavior(toolName, ToolBehavior{
			EnableFunc: func(a *types.WizardAnswers) {
				if a.EnabledTools == nil {
					a.EnabledTools = make(map[string]bool)
				}
				a.EnabledTools[toolName] = true
			},
			DisableFunc: func(a *types.WizardAnswers) {
				if a.EnabledTools == nil {
					a.EnabledTools = make(map[string]bool)
				}
				a.EnabledTools[toolName] = false
			},
		})
	}
}
