package check

import "testing"

func TestCheckDenyRuleConflicts_NoConflicts(t *testing.T) {
	ctx := CheckContext{
		DenyRules: []string{
			"Bash(npm install *)",
			"Bash(pip install *)",
		},
		SkillOps: []SkillOps{
			{Name: "review-pr", AllowedTools: []string{"Bash(git *)", "Bash(gh *)"}},
			{Name: "add-tests", AllowedTools: []string{"Bash(npm test *)", "Bash(go test *)"}},
		},
		ExpectedConflictKeys: map[string]string{},
	}

	results := CheckDenyRuleConflicts(ctx)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusPass {
		t.Errorf("status = %s, want %s", results[0].Status, StatusPass)
	}
}

func TestCheckDenyRuleConflicts_SkipWhenEmpty(t *testing.T) {
	// No deny rules.
	ctx := CheckContext{
		DenyRules:            nil,
		SkillOps:             []SkillOps{{Name: "test", AllowedTools: []string{"Bash(test *)"}}},
		ExpectedConflictKeys: map[string]string{},
	}

	results := CheckDenyRuleConflicts(ctx)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusSkip {
		t.Errorf("status = %s, want %s", results[0].Status, StatusSkip)
	}

	// No skills.
	ctx2 := CheckContext{
		DenyRules:            []string{"Bash(npm install *)"},
		SkillOps:             nil,
		ExpectedConflictKeys: map[string]string{},
	}

	results2 := CheckDenyRuleConflicts(ctx2)
	if len(results2) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results2))
	}
	if results2[0].Status != StatusSkip {
		t.Errorf("status = %s, want %s", results2[0].Status, StatusSkip)
	}
}

func TestCheckDenyRuleConflicts_ReportsUnexpected(t *testing.T) {
	ctx := CheckContext{
		DenyRules: []string{
			"Bash(npm *)", // Overly broad — blocks npm test.
		},
		SkillOps: []SkillOps{
			{Name: "add-tests", AllowedTools: []string{"Bash(npm test *)"}},
		},
		ExpectedConflictKeys: map[string]string{},
	}

	results := CheckDenyRuleConflicts(ctx)
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Status != StatusFail {
		t.Errorf("status = %s, want %s", results[0].Status, StatusFail)
	}
	if results[0].Severity != SeverityHigh {
		t.Errorf("severity = %s, want %s", results[0].Severity, SeverityHigh)
	}
	if results[0].Category != CategoryDenyConflicts {
		t.Errorf("category = %s, want %s", results[0].Category, CategoryDenyConflicts)
	}
}

func TestCheckDenyRuleConflicts_FiltersExpected(t *testing.T) {
	ctx := CheckContext{
		DenyRules: []string{
			"Bash(npm install *)",
			"Bash(pip install *)",
		},
		SkillOps: []SkillOps{
			{Name: "upgrade-dep", AllowedTools: []string{
				"Bash(npm install *)",
				"Bash(pip install *)",
			}},
		},
		ExpectedConflictKeys: map[string]string{
			"upgrade-dep:Bash(npm install *)": "expected",
			"upgrade-dep:Bash(pip install *)": "expected",
		},
	}

	results := CheckDenyRuleConflicts(ctx)
	if len(results) != 1 {
		t.Fatalf("expected 1 result (pass), got %d", len(results))
	}
	if results[0].Status != StatusPass {
		t.Errorf("status = %s, want %s (all conflicts are expected)", results[0].Status, StatusPass)
	}
}

func TestCheckDenyRuleConflicts_MixedExpectedAndUnexpected(t *testing.T) {
	ctx := CheckContext{
		DenyRules: []string{
			"Bash(npm install *)",
			"Bash(npm *)", // Overly broad.
		},
		SkillOps: []SkillOps{
			{Name: "upgrade-dep", AllowedTools: []string{"Bash(npm install *)"}},
			{Name: "add-tests", AllowedTools: []string{"Bash(npm test *)"}},
		},
		ExpectedConflictKeys: map[string]string{
			"upgrade-dep:Bash(npm install *)": "expected",
		},
	}

	results := CheckDenyRuleConflicts(ctx)

	// Should report failures for the unexpected conflicts.
	hasFail := false
	for _, r := range results {
		if r.Status == StatusFail {
			hasFail = true
		}
	}
	if !hasFail {
		t.Error("expected at least one fail result for unexpected conflicts")
	}
}

func TestCheckMatchesDenyRule_SameAsCoreLogic(t *testing.T) {
	// Verify the check-local copy of matching logic behaves correctly.
	tests := []struct {
		deny   string
		op     string
		expect bool
	}{
		{"Bash(npm install *)", "Bash(npm install lodash)", true},
		{"Bash(npm install *)", "Bash(npm test *)", false},
		{"Bash(npm *)", "Bash(npm test *)", true},
		{"Bash(npm *)", "Read(.env)", false},
		{"Read(./.env)", "Read(./.env)", true},
		{"Read(./.env)", "Read(./README.md)", false},
	}

	for _, tc := range tests {
		got := checkMatchesDenyRule(tc.deny, tc.op)
		if got != tc.expect {
			t.Errorf("checkMatchesDenyRule(%q, %q) = %v, want %v",
				tc.deny, tc.op, got, tc.expect)
		}
	}
}

func TestCategoryDenyConflicts_DisplayName(t *testing.T) {
	name := categoryDisplayName(CategoryDenyConflicts)
	if name != "Deny Rule Conflicts" {
		t.Errorf("categoryDisplayName(CategoryDenyConflicts) = %q, want %q", name, "Deny Rule Conflicts")
	}
}
