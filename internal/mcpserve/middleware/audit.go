package middleware

import (
	"context"
	"log/slog"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// AuditRecord is the structured record of a single tool-call decision.
type AuditRecord struct {
	// AgentID is the resolved caller identity.
	AgentID string
	// Tool is the tool name invoked.
	Tool string
	// Category is the tool's taxonomy category (may be empty).
	Category string
	// Decision is the resolved outcome (allowed / error / denied / rate_limited
	// / panic).
	Decision Decision
	// Duration is the wall-clock time spent in the inner chain (everything below
	// Audit, i.e. Guardrail onward).
	Duration time.Duration
	// IsError reports whether the surfaced result was a tool-level error.
	IsError bool
}

// AuditSink consumes audit records. Implementations must be safe for concurrent
// use; tool calls are dispatched concurrently by mcp-go.
type AuditSink interface {
	Record(ctx context.Context, rec AuditRecord)
}

// slogSink is the default AuditSink: it emits each record as a structured log
// line at info level to a *slog.Logger (stderr in the running server, since the
// stdio transport reserves stdout for the protocol).
type slogSink struct{ logger *slog.Logger }

// NewSlogSink returns an AuditSink that writes records to logger. A nil logger
// falls back to slog.Default(), which the server configures to stderr.
func NewSlogSink(logger *slog.Logger) AuditSink {
	if logger == nil {
		logger = slog.Default()
	}
	return slogSink{logger: logger}
}

func (s slogSink) Record(ctx context.Context, rec AuditRecord) {
	s.logger.LogAttrs(ctx, slog.LevelInfo, "mcp tool call",
		slog.String("agent_id", rec.AgentID),
		slog.String("tool", rec.Tool),
		slog.String("category", rec.Category),
		slog.String("decision", string(rec.Decision)),
		slog.Duration("duration", rec.Duration),
		slog.Bool("is_error", rec.IsError),
	)
}

// Audit is the audit layer (Order 15). It sits OUTSIDE Guardrail and RateLimit
// (correction C9) so that calls denied by Guardrail or throttled by RateLimit
// are STILL recorded. It installs an outcome carrier on the context before
// calling inward, times the inner chain, then records the resolved decision —
// reading the precise short-circuit reason from the carrier when an inner layer
// set one, and otherwise inferring allowed/error from the result.
type Audit struct {
	sink  AuditSink
	clock func() time.Time
}

// Order returns 15.
func (Audit) Order() int { return orderAudit }

// Handle records the call decision and duration around the inner chain.
func (a Audit) Handle(ctx context.Context, cc *spi.ToolCallContext, req *spi.ToolRequest, next spi.ToolHandler) (*spi.ToolResult, error) {
	ctx, oc := withOutcome(ctx)

	start := a.now()
	res, err := next(ctx, cc, req)
	dur := a.now().Sub(start)

	rec := AuditRecord{
		AgentID:  agentID(cc),
		Tool:     toolName(cc, req),
		Category: category(cc),
		Decision: resolveDecision(oc, res, err),
		Duration: dur,
		IsError:  res != nil && res.IsError,
	}
	if a.sink != nil {
		a.sink.Record(ctx, rec)
	}
	return res, err
}

func (a Audit) now() time.Time {
	if a.clock != nil {
		return a.clock()
	}
	return time.Now()
}

// resolveDecision prefers the explicit decision an inner layer recorded on the
// carrier; absent that it infers from the result and error.
func resolveDecision(oc *outcome, res *spi.ToolResult, err error) Decision {
	if oc != nil && oc.set {
		return oc.decision
	}
	if err != nil {
		return DecisionError
	}
	if res != nil && res.IsError {
		return DecisionError
	}
	return DecisionAllowed
}

// agentID, toolName, and category read request metadata defensively (cc may be
// nil in isolated chain tests).
func agentID(cc *spi.ToolCallContext) string {
	if cc == nil {
		return ""
	}
	return cc.AgentID
}

func toolName(cc *spi.ToolCallContext, req *spi.ToolRequest) string {
	if cc != nil && cc.ToolName != "" {
		return cc.ToolName
	}
	if req != nil {
		return req.Name
	}
	return ""
}

func category(cc *spi.ToolCallContext) string {
	if cc == nil {
		return ""
	}
	return cc.Category
}
