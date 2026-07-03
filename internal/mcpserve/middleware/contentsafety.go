package middleware

import (
	"context"

	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// ContentSafety is the L1 sanitization layer (Order 45). It POST-processes the
// result returned by the inner chain, redacting secret patterns (AWS access
// keys, GitHub/GitLab/Slack/Stripe/npm tokens, JWTs, private keys, URL
// credentials, ...) from res.Text and, recursively, from JSON-serializable
// structured content. Redaction reuses internal/logging.Redactor so the MCP
// surface and the log surface share one battle-tested pattern set rather than
// maintaining a divergent copy. It never short-circuits and is nil-safe.
type ContentSafety struct {
	redactor *logging.Redactor
}

// Order returns 45.
func (ContentSafety) Order() int { return orderContentSafety }

// Handle continues the chain, then sanitizes the returned result. Go errors and
// nil results are passed through untouched (ErrorHandling, which sits inside
// this layer, has already converted handler failures into IsError results, so a
// non-nil err here is a propagated context cancellation).
func (cs ContentSafety) Handle(ctx context.Context, cc *spi.ToolCallContext, req *spi.ToolRequest, next spi.ToolHandler) (*spi.ToolResult, error) {
	res, err := next(ctx, cc, req)
	if err != nil || res == nil {
		return res, err
	}

	// The credential-vend tool is the sole surface sanctioned to emit credentials:
	// CredentialVendToolName exists precisely to return short-lived cloud tokens
	// (AWS STS, GCP IAM, Azure MI) to the agent. Redacting its output would strip
	// the AWS-key/JWT-shaped material the tool is meant to deliver, defeating its
	// purpose. The exemption is keyed on the TRUSTED, server-registered tool
	// identity (resolved by toolName from the ToolCallContext/request, both set by
	// the bridge from the registration) — NOT on the caller-declared
	// cc.Category == CategoryCredential, which any tool or adapter could assert to
	// pass its output through un-redacted. The credential category still gets the
	// tightest rate limit and the same Guardrail/Audit layers as every other tool,
	// so this narrows only redaction, and only for the one trusted tool.
	if toolName(cc, req) == CredentialVendToolName {
		return res, nil
	}

	red := cs.redactor
	if red == nil {
		red = logging.NewRedactor()
	}

	res.Text = red.RedactString(res.Text)
	if res.Structured != nil {
		res.Structured = red.RedactStructured(res.Structured)
	}
	return res, nil
}
