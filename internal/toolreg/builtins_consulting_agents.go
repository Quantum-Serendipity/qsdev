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
		n := toolName
		r.AttachBehavior(n, ToolBehavior{
			EnableFunc: func(a *types.WizardAnswers) {
				if a.EnabledTools == nil {
					a.EnabledTools = make(map[string]bool)
				}
				a.EnabledTools[n] = true
			},
			DisableFunc: func(a *types.WizardAnswers) {
				if a.EnabledTools != nil {
					a.EnabledTools[n] = false
				}
			},
		})
	}
}
