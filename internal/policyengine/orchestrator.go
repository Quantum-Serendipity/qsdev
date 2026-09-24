package policyengine

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/sarif"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/trust"
)

type SecurityOrchestrator struct {
	policy PolicyEvaluator
	risk   PackageRiskScorer
	trust  McpTrustEvaluator
	// servers holds the configured MCP server definitions keyed by server
	// name, used to score a server's trust tier from what actually runs.
	servers map[string]trust.McpServerInfo
}

func NewSecurityOrchestrator(p PolicyEvaluator, r PackageRiskScorer, t McpTrustEvaluator) *SecurityOrchestrator {
	return &SecurityOrchestrator{
		policy: p,
		risk:   r,
		trust:  t,
	}
}

// WithMcpServers supplies the configured MCP server definitions (for example
// from the project's .mcp.json) that trust tiers are scored from. A server
// with no definition scores into the fallback tier.
func (o *SecurityOrchestrator) WithMcpServers(servers map[string]trust.McpServerInfo) *SecurityOrchestrator {
	o.servers = servers
	return o
}

// ConfusedDeputyRuleID is the RuleID reported on a decision produced by the
// MCP confused-deputy check rather than by a policy rule.
const ConfusedDeputyRuleID = "confused-deputy"

// RunPreToolUse evaluates a tool call in three stages: enforce_always rules,
// the MCP confused-deputy check, then session/command-tier rules. It returns
// the decision together with the hook exit code. On a block (exit 2) the
// decision carries the blocking RuleID and Message so the caller can explain
// the denial. On an allow (exit 0) it carries the non-blocking warn, audit and
// monitor findings of both policy passes so the caller can surface them.
//
// An allow that relied on one-shot command-tier bypass tokens redeems them
// before returning; when a token can no longer be redeemed (a concurrent call
// spent it first) the call is blocked instead.
func (o *SecurityOrchestrator) RunPreToolUse(ctx *policy.EvalContext) (policy.PolicyDecision, int) {
	enforceCtx := *ctx
	enforceCtx.TierFilter = policy.EnforceAlwaysOnly
	enforceDecision, code := o.safeEvalPolicy(&enforceCtx)
	if code != 0 {
		return enforceDecision, code
	}

	var consumed []string
	if strings.HasPrefix(ctx.ToolName, "mcp__") && o.trust != nil {
		// The policy evaluator resolves the session's bypass overrides into the
		// context it evaluates (enforceCtx starts as a copy of ctx), so the
		// deputy check honors exactly the overrides the policy passes see.
		deputy, code := o.safeConfusedDeputyCheck(ctx, enforceCtx.Overrides)
		if code != 0 {
			deputy.Findings = append(enforceDecision.Findings, deputy.Findings...)
			return deputy, code
		}
		consumed = deputy.ConsumedTokens
	}

	sessionCtx := *ctx
	sessionCtx.TierFilter = policy.SessionCommandOnly
	sessionDecision, code := o.safeEvalPolicy(&sessionCtx)
	sessionDecision.Findings = append(enforceDecision.Findings, sessionDecision.Findings...)
	if code != 0 {
		return sessionDecision, code
	}

	if sessionDecision.Action == "" {
		sessionDecision.Action = enforceDecision.Action
		sessionDecision.RuleID = enforceDecision.RuleID
		sessionDecision.Message = enforceDecision.Message
	}

	for _, id := range sessionDecision.ConsumedTokens {
		if !slices.Contains(consumed, id) {
			consumed = append(consumed, id)
		}
	}
	sessionDecision.ConsumedTokens = consumed
	if err := o.policy.ConsumeCommandTokens(ctx, consumed); err != nil {
		return policy.PolicyDecision{
			Action:   policy.Block,
			ExitCode: 2,
			RuleID:   consumed[0],
			Message:  fmt.Sprintf("the one-shot bypass for this call could not be redeemed, so the rule applies again: %v", err),
			Err:      err,
			Findings: sessionDecision.Findings,
		}, 2
	}
	return sessionDecision, 0
}

func (o *SecurityOrchestrator) safeEvalPolicy(ctx *policy.EvalContext) (decision policy.PolicyDecision, exitCode int) {
	defer func() {
		if r := recover(); r != nil {
			exitCode = 2
			decision = policy.PolicyDecision{
				Action:   policy.Block,
				ExitCode: 2,
				Message:  fmt.Sprintf("policy engine panic: %v", r),
			}
		}
	}()

	d := o.policy.Evaluate(ctx)
	if d.Action == policy.Block {
		return d, 2
	}
	return d, 0
}

