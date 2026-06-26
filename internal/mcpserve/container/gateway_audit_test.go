package container

import (
	"context"
	"sync"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// spyAuditSink records every AuditRecord it receives. It is safe for concurrent
// use because mcp-go dispatches tool calls concurrently.
type spyAuditSink struct {
	mu      sync.Mutex
	records []middleware.AuditRecord
}

func (s *spyAuditSink) Record(_ context.Context, rec middleware.AuditRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, rec)
}

func (s *spyAuditSink) all() []middleware.AuditRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]middleware.AuditRecord, len(s.records))
	copy(out, s.records)
	return out
}

// TestGatewayInterceptorOrder pins the interceptor between Audit (15) and
// Guardrail (20): inside Audit so denials are recorded, outside Guardrail so
// allow-list authorization runs before policy/rate-limit/content-safety.
func TestGatewayInterceptorOrder(t *testing.T) {
	t.Parallel()
	got := NewGatewayInterceptor(nil, false).Order()
	if got != 16 {
		t.Fatalf("GatewayInterceptor.Order() = %d, want 16 (between Audit=15 and Guardrail=20)", got)
	}
}

// TestGatewayDenialAuditedAsDenied is the R4 assertion: an authenticated (cert)
// identity NOT in the allow-list is denied by the gateway authorization layer,
// and because that layer now sits INSIDE Audit, the audit record classifies the
// outcome as DecisionDenied — not a generic error.
func TestGatewayDenialAuditedAsDenied(t *testing.T) {
	t.Parallel()

	t.Run("denied identity recorded as denied", func(t *testing.T) {
		t.Parallel()
		spy := &spyAuditSink{}
		chain := GatewayChain(GatewayOptions{
			AllowedAgents: []string{"trusted-cn"},
			RequireAuth:   true,
			AuditSink:     spy,
		})

		ran := false
		final := func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
			ran = true
			return &spi.ToolResult{Text: "ok"}, nil
		}

		res, err := chain.Execute(context.Background(), authedCtx("intruder-cn", "general"),
			&spi.ToolRequest{Name: "qsdev_status"}, final)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if ran {
			t.Error("handler ran for an un-allow-listed identity")
		}
		if res == nil || !res.IsError {
			t.Fatalf("result = %+v, want an IsError denial", res)
		}

		recs := spy.all()
		if len(recs) != 1 {
			t.Fatalf("audit records = %d, want exactly 1", len(recs))
		}
		if recs[0].Decision != middleware.DecisionDenied {
			t.Errorf("audit Decision = %q, want %q", recs[0].Decision, middleware.DecisionDenied)
		}
		if recs[0].AgentID != "intruder-cn" {
			t.Errorf("audit AgentID = %q, want intruder-cn", recs[0].AgentID)
		}
		if !recs[0].IsError {
			t.Error("audit record IsError = false, want true for a denial")
		}
	})

	t.Run("allowed identity recorded as allowed", func(t *testing.T) {
		t.Parallel()
		spy := &spyAuditSink{}
		chain := GatewayChain(GatewayOptions{
			AllowedAgents: []string{"trusted-cn"},
			RequireAuth:   true,
			AuditSink:     spy,
		})
		final := func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
			return &spi.ToolResult{Text: "ok"}, nil
		}
		res, err := chain.Execute(context.Background(), authedCtx("trusted-cn", "general"),
			&spi.ToolRequest{Name: "qsdev_status"}, final)
		if err != nil {
			t.Fatalf("Execute: %v", err)
		}
		if res == nil || res.IsError {
			t.Fatalf("result = %+v, want a successful pass-through", res)
		}
		recs := spy.all()
		if len(recs) != 1 || recs[0].Decision != middleware.DecisionAllowed {
			t.Fatalf("audit records = %+v, want one DecisionAllowed", recs)
		}
	})
}
