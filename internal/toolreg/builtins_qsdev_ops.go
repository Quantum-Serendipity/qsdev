package toolreg

import "strings"

// NewOperationSkillTool returns the Tool definition for an operation skill
// (e.g. "qsdev-doctor"). Operation skills are AlwaysOn and own a single
// exclusive SKILL.md. The list of operation skills is not defined here: the
// claudecode addon registers one tool per entry of its operation-skill
// manifest, so the manifest that deploys the skills is the only source of
// truth for which exist.
func NewOperationSkillTool(name, description string) Tool {
	return Tool{
		Name:        name,
		DisplayName: strings.Replace(name, "-", " ", 1),
		Category:    CategoryAIAgent,
		Description: description,
		Default:     AlwaysOn,
		OwnedFiles: []FileOwnership{
			{Path: ".claude/skills/" + name + "/SKILL.md", Ownership: Exclusive},
		},
	}
}
