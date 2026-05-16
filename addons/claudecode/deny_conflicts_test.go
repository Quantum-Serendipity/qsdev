package claudecode

import "testing"

func TestValidateDenyRuleConflicts_NoConflicts(t *testing.T) {
	// Safe deny rules that don't overlap with safe skill operations.
	denyRules := []string{
		"Bash(npm install *)",
		"Bash(pip install *)",
		"Read(./.env)",
	}
	skills := []SkillDefinition{
		{Name: "review-pr", AllowedTools: []string{"Bash(git *)", "Bash(gh *)"}},
		{Name: "add-tests", AllowedTools: []string{"Bash(npm test *)", "Bash(go test *)"}},
	}

	conflicts := ValidateDenyRuleConflicts(denyRules, skills)
	if len(conflicts) != 0 {
		t.Errorf("expected no conflicts, got %d: %+v", len(conflicts), conflicts)
	}
}

func TestValidateDenyRuleConflicts_DetectsConflict(t *testing.T) {
	// An overly broad deny rule that blocks a skill operation.
	denyRules := []string{
		"Bash(npm *)", // Too broad — blocks npm test too.
	}
	skills := []SkillDefinition{
		{Name: "add-tests", AllowedTools: []string{"Bash(npm test *)"}},
	}

	conflicts := ValidateDenyRuleConflicts(denyRules, skills)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d: %+v", len(conflicts), conflicts)
	}
	if conflicts[0].Skill != "add-tests" {
		t.Errorf("conflict skill = %q, want %q", conflicts[0].Skill, "add-tests")
	}
	if conflicts[0].DenyRule != "Bash(npm *)" {
		t.Errorf("conflict deny rule = %q, want %q", conflicts[0].DenyRule, "Bash(npm *)")
	}
	if conflicts[0].Operation != "Bash(npm test *)" {
		t.Errorf("conflict operation = %q, want %q", conflicts[0].Operation, "Bash(npm test *)")
	}
}

func TestValidateDenyRuleConflicts_UpgradeDepNoConflicts(t *testing.T) {
	// Package install commands are now in ask (not deny), so upgrade-dep
	// should have zero conflicts with the deny list.
	denyRules := AllBaseDenyRules()
	skills := []SkillDefinition{
		{Name: "upgrade-dep", AllowedTools: []string{
			"Bash(npm install *)", "Bash(npm uninstall *)",
			"Bash(pip install *)", "Bash(cargo install *)",
		}},
	}

	conflicts := ValidateDenyRuleConflicts(denyRules, skills)
	if len(conflicts) != 0 {
		for _, c := range conflicts {
			t.Errorf("unexpected conflict: %s", c.Message)
		}
		t.Fatalf("expected no conflicts for upgrade-dep skill (package installs are in ask), got %d", len(conflicts))
	}
}

func TestFilterExpectedConflicts_NoExpectedConflicts(t *testing.T) {
	// With package installs moved to ask, there are no expected conflicts.
	// Any conflict passed to FilterExpectedConflicts should come back as unexpected.
	conflicts := []DenyRuleConflict{
		{Skill: "some-skill", DenyRule: "Bash(something *)", Operation: "Bash(something *)"},
	}

	unexpected := FilterExpectedConflicts(conflicts)
	if len(unexpected) != 1 {
		t.Errorf("expected 1 unexpected conflict (no expected conflicts exist), got %d",
			len(unexpected))
	}
}

func TestFilterExpectedConflicts_PassesThroughAllConflicts(t *testing.T) {
	// Since ExpectedConflicts() is empty, all conflicts are unexpected.
	conflicts := []DenyRuleConflict{
		{Skill: "add-tests", DenyRule: "Bash(npm *)", Operation: "Bash(npm test *)"},
		{Skill: "some-skill", DenyRule: "Bash(other *)", Operation: "Bash(other thing *)"},
	}

	unexpected := FilterExpectedConflicts(conflicts)
	if len(unexpected) != 2 {
		t.Fatalf("expected 2 unexpected conflicts (none are expected), got %d: %+v", len(unexpected), unexpected)
	}
}

func TestBuiltinSkillDefinitions_NotEmpty(t *testing.T) {
	skills := BuiltinSkillDefinitions()
	if len(skills) == 0 {
		t.Fatal("BuiltinSkillDefinitions should not be empty")
	}
}

func TestBuiltinSkillDefinitions_QsdevOpsUseGdevBash(t *testing.T) {
	skills := BuiltinSkillDefinitions()

	qsdevSkills := []string{
		"qsdev-init", "qsdev-onboard", "qsdev-setup", "qsdev-enable",
		"qsdev-disable", "qsdev-update", "qsdev-doctor", "qsdev-status",
		"qsdev-tools", "qsdev-detect",
	}

	skillMap := make(map[string]SkillDefinition)
	for _, s := range skills {
		skillMap[s.Name] = s
	}

	for _, name := range qsdevSkills {
		s, ok := skillMap[name]
		if !ok {
			t.Errorf("expected skill %q to exist", name)
			continue
		}
		if len(s.AllowedTools) != 1 || s.AllowedTools[0] != "Bash(qsdev *)" {
			t.Errorf("skill %q AllowedTools = %v, want [Bash(qsdev *)]", name, s.AllowedTools)
		}
	}
}

func TestExpectedConflicts_Empty(t *testing.T) {
	// Package installs are now in ask, not deny. No expected conflicts remain.
	ec := ExpectedConflicts()
	if len(ec) != 0 {
		t.Fatalf("ExpectedConflicts should be empty (package installs moved to ask), got %d entries", len(ec))
	}
}

func TestValidateDenyRuleConflicts_BuiltinSkillsNoUnexpectedConflicts(t *testing.T) {
	// This is the integration test: validate that the actual deny rules
	// and actual skill definitions produce no unexpected conflicts.
	denyRules := AllBaseDenyRules()
	skills := BuiltinSkillDefinitions()

	conflicts := ValidateDenyRuleConflicts(denyRules, skills)
	unexpected := FilterExpectedConflicts(conflicts)

	if len(unexpected) > 0 {
		for _, c := range unexpected {
			t.Errorf("unexpected conflict: %s", c.Message)
		}
		t.Fatalf("%d unexpected deny rule conflicts detected", len(unexpected))
	}
}
