package toolreg

import (
	"io"
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
// enforced again wherever it applies, also when it does not apply now (Claude
// Code switched off, say), so the off never reaches tools.disabled; and each
// tool it lists that the answers do not explicitly enable is disabled (its
// DisableFunc switches off what it backs and records any opt-out, such as
// Hooks.SafetyBlockOptOut, that answers saved before that record existed
// lack). A nil
// committed means the project has no committed config (or none that loads),
// so no opt-out is recorded: like an empty tools block, every explicit off
// for an always-on tool is dropped. The answers file alone never opts out.
//
// It returns, sorted, the always-on tools kept enabled against an answer
// source, for callers to warn about.
func Reconcile(answers *types.WizardAnswers, reg *Registry, committed *types.ToolsConfig) []string {
	if committed == nil {
		committed = &types.ToolsConfig{}
	}
	kept := dropUncommittedOptOuts(answers, reg, committed.Disabled)
	adoptOptOuts(answers, reg, committed.Disabled)
	kept = append(kept, MergeInferredTools(answers, reg)...)
	slices.Sort(kept)
	return slices.Compact(kept)
}

// ReconcileAndWarn runs Reconcile and writes to w a warning for each
// always-on tool kept enabled: one whose explicit off was dropped or
// overridden, or, when there is a committed config, one that applies to a and
// that committed lists as neither enabled nor disabled, so the regeneration
// adds it back. A new project (nil committed) records nothing yet, so keeping
// its defaults draws no warning. It also warns when the committed opt-out
// keeps the package guard off (see WarnSafetyBlockOptOut). restored names
// tools the caller already found kept enabled against an earlier answer
// source (see SwitchedOff); each tool is warned about once. Every path that
// reconciles answers calls it, so they all warn alike.
func ReconcileAndWarn(w io.Writer, a *types.WizardAnswers, reg *Registry, committed *types.ToolsConfig, restored ...string) {
	kept := append(Reconcile(a, reg, committed), restored...)
	if committed != nil {
		for _, tool := range reg.All() {
			name := tool.Name
			if tool.EnforcedFor(a) && a.EnabledTools[name] &&
				!slices.Contains(committed.Enabled, name) && !slices.Contains(committed.Disabled, name) {
				kept = append(kept, name)
			}
		}
	}
	slices.Sort(kept)
	WarnAlwaysOnRestored(w, slices.Compact(kept))
	WarnSafetyBlockOptOut(w, a)
}

// dropUncommittedOptOuts deletes the explicit off of each always-on tool that
// is not in optOuts, so enforcement turns it back on wherever it applies, and
// returns the dropped names. It drops the off whether or not the tool applies
// to a now: scope (Tool.EnforcedFor) decides enforcement, not whether an
// opt-out may be recorded, so an off given while, say, Claude Code is switched
// off is not persisted into tools.disabled to read back later as a committed
// opt-out.
func dropUncommittedOptOuts(a *types.WizardAnswers, reg *Registry, optOuts []string) []string {
	var dropped []string
	for _, tool := range reg.All() {
		enabled, set := a.EnabledTools[tool.Name]
		if !set || enabled || tool.Default != AlwaysOn || slices.Contains(optOuts, tool.Name) {
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
