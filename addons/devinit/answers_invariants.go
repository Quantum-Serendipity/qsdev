package devinit

import (
	"maps"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/internal/tier"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// enforceAnswerInvariants applies the settings that no input path (flags,
// profile, answers file, wizard, join, update) may leave off. Every path that
// hands answers to the generators must call it last.
func enforceAnswerInvariants(a *types.WizardAnswers) {
	// The self-protection hook guards qsdev's own guardrails from the agent;
	// it is always on whenever Claude Code is configured.
	if a.ClaudeCode {
		a.Hooks.SelfProtection = true
	}
	// The tier is always recorded. Create paths resolve it in FillDefaults;
	// answers saved before that are legacy, so their tier is inferred once
	// here and then persisted (answers file and .qsdev.yaml) on save.
	if a.Tier == "" {
		a.Tier = tier.Infer(a.PermissionLevel, a.MCPServers).String()
	}
}

// cloneAnswers returns a copy of a that shares no slices or maps with it, so
// the copy can be filled in (for example by FillDefaults) without mutating a.
func cloneAnswers(a types.WizardAnswers) types.WizardAnswers {
	c := a
	c.Languages = slices.Clone(a.Languages)
	for i := range c.Languages {
		c.Languages[i].Extras = slices.Clone(a.Languages[i].Extras)
	}
	c.Services = slices.Clone(a.Services)
	for i := range c.Services {
		c.Services[i].Settings = maps.Clone(a.Services[i].Settings)
	}
	c.GitHooks = slices.Clone(a.GitHooks)
	c.ExtraPackages = slices.Clone(a.ExtraPackages)
	c.EnvVars = maps.Clone(a.EnvVars)
	c.Skills = slices.Clone(a.Skills)
	c.MCPServers = slices.Clone(a.MCPServers)
	c.EnabledTools = maps.Clone(a.EnabledTools)
	c.Overlays = slices.Clone(a.Overlays)
	return c
}
