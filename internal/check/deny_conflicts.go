package check

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/pkg/denyutil"
)

// SkillOps describes a skill and the tool operations it requires.
// This mirrors claudecode.SkillDefinition to avoid circular imports.
type SkillOps struct {
	Name string
	// AllowedTools is every tool the skill or subagent declares, checked
	// against deny rules.
	AllowedTools []string
	// PreApproved is the subset a skill's allowed-tools grants without a
	// prompt. A subagent's tools list only makes tools available, so it
	// contributes nothing here.
	PreApproved []string
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
				if denyutil.Shadows(deny, op) {
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

// CheckSkillPreApprovals validates that no skill pre-approves a command the
// catalog gates behind ask. A skill's allowed-tools grant runs without a
// prompt while the skill is active, so a rule such as Bash(npm *) or Bash(*)
// would let a model-invoked skill install packages or execute code without
// the ask prompt and the package-guard review that follows it. Each ask rule
// is reduced to a sample command and every Bash pre-approval is matched
// against it. Deny rules are not considered: deny always wins.
func CheckSkillPreApprovals(ctx CheckContext) []CheckResult {
	if len(ctx.AskRules) == 0 || len(ctx.SkillOps) == 0 {
		return []CheckResult{skillPreApprovalResult(StatusSkip, "No ask rules or skill definitions to validate")}
	}

	var results []CheckResult
	for _, skill := range ctx.SkillOps {
		for _, grant := range skill.PreApproved {
			matcher := grant
			if matcher == "Bash" {
				// A bare tool name grants every command of that tool.
				matcher = "Bash(*)"
			}
			for _, ask := range ctx.AskRules {
				sample, ok := denyutil.SampleCommand(ask)
				if !ok || !denyutil.MatchesBashRule(matcher, sample) {
					continue
				}
				results = append(results, CheckResult{
					Category: CategoryDenyConflicts,
					Name:     fmt.Sprintf("skill_preapproval_%s_%s", skill.Name, sanitizeName(ask)),
					Status:   StatusFail,
					Severity: SeverityHigh,
					Message: fmt.Sprintf(
						"skill %q pre-approves %q, which bypasses ask rule %q",
						skill.Name, grant, ask),
					Remediation: "Narrow the skill's allowed-tools to read-only commands; let ask-gated commands prompt",
				})
			}
		}
	}
	if len(results) == 0 {
		return []CheckResult{skillPreApprovalResult(StatusPass,
			fmt.Sprintf("No skill pre-approves a command gated by any of %d ask rules", len(ctx.AskRules)))}
	}
	return results
}

func skillPreApprovalResult(status CheckStatus, msg string) CheckResult {
	return CheckResult{
		Category: CategoryDenyConflicts,
		Name:     "skill_preapproval",
		Status:   status,
		Severity: SeverityInfo,
		Message:  msg,
	}
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
