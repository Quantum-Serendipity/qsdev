package toolreg

import (
	"slices"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Reconcile settles the answers' enabled tools against committed, the tools
// block of the committed .qsdev.yaml, and then runs MergeInferredTools. Every
// path that regenerates from answers it did not just take from .qsdev.yaml
// (init, update, enable/disable, claude init) calls it, so they agree on what
// counts as an opt-out.
//
// committed.Disabled, which only `disable --force` writes, is the one record
// of an always-on opt-out: an explicit off for an always-on tool that it does
// not list (from a hand-edited answers file, say) is dropped so the tool is
// enforced again, and each tool it lists that the answers do not explicitly
// enable is disabled (its DisableFunc switches off what it backs and records
// any opt-out, such as Hooks.SafetyBlockOptOut, that answers saved before
// that record existed lack). A nil
// committed means the project has no committed config, so the answers are
// the only record and their explicit offs stand.
//
// It returns, sorted, the always-on tools kept enabled against an answer
// source, for callers to warn about.
func Reconcile(answers *types.WizardAnswers, reg *Registry, committed *types.ToolsConfig) []string {
	var kept []string
	if committed != nil {
		kept = dropUncommittedOptOuts(answers, reg, committed.Disabled)
		adoptOptOuts(answers, reg, committed.Disabled)
	}
	kept = append(kept, MergeInferredTools(answers, reg)...)
	slices.Sort(kept)
	return slices.Compact(kept)
}

// dropUncommittedOptOuts deletes the explicit off of each always-on tool that
// applies to a and is not in optOuts, so enforcement turns it back on, and
// returns the dropped names.
func dropUncommittedOptOuts(a *types.WizardAnswers, reg *Registry, optOuts []string) []string {
	var dropped []string
	for _, tool := range reg.All() {
		enabled, set := a.EnabledTools[tool.Name]
		if !set || enabled || !tool.EnforcedFor(a) || slices.Contains(optOuts, tool.Name) {
			continue
		}
		delete(a.EnabledTools, tool.Name)
		dropped = append(dropped, tool.Name)
	}
	return dropped
}

// adoptOptOuts disables each named tool the answers do not explicitly
// enable. DisableFunc is idempotent, so a tool the answers already recorded
// as off is disabled again, which migrates its backing to the current
// opt-out record.
func adoptOptOuts(a *types.WizardAnswers, reg *Registry, disabled []string) {
	if a.EnabledTools == nil {
		a.EnabledTools = make(map[string]bool)
	}
	for _, name := range disabled {
		if a.EnabledTools[name] {
			continue
		}
		if tool, ok := reg.ByName(name); ok && tool.DisableFunc != nil {
			tool.DisableFunc(a)
		}
		a.EnabledTools[name] = false
	}
}
