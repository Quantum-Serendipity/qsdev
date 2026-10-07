package check

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
)

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

func TestDenyRuleConflicts_Shadows(t *testing.T) {
	// The conflict check reports a deny rule that shadows a skill operation.
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
		got := denyutil.Shadows(tc.deny, tc.op)
		if got != tc.expect {
			t.Errorf("denyutil.Shadows(%q, %q) = %v, want %v",
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

func TestCheckSkillPreApprovals(t *testing.T) {
	t.Parallel()
	askRules := []string{"Bash(npm install *)", "Bash(go get *)", "Bash(npm run *)"}
	tests := []struct {
		name        string
		askRules    []string
		skills      []SkillOps
		wantStatus  CheckStatus
		wantFails   int
		wantMessage []string
	}{
		{
			name:        "bare wildcard bypasses every ask rule",
			askRules:    askRules,
			skills:      []SkillOps{{Name: "add-tests", PreApproved: []string{"Bash(*)", "Read"}}},
			wantStatus:  StatusFail,
			wantFails:   3,
			wantMessage: []string{`"add-tests"`, `"Bash(*)"`, `"Bash(npm install *)"`},
		},
		{
			name:        "bare Bash tool is an unrestricted grant",
			askRules:    askRules,
			skills:      []SkillOps{{Name: "s", PreApproved: []string{"Bash"}}},
			wantStatus:  StatusFail,
			wantFails:   3,
			wantMessage: []string{`"Bash"`, `"Bash(go get *)"`},
		},
		{
			name:        "package manager wildcard bypasses its ask rules",
			askRules:    askRules,
			skills:      []SkillOps{{Name: "qsdev-add-dep", PreApproved: []string{"Bash(npm *)"}}},
			wantStatus:  StatusFail,
			wantFails:   2,
			wantMessage: []string{`"Bash(npm *)"`, `"Bash(npm run *)"`},
		},
		{
			name:       "prefix wildcard without a space bypasses ask",
			askRules:   askRules,
			skills:     []SkillOps{{Name: "s", PreApproved: []string{"Bash(npm i*)"}}},
			wantStatus: StatusFail,
			wantFails:  1,
		},
		{
			name:       "read-only subcommand passes",
			askRules:   askRules,
			skills:     []SkillOps{{Name: "qsdev-add-dep", PreApproved: []string{"Bash(npm view *)", "Bash(go list *)"}}},
			wantStatus: StatusPass,
		},
		{
			name:       "non-Bash tool is ignored",
			askRules:   askRules,
			skills:     []SkillOps{{Name: "s", PreApproved: []string{"Read", "mcp__context7(*)", "Write"}}},
			wantStatus: StatusPass,
		},
		{
			name:       "agent tools are not pre-approvals",
			askRules:   askRules,
			skills:     []SkillOps{{Name: "security-reviewer", AllowedTools: []string{"Bash"}}},
			wantStatus: StatusPass,
		},
		{
			name:       "no ask rules skips",
			skills:     []SkillOps{{Name: "s", PreApproved: []string{"Bash(*)"}}},
			wantStatus: StatusSkip,
		},
		{
			name:       "no skills skips",
			askRules:   askRules,
			wantStatus: StatusSkip,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			results := CheckSkillPreApprovals(CheckContext{AskRules: tt.askRules, SkillOps: tt.skills})
			var fails []CheckResult
			for _, r := range results {
				if r.Category != CategoryDenyConflicts {
					t.Errorf("result %q category = %s, want %s", r.Name, r.Category, CategoryDenyConflicts)
				}
				if r.Status == StatusFail {
					fails = append(fails, r)
					if r.Severity != SeverityHigh {
						t.Errorf("fail %q severity = %s, want %s", r.Name, r.Severity, SeverityHigh)
					}
				}
			}
			if tt.wantStatus != StatusFail {
				if len(results) != 1 || results[0].Status != tt.wantStatus || results[0].Name != "skill_preapproval" {
					t.Fatalf("results = %+v, want one skill_preapproval %s result", results, tt.wantStatus)
				}
				return
			}
			if len(fails) != tt.wantFails || len(fails) != len(results) {
				t.Fatalf("got %d fails of %d results, want %d: %+v", len(fails), len(results), tt.wantFails, results)
			}
			all := ""
			for _, f := range fails {
				all += f.Message + "\n"
			}
			for _, want := range tt.wantMessage {
				if !strings.Contains(all, want) {
					t.Errorf("messages %q do not name %s", all, want)
				}
			}
		})
	}
}
