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
	for _, name := range consultingWorkflowNames {
		toolKey := "consulting-workflow-" + name
		b := ToolBehavior{
			EnableFunc: func(a *types.WizardAnswers) {
				if a.EnabledTools == nil {
					a.EnabledTools = make(map[string]bool)
				}
				a.EnabledTools[toolKey] = true
			},
			DisableFunc: func(a *types.WizardAnswers) {
				if a.EnabledTools == nil {
					a.EnabledTools = make(map[string]bool)
				}
				a.EnabledTools[toolKey] = false
			},
		}

		if name == "review-pr" {
			innerEnable := b.EnableFunc
			b.EnableFunc = func(a *types.WizardAnswers) {
				if innerEnable != nil {
					innerEnable(a)
				}
				a.Skills = removeStr(a.Skills, "review-pr")
			}
		}

		r.AttachBehavior(toolKey, b)
	}
}
