package toolreg

import "github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"

func init() {
	r := DefaultRegistry()
	for _, t := range gdevOpsTools() {
		_ = r.Register(t)
	}
}

func gdevOpsTools() []Tool {
	return []Tool{
		gdevSkillTool("gdev-init", "gdev init", "Initialize gdev project configuration with security-hardened defaults"),
		gdevSkillTool("gdev-onboard", "gdev onboard", "Onboard existing project to gdev with non-destructive merge"),
		gdevSkillTool("gdev-setup", "gdev setup", "Install missing prerequisites detected by gdev doctor"),
		gdevSkillTool("gdev-enable", "gdev enable", "Enable a tool in the gdev project configuration"),
		gdevSkillTool("gdev-disable", "gdev disable", "Disable a tool from the gdev project configuration"),
		gdevSkillTool("gdev-update", "gdev update", "Update gdev-managed configuration to latest templates"),
		gdevSkillTool("gdev-doctor", "gdev doctor", "Run gdev health diagnostics and analyze results"),
		gdevSkillTool("gdev-status", "gdev status", "Show gdev configuration status and security posture"),
		gdevSkillTool("gdev-tools", "gdev tools", "List available gdev tools organized by category"),
		gdevSkillTool("gdev-detect", "gdev detect", "Detect project ecosystems, languages, and frameworks"),
	}
}

// gdevSkillTool creates a Tool definition for a gdev operation skill.
// All gdev-ops skills are AlwaysOn and own a single exclusive SKILL.md file.
func gdevSkillTool(name, displayName, description string) Tool {
	return Tool{
		Name:        name,
		DisplayName: displayName,
		Category:    CategoryAIAgent,
		Description: description,
		Default:     AlwaysOn,
		OwnedFiles: []FileOwnership{
			{Path: ".claude/skills/" + name + "/SKILL.md", Ownership: Exclusive},
		},
		EnableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools[name] = true
		},
		DisableFunc: func(a *types.WizardAnswers) {
			if a.EnabledTools == nil {
				a.EnabledTools = make(map[string]bool)
			}
			a.EnabledTools[name] = false
		},
	}
}
