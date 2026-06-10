package policy

import (
	"fmt"
	"slices"
)

func Evaluate(set *CompiledPolicySet, ctx *EvalContext) (decision PolicyDecision) {
	defer func() {
		if r := recover(); r != nil {
			decision = PolicyDecision{
				Action:   Block,
				ExitCode: 2,
				Message:  fmt.Sprintf("policy engine panic: %v", r),
			}
		}
	}()

	candidates := RulesForTool(set, ctx.ToolName)

	var findings []Finding

	for _, rule := range candidates {
		if !matchesTierFilter(rule.Rule.BypassTier, ctx.TierFilter) {
			continue
		}

		if rule.Rule.MonitorMode && rule.Rule.Action.Type == Block {
			findings = append(findings, Finding{
				RuleID:   rule.Rule.ID,
				Category: rule.Rule.Category,
				Severity: rule.Rule.Severity,
				Message:  interpolateMessage(rule.Rule.Action.Message, ctx),
				Monitor:  true,
			})
			continue
		}

		if (rule.Rule.BypassTier == Session || rule.Rule.BypassTier == Command) &&
			slices.Contains(ctx.SessionOverrides, rule.Rule.ID) {
			continue
		}

		matched, err := rule.Condition.Evaluate(ctx)
		if err != nil {
			return PolicyDecision{
				Action:   Block,
				ExitCode: 2,
				RuleID:   rule.Rule.ID,
				Message:  fmt.Sprintf("evaluating condition for rule %s: %v", rule.Rule.ID, err),
				Err:      fmt.Errorf("evaluating condition for rule %s: %w", rule.Rule.ID, err),
			}
		}

		if !matched {
			continue
		}

		result := rule.Action.Execute(&rule.Rule, ctx)

		if rule.Rule.MonitorMode {
			for i := range result.Findings {
				result.Findings[i].Monitor = true
			}
			if result.Action == Block {
				findings = append(findings, Finding{
					RuleID:   rule.Rule.ID,
					Category: rule.Rule.Category,
					Severity: rule.Rule.Severity,
					Message:  result.Message,
					Monitor:  true,
				})
				continue
			}
			findings = append(findings, result.Findings...)
			continue
		}

		if result.Action == Block {
			return result
		}

		if result.Action == Prompt {
			result.Findings = append(result.Findings, findings...)
			return result
		}

		findings = append(findings, result.Findings...)
	}

	return PolicyDecision{
		Action:   "",
		ExitCode: 0,
		Findings: findings,
	}
}

func matchesTierFilter(tier BypassTier, filter TierFilter) bool {
	switch filter {
	case AllTiers:
		return true
	case EnforceAlwaysOnly:
		return tier == EnforceAlways
	case SessionCommandOnly:
		return tier == Session || tier == Command
	default:
		return true
	}
}
