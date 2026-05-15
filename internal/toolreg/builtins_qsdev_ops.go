package toolreg

import "github.com/Quantum-Serendipity/qsdev/pkg/types"

func init() {
	r := DefaultRegistry()
	for _, t := range qsdevOpsTools() {
		_ = r.Register(t)
	}
}

func qsdevOpsTools() []Tool {
	return []Tool{
		qsdevSkillTool("qsdev-init", "qsdev init", "Initialize qsdev project configuration with security-hardened defaults"),
		qsdevSkillTool("qsdev-onboard", "qsdev onboard", "Onboard existing project to qsdev with non-destructive merge"),
		qsdevSkillTool("qsdev-setup", "qsdev setup", "Install missing prerequisites detected by qsdev doctor"),
		qsdevSkillTool("qsdev-enable", "qsdev enable", "Enable a tool in the qsdev project configuration"),
		qsdevSkillTool("qsdev-disable", "qsdev disable", "Disable a tool from the qsdev project configuration"),
		qsdevSkillTool("qsdev-update", "qsdev update", "Update qsdev-managed configuration to latest templates"),
		qsdevSkillTool("qsdev-doctor", "qsdev doctor", "Run qsdev health diagnostics and analyze results"),
		qsdevSkillTool("qsdev-status", "qsdev status", "Show qsdev configuration status and security posture"),
		qsdevSkillTool("qsdev-tools", "qsdev tools", "List available qsdev tools organized by category"),
		qsdevSkillTool("qsdev-detect", "qsdev detect", "Detect project ecosystems, languages, and frameworks"),
	}
}

// qsdevSkillTool creates a Tool definition for a qsdev operation skill.
// All qsdev-ops skills are AlwaysOn and own a single exclusive SKILL.md file.
func qsdevSkillTool(name, displayName, description string) Tool {
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
