package middleware

import (
	"context"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// TestContextInjectionPublishesContext proves the resolved *ToolCallContext is
// retrievable via spi.FromContext inside the inner handler.
func TestContextInjectionPublishesContext(t *testing.T) {
	t.Parallel()

	ci := ContextInjection{}
	cc := &spi.ToolCallContext{AgentID: "bot", ToolName: "grep", Category: CategorySearch}

	var seen *spi.ToolCallContext
	var present bool
	handler := func(ctx context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
		seen, present = spi.FromContext(ctx)
		return &spi.ToolResult{Text: "ok"}, nil
	}

	if _, err := ci.Handle(context.Background(), cc, &spi.ToolRequest{Name: "grep"}, handler); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !present {
		t.Fatal("ToolCallContext was not published onto ctx")
	}
	if seen != cc {
		t.Errorf("FromContext returned %+v, want the injected cc %+v", seen, cc)
	}
}

// TestContextInjectionNilCallContext confirms a nil cc does not panic and the
// chain still proceeds.
func TestContextInjectionNilCallContext(t *testing.T) {
	t.Parallel()

	var ran bool
	_, err := ContextInjection{}.Handle(context.Background(), nil, &spi.ToolRequest{}, okHandler(&ran, "ok"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ran {
		t.Error("handler did not run with nil cc")
	}
}
