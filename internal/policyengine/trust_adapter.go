package policyengine

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/contentsign"
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

// ApplyHardening composes the output hardening primitives (hardening.Sanitize,
// contentsign datamarking, hardening.Frame) into a tier-scaled transformation
// of MCP tool output before it enters the agent context. Lower trust tiers
// receive stronger hardening: Tier 1 (local, trusted) is only structurally
// framed, Tier 2 is datamarked and framed, and the fallback/untrusted tier
// additionally runs a strict injection scan whose detections are surfaced as an
// inline warning.
func (a *TrustAdapter) ApplyHardening(serverName string, tier trust.TrustTier, output string) string {
	source := "mcp://" + serverName
	hardened := output

	switch tier {
	case trust.Tier1Local:
		hardened = hardening.Frame(hardened, serverName, int(tier), source)
	case trust.Tier2Enterprise:
		hardened = datamark(source, hardened)
		hardened = hardening.Frame(hardened, serverName, int(tier), source)
	default:
		// The warning is qsdev's own annotation, so it goes outside the
		// datamarked (untrusted) region where the model can read it as is.
		var warning string
		if res := hardening.Sanitize(hardened, hardening.StrictMode); res.Detections > 0 {
			warning = injectionWarning(res.Patterns)
		}
		hardened = warning + datamark(source, hardened)
		hardened = hardening.Frame(hardened, serverName, int(tier), source)
	}

	return hardened
}

// datamark applies the contentsign datamarking transform to MCP tool output:
// prose whitespace is replaced with a randomized Private Use Area marker rune so
// injected instructions stop reading as instructions, code is preserved, a tool
// result that is a JSON document stays parseable (only its strings are marked),
// and the result is wrapped in framing that tells the model what the marker
// means. The framing's delimiters cannot be forged by the output: contentsign
// neutralizes any body line that would match one.
func datamark(source, output string) string {
	sum := sha256.Sum256([]byte(output))
	opts := contentsign.DefaultDatamarkOptions()
	opts.Source = source
	opts.VerificationStatus = contentsign.StatusUnverified
	opts.ContentHashPrefix = hex.EncodeToString(sum[:6])
	marked, _ := contentsign.Datamark(output, opts)
	return marked
}

func injectionWarning(patterns []string) string {
	return fmt.Sprintf("[qsdev:warning] potential prompt injection detected (%s)\n", strings.Join(patterns, ", "))
}
