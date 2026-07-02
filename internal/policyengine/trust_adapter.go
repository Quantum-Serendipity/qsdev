package policyengine

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/policy"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/risk"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/trust"
	"github.com/Quantum-Serendipity/qsdev/internal/policyengine/trust/hardening"
)

// Compile-time assertions that the production adapters satisfy the orchestrator
// interfaces. McpTrustEngine implements only ScoreServer; TrustAdapter adds the
// CheckAccess and ApplyHardening methods the interface requires so the trust
// subsystem can finally be wired non-nil into the orchestrator.
var (
	_ McpTrustEvaluator = (*TrustAdapter)(nil)
	_ PackageRiskScorer = risk.Scorer{}
)

// TrustAdapter adapts *trust.McpTrustEngine to the McpTrustEvaluator interface.
// The engine only provides ScoreServer; the confused-deputy check lives as a
// package-level function (trust.CheckAccess) and output hardening lives as a set
// of primitives in trust/hardening. TrustAdapter binds all three into a single
// value that can be passed to NewSecurityOrchestrator.
type TrustAdapter struct {
	engine *trust.McpTrustEngine
}

// NewTrustAdapter wraps an McpTrustEngine so it satisfies McpTrustEvaluator.
func NewTrustAdapter(engine *trust.McpTrustEngine) *TrustAdapter {
	return &TrustAdapter{engine: engine}
}

// CheckAccess delegates to the package-level confused-deputy check, which blocks
// an MCP tool call whose file-path argument matches a deny rule aimed at a
// first-party tool (path laundering, e.g. mcp__filesystem__write_file -> ~/.ssh).
func (a *TrustAdapter) CheckAccess(toolName string, toolArgs json.RawMessage, denyRules []policy.DenyRule) (bool, string) {
	return trust.CheckAccess(toolName, toolArgs, denyRules)
}

// ScoreServer delegates to the wrapped engine's trust scorer.
func (a *TrustAdapter) ScoreServer(info *trust.McpServerInfo) trust.TrustScore {
	return a.engine.ScoreServer(info)
}

// ApplyHardening composes the trust/hardening primitives (Sanitize, Datamark,
// Frame) into a tier-scaled transformation of MCP tool output before it enters
// the agent context. Lower trust tiers receive stronger hardening: Tier 1 (local,
// trusted) is only structurally framed, Tier 2 is datamarked and framed, and the
// fallback/untrusted tier additionally runs a strict injection scan whose
// detections are surfaced as an inline warning.
func (a *TrustAdapter) ApplyHardening(serverName string, tier trust.TrustTier, output string) string {
	source := "mcp://" + serverName
	hardened := output

	switch tier {
	case trust.Tier1Local:
		hardened = hardening.Frame(hardened, serverName, int(tier), source)
	case trust.Tier2Enterprise:
		hardened = hardening.Datamark(hardened)
		hardened = hardening.Frame(hardened, serverName, int(tier), source)
	default:
		if res := hardening.Sanitize(hardened, hardening.StrictMode); res.Detections > 0 {
			hardened = injectionWarning(res.Patterns) + hardened
		}
		hardened = hardening.Datamark(hardened)
		hardened = hardening.Frame(hardened, serverName, int(tier), source)
	}

	return hardened
}

func injectionWarning(patterns []string) string {
	return fmt.Sprintf("[qsdev:warning] potential prompt injection detected (%s)\n", strings.Join(patterns, ", "))
}
