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

func setToggle(a *types.WizardAnswers, field string, val bool) {
	switch field {
	case "hooks.safety_block":
		a.Hooks.SafetyBlock = val
	case "agent_tools.postmortem_enabled":
		a.AgentTools.PostmortemEnabled = val
	case "agent_tools.version_sentinel":
		a.AgentTools.VersionSentinel = val
	case "agent_tools.semble_enabled":
		a.AgentTools.SembleEnabled = val
	}
}
