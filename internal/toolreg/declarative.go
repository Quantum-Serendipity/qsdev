package toolreg

import (
	"slices"

	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func mcpEnableFunc(serverName string) func(*types.WizardAnswers) {
	return func(a *types.WizardAnswers) {
		if !slices.Contains(a.MCPServers, serverName) {
			a.MCPServers = append(a.MCPServers, serverName)
		}
	}
}

func mcpDisableFunc(serverName string) func(*types.WizardAnswers) {
	return func(a *types.WizardAnswers) {
		a.MCPServers = sliceutil.Remove(a.MCPServers, serverName)
	}
}

func skillEnableFunc(skillName string) func(*types.WizardAnswers) {
	return func(a *types.WizardAnswers) {
		if !slices.Contains(a.Skills, skillName) {
			a.Skills = append(a.Skills, skillName)
		}
	}
}

// skillForceOnFunc returns a ForceOnFunc that adds the skill. A skill missing
// from the list is not an explicit off, so it never reports an override.
func skillForceOnFunc(skillName string) ForceOnFunc {
	enable := skillEnableFunc(skillName)
	return func(a *types.WizardAnswers) bool {
		enable(a)
		return false
	}
}

func skillDisableFunc(skillName string) func(*types.WizardAnswers) {
	return func(a *types.WizardAnswers) {
		a.Skills = sliceutil.Remove(a.Skills, skillName)
	}
}

func toggleEnableFunc(field string) func(*types.WizardAnswers) {
	return func(a *types.WizardAnswers) {
		setToggle(a, field, true)
	}
}

// toggleForceOnFunc returns a ForceOnFunc that turns the toggle on and
// reports whether it was off.
func toggleForceOnFunc(field string) ForceOnFunc {
	return func(a *types.WizardAnswers) bool {
		acc, ok := toggleFields[field]
		if !ok || acc.get(a) {
			return false
		}
		acc.set(a, true)
		return true
	}
}

func toggleDisableFunc(field string) func(*types.WizardAnswers) {
	return func(a *types.WizardAnswers) {
		setToggle(a, field, false)
	}
}

// toggleAccessor reads and writes the WizardAnswers state behind one catalog
// toggle_field.
type toggleAccessor struct {
	get func(*types.WizardAnswers) bool
	set func(*types.WizardAnswers, bool)
}

// boolField returns the accessor for a toggle that is a plain boolean field.
func boolField(ptr func(*types.WizardAnswers) *bool) toggleAccessor {
	return toggleAccessor{
		get: func(a *types.WizardAnswers) bool { return *ptr(a) },
		set: func(a *types.WizardAnswers, v bool) { *ptr(a) = v },
	}
}

// toggleFields maps each catalog toggle_field value to the WizardAnswers
// state it controls. The catalog bridge rejects any other value. The safety
// block goes through HookChoices.SetSafetyBlock, so switching it off records
// the opt-out and switching it on clears it; it counts as on only when no
// opt-out is recorded.
var toggleFields = map[string]toggleAccessor{
	"hooks.safety_block": {
		get: func(a *types.WizardAnswers) bool { return a.Hooks.SafetyBlock && !a.Hooks.SafetyBlockOptOut },
		set: func(a *types.WizardAnswers, v bool) { a.Hooks.SetSafetyBlock(v) },
	},
	"agent_tools.postmortem_enabled": boolField(func(a *types.WizardAnswers) *bool { return &a.AgentTools.PostmortemEnabled }),
	"agent_tools.version_sentinel":   boolField(func(a *types.WizardAnswers) *bool { return &a.AgentTools.VersionSentinel }),
	"agent_tools.semble_enabled":     boolField(func(a *types.WizardAnswers) *bool { return &a.AgentTools.SembleEnabled }),
}

func setToggle(a *types.WizardAnswers, field string, val bool) {
	if acc, ok := toggleFields[field]; ok {
		acc.set(a, val)
	}
}
