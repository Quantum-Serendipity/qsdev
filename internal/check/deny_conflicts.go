package check

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
)

// SkillOps describes a skill and the tool operations it requires.
// This mirrors claudecode.SkillDefinition to avoid circular imports.
type SkillOps struct {
	Name         string
	AllowedTools []string
}

// CheckDenyRuleConflicts validates that deny rules don't block skill operations.
// It uses the deny rules and skill definitions from CheckContext and reports
// any unexpected conflicts (after filtering out known expected conflicts).
func CheckDenyRuleConflicts(ctx CheckContext) []CheckResult {
	if len(ctx.DenyRules) == 0 || len(ctx.SkillOps) == 0 {
		return []CheckResult{{
			Category: CategoryDenyConflicts,
			Name:     "deny_rule_conflicts",
			Status:   StatusSkip,
			Severity: SeverityInfo,
			Message:  "No deny rules or skill definitions to validate",
		}}
	}

	// Find all conflicts.
	var allConflicts []denyConflict
	for _, skill := range ctx.SkillOps {
		for _, op := range skill.AllowedTools {
			for _, deny := range ctx.DenyRules {
				if checkMatchesDenyRule(deny, op) {
					allConflicts = append(allConflicts, denyConflict{
						skill:     skill.Name,
						operation: op,
						denyRule:  deny,
					})
				}
			}
		}
	}

	// Filter out expected conflicts.
	var unexpected []denyConflict
	for _, c := range allConflicts {
		key := c.skill + ":" + c.denyRule
		if _, ok := ctx.ExpectedConflictKeys[key]; !ok {
			unexpected = append(unexpected, c)
		}
	}

	if len(unexpected) == 0 {
		return []CheckResult{{
			Category: CategoryDenyConflicts,
			Name:     "deny_rule_conflicts",
			Status:   StatusPass,
			Severity: SeverityInfo,
			Message:  fmt.Sprintf("No unexpected deny rule conflicts (%d expected conflicts verified)", len(allConflicts)),
		}}
	}

	// Report each unexpected conflict as a separate check result.
	var results []CheckResult
	for _, c := range unexpected {
		results = append(results, CheckResult{
			Category: CategoryDenyConflicts,
			Name:     fmt.Sprintf("deny_conflict_%s_%s", c.skill, sanitizeName(c.denyRule)),
			Status:   StatusFail,
			Severity: SeverityHigh,
			Message: fmt.Sprintf(
				"skill %q needs %q but deny rule %q would block it",
				c.skill, c.operation, c.denyRule),
			Remediation: "Either add this conflict to the expected conflicts list or adjust the deny rule",
		})
	}
	return results
}

// denyConflict is an internal type used within the check package.
type denyConflict struct {
	skill     string
	operation string
	denyRule  string
}

// sanitizeName converts a deny rule pattern to a safe check name suffix.
func sanitizeName(rule string) string {
	r := strings.NewReplacer(
		"(", "_",
		")", "",
		" ", "_",
		"*", "star",
		".", "dot",
		"/", "_",
	)
	return r.Replace(rule)
}

func checkMatchesDenyRule(denyRule, operation string) bool {
	return denyutil.MatchesDenyRule(denyRule, operation)
}
