package claudecode

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// DenyRuleConflict represents a conflict between a deny rule and a skill operation.
type DenyRuleConflict struct {
	Skill     string
	Operation string
	DenyRule  string
	Message   string
}

// SkillDefinition describes a skill and the tool operations it requires.
type SkillDefinition struct {
	Name         string
	AllowedTools []string
}

// ValidateDenyRuleConflicts checks all deny rules against all skill operations
// and returns any conflicts where a deny rule would block an operation that a
// skill needs to function.
func ValidateDenyRuleConflicts(denyRules []string, skills []SkillDefinition) []DenyRuleConflict {
	var conflicts []DenyRuleConflict
	for _, skill := range skills {
		for _, op := range skill.AllowedTools {
			for _, deny := range denyRules {
				if matchesDenyRule(deny, op) {
					conflicts = append(conflicts, DenyRuleConflict{
						Skill:     skill.Name,
						Operation: op,
						DenyRule:  deny,
						Message: fmt.Sprintf(
							"skill %q needs %q but deny rule %q would block it",
							skill.Name, op, deny),
					})
				}
			}
		}
	}
	return conflicts
}

// ExpectedConflicts returns the map of known expected conflicts.
// These are conflicts that exist by design — where deny rules intentionally
// block operations that a skill would otherwise need.
// Key format: "skillName:denyRule"
//
// Note: Package install operations (npm install, pip install, etc.) are no
// longer in deny — they are in the ask list, gated by the PreToolUse
// package-guard hook. This means upgrade-dep no longer conflicts with deny rules.
func ExpectedConflicts() map[string]string {
	return map[string]string{}
}

// FilterExpectedConflicts removes known expected conflicts from the list,
// returning only unexpected conflicts that indicate a genuine configuration error.
func FilterExpectedConflicts(conflicts []DenyRuleConflict) []DenyRuleConflict {
	expected := ExpectedConflicts()
	var unexpected []DenyRuleConflict
	for _, c := range conflicts {
		key := c.Skill + ":" + c.DenyRule
		if _, ok := expected[key]; !ok {
			unexpected = append(unexpected, c)
		}
	}
	return unexpected
}

// BuiltinSkillDefinitions returns the operations required by all built-in skills.
// This is a static registry derived from the skill YAML frontmatter allowed-tools fields.
func BuiltinSkillDefinitions() []SkillDefinition {
	app := branding.Get().AppName
	bashTool := "Bash(" + app + " *)"
	return []SkillDefinition{
		// qsdev operation skills (14.1)
		{Name: app + "-init", AllowedTools: []string{bashTool}},
		{Name: app + "-onboard", AllowedTools: []string{bashTool}},
		{Name: app + "-setup", AllowedTools: []string{bashTool}},
		{Name: app + "-enable", AllowedTools: []string{bashTool}},
		{Name: app + "-disable", AllowedTools: []string{bashTool}},
		{Name: app + "-update", AllowedTools: []string{bashTool}},
		{Name: app + "-doctor", AllowedTools: []string{bashTool}},
		{Name: app + "-status", AllowedTools: []string{bashTool}},
		{Name: app + "-tools", AllowedTools: []string{bashTool}},
		{Name: app + "-detect", AllowedTools: []string{bashTool}},
		// consulting workflow skills (14.3)
		{Name: "review-pr", AllowedTools: []string{"Bash(git *)", "Bash(gh *)"}},
		{Name: "add-tests", AllowedTools: []string{"Bash(npm test *)", "Bash(go test *)", "Bash(pytest *)", "Bash(cargo test *)"}},
		{Name: "upgrade-dep", AllowedTools: []string{
			"Bash(npm install *)", "Bash(npm uninstall *)",
			"Bash(yarn add *)", "Bash(yarn remove *)",
			"Bash(pnpm add *)", "Bash(pnpm remove *)",
			"Bash(bun add *)", "Bash(bun remove *)",
			"Bash(pip install *)", "Bash(pip uninstall *)",
			"Bash(uv pip install *)",
			"Bash(cargo install *)", "Bash(go install *)",
		}},
		{Name: "incident-debug", AllowedTools: []string{"Bash(git *)", "Bash(grep *)"}},
		{Name: "migration-plan", AllowedTools: []string{"Bash(git *)", "Bash(find *)", "Bash(wc *)"}},
		// Container runtime migration skill (17.3)
		{
			Name: "container-migrate",
			AllowedTools: []string{
				fmt.Sprintf("Bash(%s container migrate *)", app),
				fmt.Sprintf("Bash(%s container detect)", app),
			},
		},
	}
}