func (o *SecurityOrchestrator) RunPostToolUse(ctx *policy.EvalContext, output string) (string, int) {
	if strings.HasPrefix(ctx.ToolName, "mcp__") && o.trust != nil {
		serverName := extractServerName(ctx.ToolName)
		tier := o.resolveServerTier(serverName)
		output = o.trust.ApplyHardening(serverName, tier, output)
	}

	return output, 0
}

// PostureSnapshot summarizes the loaded policy. RulesTotal counts every loaded
// rule, including disabled ones; RulesActive counts only rules that enforce
// (enabled and not in monitor mode), so a policy whose rules are all disabled
// or monitor-only reports zero active rules. MonitorModeCount counts enabled
// monitor-only rules. The bypass-tier distribution and category coverage
// describe the enforcing rules only, so monitor-only categories are not
// reported as covered.
//
// The package-risk and MCP-trust postures are nil rather than fabricated: the
// orchestrator holds no dependency inventory to score, and no caller reports
// an MCP-trust posture yet.
func (o *SecurityOrchestrator) PostureSnapshot() (*sarif.PolicyPosture, *sarif.PackageRiskPosture, *sarif.McpTrustPosture) {
	posture := &sarif.PolicyPosture{
		BypassTierSummary: make(map[string]int),
		CategoryCoverage:  make(map[string]bool),
	}

	rules := o.policy.CurrentRules()
	posture.RulesTotal = len(rules)

	for _, r := range rules {
		if !r.Rule.IsEnabled() {
			continue
		}
		if r.Rule.MonitorMode {
			posture.MonitorModeCount++
			continue
		}
		posture.RulesActive++
		posture.BypassTierSummary[r.Rule.BypassTier.String()]++
		posture.CategoryCoverage[r.Rule.Category] = true
	}

	return posture, nil, nil
}

// safeConfusedDeputyCheck blocks an MCP call that would reach a path a
// blocking policy rule denies. Deny rules lifted by an active session grant
// are skipped. Rules lifted by a command-tier token are skipped too, and the
// tokens of those that would have blocked this call are reported in the
// decision's ConsumedTokens for the caller to redeem.
func (o *SecurityOrchestrator) safeConfusedDeputyCheck(ctx *policy.EvalContext, overrides policy.ActiveOverrides) (decision policy.PolicyDecision, exitCode int) {
	defer func() {
		if r := recover(); r != nil {
			exitCode = 2
			decision = policy.PolicyDecision{
				Action:   policy.Block,
				ExitCode: 2,
				RuleID:   ConfusedDeputyRuleID,
				Message:  fmt.Sprintf("confused-deputy check panic: %v", r),
			}
		}
	}()

	var denyRules, tokenLifted []policy.DenyRule
	for _, rule := range o.policy.FilePathDenyRules() {
		switch {
		case rule.SessionBypassed(overrides):
		case rule.CommandTokenHeld(overrides):
			tokenLifted = append(tokenLifted, rule)
		default:
			denyRules = append(denyRules, rule)
		}
	}

	blocked, reason := o.trust.CheckAccess(ctx.ToolName, ctx.ToolInput, denyRules)
	if blocked {
		return policy.PolicyDecision{
			Action:   policy.Block,
			ExitCode: 2,
			RuleID:   ConfusedDeputyRuleID,
			Message:  reason,
		}, 2
	}

	var consumed []string
	for _, rule := range tokenLifted {
		if hit, _ := o.trust.CheckAccess(ctx.ToolName, ctx.ToolInput, []policy.DenyRule{rule}); hit &&
			!slices.Contains(consumed, rule.RuleID) {
			consumed = append(consumed, rule.RuleID)
		}
	}
	return policy.PolicyDecision{ConsumedTokens: consumed}, 0
}

// resolveServerTier scores a server from its configured definition, enriched
// with the known-server database when the configured command matches.
func (o *SecurityOrchestrator) resolveServerTier(serverName string) trust.TrustTier {
	var configured *trust.McpServerInfo
	if def, ok := o.servers[serverName]; ok {
		configured = &def
	}
	info := trust.ResolveServerInfo(serverName, configured)
	return o.trust.ScoreServer(&info).Tier
}

func extractServerName(toolName string) string {
	trimmed := strings.TrimPrefix(toolName, "mcp__")
	if name, _, ok := strings.Cut(trimmed, "__"); ok {
		return name
	}
	return trimmed
}
