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

func toggleDisableFunc(field string) func(*types.WizardAnswers) {
	return func(a *types.WizardAnswers) {
		setToggle(a, field, false)
	}
}

// toggleFields maps each catalog toggle_field value to the WizardAnswers
// boolean it controls. The catalog bridge rejects any other value.
var toggleFields = map[string]func(*types.WizardAnswers) *bool{
	"hooks.safety_block":             func(a *types.WizardAnswers) *bool { return &a.Hooks.SafetyBlock },
	"agent_tools.postmortem_enabled": func(a *types.WizardAnswers) *bool { return &a.AgentTools.PostmortemEnabled },
	"agent_tools.version_sentinel":   func(a *types.WizardAnswers) *bool { return &a.AgentTools.VersionSentinel },
	"agent_tools.semble_enabled":     func(a *types.WizardAnswers) *bool { return &a.AgentTools.SembleEnabled },
}

func setToggle(a *types.WizardAnswers, field string, val bool) {
	if ptr, ok := toggleFields[field]; ok {
		*ptr(a) = val
	}
}
