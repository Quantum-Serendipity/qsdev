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

func TestValidateDenyRuleConflicts_UpgradeDepExpected(t *testing.T) {
	// The upgrade-dep skill intentionally conflicts with package install deny rules.
	denyRules := AllBaseDenyRules()
	skills := []SkillDefinition{
		{Name: "upgrade-dep", AllowedTools: []string{
			"Bash(npm install *)", "Bash(npm uninstall *)",
			"Bash(pip install *)", "Bash(cargo install *)",
		}},
	}

	conflicts := ValidateDenyRuleConflicts(denyRules, skills)
	if len(conflicts) == 0 {
		t.Fatal("expected conflicts for upgrade-dep skill")
	}

	// All conflicts should be for the upgrade-dep skill.
	for _, c := range conflicts {
		if c.Skill != "upgrade-dep" {
			t.Errorf("unexpected conflict for skill %q", c.Skill)
		}
	}
}

func TestFilterExpectedConflicts_RemovesKnown(t *testing.T) {
	conflicts := []DenyRuleConflict{
		{Skill: "upgrade-dep", DenyRule: "Bash(npm install *)", Operation: "Bash(npm install *)"},
		{Skill: "upgrade-dep", DenyRule: "Bash(pip install *)", Operation: "Bash(pip install *)"},
		{Skill: "upgrade-dep", DenyRule: "Bash(cargo install *)", Operation: "Bash(cargo install *)"},
	}

	unexpected := FilterExpectedConflicts(conflicts)
	if len(unexpected) != 0 {
		t.Errorf("expected all conflicts to be filtered as expected, got %d unexpected: %+v",
			len(unexpected), unexpected)
	}
}

func TestFilterExpectedConflicts_KeepsUnexpected(t *testing.T) {
	conflicts := []DenyRuleConflict{
		{Skill: "upgrade-dep", DenyRule: "Bash(npm install *)", Operation: "Bash(npm install *)"},
		{Skill: "add-tests", DenyRule: "Bash(npm *)", Operation: "Bash(npm test *)"},
	}

	unexpected := FilterExpectedConflicts(conflicts)
	if len(unexpected) != 1 {
		t.Fatalf("expected 1 unexpected conflict, got %d: %+v", len(unexpected), unexpected)
	}
	if unexpected[0].Skill != "add-tests" {
		t.Errorf("unexpected conflict skill = %q, want %q", unexpected[0].Skill, "add-tests")
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

func TestExpectedConflicts_NonEmpty(t *testing.T) {
	ec := ExpectedConflicts()
	if len(ec) == 0 {
		t.Fatal("ExpectedConflicts should not be empty")
	}
}

func TestExpectedConflicts_AllKeysReferenceUpgradeDep(t *testing.T) {
	ec := ExpectedConflicts()
	for key := range ec {
		if len(key) < len("upgrade-dep:") {
			t.Errorf("expected conflict key %q is too short", key)
			continue
		}
		if key[:len("upgrade-dep:")] != "upgrade-dep:" {
			t.Errorf("expected conflict key %q does not start with 'upgrade-dep:'", key)
		}
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
