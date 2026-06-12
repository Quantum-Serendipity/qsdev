package policyengine

import (
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/sarif"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/trust"
)

type SecurityOrchestrator struct {
	policy PolicyEvaluator
	risk   PackageRiskScorer
	trust  McpTrustEvaluator
}

func NewSecurityOrchestrator(p PolicyEvaluator, r PackageRiskScorer, t McpTrustEvaluator) *SecurityOrchestrator {
	return &SecurityOrchestrator{
		policy: p,
		risk:   r,
		trust:  t,
	}
}

func (o *SecurityOrchestrator) RunPreToolUse(ctx *policy.EvalContext) int {
	enforceCtx := *ctx
	enforceCtx.TierFilter = policy.EnforceAlwaysOnly
	if _, code := o.safeEvalPolicy(&enforceCtx); code != 0 {
		return code
	}

	if strings.HasPrefix(ctx.ToolName, "mcp__") && o.trust != nil {
		if code := o.safeConfusedDeputyCheck(ctx); code != 0 {
			return code
		}
	}

	sessionCtx := *ctx
	sessionCtx.TierFilter = policy.SessionCommandOnly
	if _, code := o.safeEvalPolicy(&sessionCtx); code != 0 {
		return code
	}

	return 0
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

func (o *SecurityOrchestrator) PostureSnapshot() (*sarif.PolicyPosture, *sarif.PackageRiskPosture, *sarif.McpTrustPosture) {
	posture := &sarif.PolicyPosture{
		BypassTierSummary: make(map[string]int),
		CategoryCoverage:  make(map[string]bool),
	}

	rules := o.policy.CurrentRules()
	posture.RulesTotal = len(rules)

	for _, r := range rules {
		if r.Rule.IsEnabled() {
			posture.RulesActive++
		}
		if r.Rule.MonitorMode {
			posture.MonitorModeCount++
		}
		posture.BypassTierSummary[r.Rule.BypassTier.String()]++
		posture.CategoryCoverage[r.Rule.Category] = true
	}

	return posture, nil, nil
}

func (o *SecurityOrchestrator) safeConfusedDeputyCheck(ctx *policy.EvalContext) (exitCode int) {
	defer func() {
		if r := recover(); r != nil {
			exitCode = 2
		}
	}()

	denyRules := o.policy.FilePathDenyRules()
	blocked, _ := o.trust.CheckAccess(ctx.ToolName, ctx.ToolInput, denyRules)
	if blocked {
		return 2
	}
	return 0
}

func (o *SecurityOrchestrator) resolveServerTier(serverName string) trust.TrustTier {
	info := &trust.McpServerInfo{Name: serverName}
	score := o.trust.ScoreServer(info)
	return score.Tier
}

func extractServerName(toolName string) string {
	trimmed := strings.TrimPrefix(toolName, "mcp__")
	if name, _, ok := strings.Cut(trimmed, "__"); ok {
		return name
	}
	return trimmed
}
