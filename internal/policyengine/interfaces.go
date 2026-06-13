package policyengine

import (
	"encoding/json"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/risk"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/trust"
)

type PolicyEvaluator interface {
	Evaluate(ctx *policy.EvalContext) policy.PolicyDecision
	FilePathDenyRules() []policy.DenyRule
	CurrentRules() []policy.CompiledRule
}

type PackageRiskScorer interface {
	ScorePackage(info *risk.PackageInfo) risk.PackageScore
	ScoreAll(packages []risk.PackageInfo) risk.DependencyHealth
}

type McpTrustEvaluator interface {
	CheckAccess(toolName string, toolArgs json.RawMessage, denyRules []policy.DenyRule) (blocked bool, reason string)
	ApplyHardening(serverName string, tier trust.TrustTier, output string) string
	ScoreServer(info *trust.McpServerInfo) trust.TrustScore
}
