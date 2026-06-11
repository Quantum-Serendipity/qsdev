package toolreg

import (
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func init() {
	r := DefaultRegistry()
	for _, t := range qsdevOpsTools() {
		_ = r.Register(t)
	}
}

func qsdevOpsTools() []Tool {
	app := branding.Get().AppName
	return []Tool{
		qsdevSkillTool(app+"-init", app+" init", "Initialize "+app+" project configuration with security-hardened defaults"),
		qsdevSkillTool(app+"-onboard", app+" onboard", "Onboard existing project to "+app+" with non-destructive merge"),
		qsdevSkillTool(app+"-setup", app+" setup", "Install missing prerequisites detected by "+app+" doctor"),
		qsdevSkillTool(app+"-enable", app+" enable", "Enable a tool in the "+app+" project configuration"),
		qsdevSkillTool(app+"-disable", app+" disable", "Disable a tool from the "+app+" project configuration"),
		qsdevSkillTool(app+"-update", app+" update", "Update "+app+"-managed configuration to latest templates"),
		qsdevSkillTool(app+"-doctor", app+" doctor", "Run "+app+" health diagnostics and analyze results"),
		qsdevSkillTool(app+"-status", app+" status", "Show "+app+" configuration status and security posture"),
		qsdevSkillTool(app+"-tools", app+" tools", "List available "+app+" tools organized by category"),
		qsdevSkillTool(app+"-detect", app+" detect", "Detect project ecosystems, languages, and frameworks"),
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
			ensureEnabledTools(a)
			a.EnabledTools[name] = true
		},
		DisableFunc: func(a *types.WizardAnswers) {
			ensureEnabledTools(a)
			a.EnabledTools[name] = false
		},
	}
}
