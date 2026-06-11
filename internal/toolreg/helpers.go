package toolreg

import "github.com/Quantum-Serendipity/qsdev/pkg/types"

func ensureEnabledTools(a *types.WizardAnswers) {
	if a.EnabledTools == nil {
		a.EnabledTools = make(map[string]bool)
	}
}
