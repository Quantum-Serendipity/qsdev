package claudecode

import "fmt"

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
// These are conflicts that exist by design — the deny rules block package
// installs, but the upgrade-dep skill legitimately needs those operations.
// The PreToolUse guardrail hook validates packages before allowing them.
// Key format: "skillName:denyRule"
func ExpectedConflicts() map[string]string {
	return map[string]string{
		"upgrade-dep:Bash(npm install *)":    "Works with PreToolUse guardrail hook which validates packages",
		"upgrade-dep:Bash(npm uninstall *)":  "Package removal through guardrail hook",
		"upgrade-dep:Bash(yarn add *)":       "Works with PreToolUse guardrail hook",
		"upgrade-dep:Bash(yarn remove *)":    "Package removal through guardrail hook",
		"upgrade-dep:Bash(pnpm add *)":       "Works with PreToolUse guardrail hook",
		"upgrade-dep:Bash(pnpm remove *)":    "Package removal through guardrail hook",
		"upgrade-dep:Bash(bun add *)":        "Works with PreToolUse guardrail hook",
		"upgrade-dep:Bash(bun remove *)":     "Package removal through guardrail hook",
		"upgrade-dep:Bash(pip install *)":    "Works with PreToolUse guardrail hook",
		"upgrade-dep:Bash(pip uninstall *)":  "Package removal through guardrail hook",
		"upgrade-dep:Bash(uv pip install *)": "Works with PreToolUse guardrail hook",
		"upgrade-dep:Bash(cargo install *)":  "Works with PreToolUse guardrail hook",
		"upgrade-dep:Bash(go install *)":     "Works with PreToolUse guardrail hook",
	}
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
	return []SkillDefinition{
		// gdev operation skills (14.1)
		{Name: "gdev-init", AllowedTools: []string{"Bash(gdev *)"}},
		{Name: "gdev-onboard", AllowedTools: []string{"Bash(gdev *)"}},
		{Name: "gdev-setup", AllowedTools: []string{"Bash(gdev *)"}},
		{Name: "gdev-enable", AllowedTools: []string{"Bash(gdev *)"}},
		{Name: "gdev-disable", AllowedTools: []string{"Bash(gdev *)"}},
		{Name: "gdev-update", AllowedTools: []string{"Bash(gdev *)"}},
		{Name: "gdev-doctor", AllowedTools: []string{"Bash(gdev *)"}},
		{Name: "gdev-status", AllowedTools: []string{"Bash(gdev *)"}},
		{Name: "gdev-tools", AllowedTools: []string{"Bash(gdev *)"}},
		{Name: "gdev-detect", AllowedTools: []string{"Bash(gdev *)"}},
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
	}
}
