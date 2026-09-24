package policy

import (
	"fmt"
	"slices"
)

// Evaluate runs the candidate rules for ctx's tool in (tier, severity) order and
// returns the decision. A Block ends evaluation immediately. An allowing Prompt
// (default_on_timeout allow) is recorded but does not end evaluation: a later
// rule that blocks the same call must still block it, so the prompt decision is
// returned only once every candidate rule has been checked. Monitor-mode rules
// are evaluated like any other rule, but a match only records a finding.
//
// A session-tier rule with an active session grant in ctx.Overrides is
// skipped. A command-tier rule holding a one-shot token is still evaluated;
// when it matches, the rule is lifted for this call and its ID is reported in
// the decision's ConsumedTokens, which the caller must redeem. A blocking
// decision reports no tokens, since the call does not run.
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
	var prompt *PolicyDecision
	var consumed []string

	for _, rule := range candidates {
		if !matchesTierFilter(rule.Rule.BypassTier, ctx.TierFilter) {
			continue
		}

		if rule.Rule.BypassTier == Session && slices.Contains(ctx.Overrides.Session, rule.Rule.ID) {
			continue
		}
		// A monitor-mode rule never blocks, so it must not spend a token.
		tokenHeld := rule.Rule.BypassTier == Command && !rule.Rule.MonitorMode &&
			slices.Contains(ctx.Overrides.Command, rule.Rule.ID)

		matched, err := rule.Condition.Evaluate(ctx)
		if err != nil {
			if rule.Rule.MonitorMode {
				// A monitor-only rule never blocks, not even on an error.
				findings = append(findings, Finding{
					RuleID:   rule.Rule.ID,
					Category: rule.Rule.Category,
					Severity: rule.Rule.Severity,
					Message:  fmt.Sprintf("evaluating condition for rule %s: %v", rule.Rule.ID, err),
					Monitor:  true,
				})
				continue
			}
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

		if tokenHeld {
			if !slices.Contains(consumed, rule.Rule.ID) {
				consumed = append(consumed, rule.Rule.ID)
			}
			continue
		}

		result := rule.Action.Execute(&rule.Rule, ctx)

		if rule.Rule.MonitorMode {
			findings = append(findings, monitorFindings(&rule.Rule, result)...)
			continue
		}

		switch result.Action {
		case Block:
			result.BypassTier = rule.Rule.BypassTier
			return result
		case Prompt:
			if prompt == nil {
				prompt = &result
			}
		default:
			findings = append(findings, result.Findings...)
		}
	}

	if prompt != nil {
		// A prompt reaches here only when it resolved to allow (its
		// fail-closed form returns Block above); it did not end evaluation,
		// so a later rule could still block the call.
		prompt.Findings = append(prompt.Findings, findings...)
		prompt.ConsumedTokens = consumed
		return *prompt
	}

	return PolicyDecision{
		Action:         "",
		ExitCode:       0,
		Findings:       findings,
		ConsumedTokens: consumed,
	}
}

// monitorFindings downgrades a matched monitor-mode rule's decision to
// findings: a would-be Block becomes a single monitor finding, and any findings
// the action produced are marked as monitor findings.
func monitorFindings(rule *PolicyRule, result PolicyDecision) []Finding {
	if result.Action == Block {
		return []Finding{{
			RuleID:   rule.ID,
			Category: rule.Category,
			Severity: rule.Severity,
			Message:  result.Message,
			Monitor:  true,
		}}
	}
	for i := range result.Findings {
		result.Findings[i].Monitor = true
	}
	return result.Findings
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
