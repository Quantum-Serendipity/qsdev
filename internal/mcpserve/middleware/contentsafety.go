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

	red := cs.redactor
	if red == nil {
		red = logging.NewRedactor()
	}

	res.Text = red.RedactString(res.Text)
	if res.Structured != nil {
		res.Structured = redactValue(red, res.Structured)
	}
	return res, nil
}

// redactValue walks a JSON-shaped value (maps, slices, strings) and redacts any
// string it finds. Non-string, non-container leaves are returned unchanged.
func redactValue(red *logging.Redactor, v any) any {
	switch t := v.(type) {
	case string:
		return red.RedactString(t)
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = redactValue(red, val)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = redactValue(red, val)
		}
		return out
	default:
		return v
	}
}
