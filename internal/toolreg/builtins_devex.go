package toolreg

import (
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func init() {
	r := DefaultRegistry()

	r.AttachBehavior("changelog", ToolBehavior{
		EnableFunc: func(a *types.WizardAnswers) {
			ensureEnabledTools(a)
			a.EnabledTools["changelog"] = true
		},
		DisableFunc: func(a *types.WizardAnswers) {
			ensureEnabledTools(a)
			a.EnabledTools["changelog"] = false
		},
	})

	r.AttachBehavior("commitlint", ToolBehavior{
		EnableFunc: func(a *types.WizardAnswers) {
			ensureEnabledTools(a)
			a.EnabledTools["commitlint"] = true
		},
		DisableFunc: func(a *types.WizardAnswers) {
			ensureEnabledTools(a)
			a.EnabledTools["commitlint"] = false
		},
	})

	r.AttachBehavior("secretspec", ToolBehavior{
		EnableFunc: func(a *types.WizardAnswers) {
			ensureEnabledTools(a)
			a.EnabledTools["secretspec"] = true
		},
		DisableFunc: func(a *types.WizardAnswers) {
			ensureEnabledTools(a)
			a.EnabledTools["secretspec"] = false
		},
	})
}
