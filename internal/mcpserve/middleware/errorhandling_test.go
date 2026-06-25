package middleware

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

func TestErrorHandlingConvertsError(t *testing.T) {
	t.Parallel()

	eh := ErrorHandling{}
	cc := &spi.ToolCallContext{ToolName: "explode"}
	handler := func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
		return nil, errors.New("disk on fire")
	}
	res, err := eh.Handle(context.Background(), cc, &spi.ToolRequest{Name: "explode"}, handler)
	if err != nil {
		t.Fatalf("Go error should be absorbed, got %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("result = %+v, want IsError", res)
	}
	if !strings.Contains(res.Text, "disk on fire") {
		t.Errorf("error text = %q, want it to mention the underlying error", res.Text)
	}
}

func TestErrorHandlingPropagatesCancellation(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
	}{
		{"canceled", context.Canceled},
		{"deadline", context.DeadlineExceeded},
		{"wrapped canceled", fmt.Errorf("layer: %w", context.Canceled)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			handler := func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
				return nil, tc.err
			}
			res, err := ErrorHandling{}.Handle(context.Background(), &spi.ToolCallContext{}, &spi.ToolRequest{}, handler)
			if !errors.Is(err, context.Canceled) && !errors.Is(err, context.DeadlineExceeded) {
				t.Errorf("error = %v, want it to propagate cancellation/deadline", err)
			}
			if res != nil {
				t.Errorf("result = %+v, want nil on propagated cancellation", res)
			}
		})
	}
}

func TestErrorHandlingRecoversPanic(t *testing.T) {
	t.Parallel()

	handler := func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
		panic("kaboom")
	}
	res, err := ErrorHandling{}.Handle(context.Background(), &spi.ToolCallContext{ToolName: "boom"}, &spi.ToolRequest{Name: "boom"}, handler)
	if err != nil {
		t.Fatalf("panic should be recovered into a result, got err %v", err)
	}
	if res == nil || !res.IsError {
		t.Fatalf("result = %+v, want IsError after panic", res)
	}
	if !strings.Contains(res.Text, "kaboom") {
		t.Errorf("panic text = %q, want it to mention the panic value", res.Text)
	}
}

func TestErrorHandlingPassesThroughSuccess(t *testing.T) {
	t.Parallel()

	var ran bool
	res, err := ErrorHandling{}.Handle(context.Background(), &spi.ToolCallContext{}, &spi.ToolRequest{}, okHandler(&ran, "fine"))
	if err != nil || res == nil || res.IsError || res.Text != "fine" {
		t.Fatalf("success path: res=%+v err=%v", res, err)
	}
	if !ran {
		t.Error("handler did not run")
	}
}

// TestErrorHandlingMarksPanicOutcome confirms the panic decision propagates to
// an installed audit outcome carrier.
func TestErrorHandlingMarksPanicOutcome(t *testing.T) {
	t.Parallel()

	ctx, oc := withOutcome(context.Background())
	handler := func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
		panic("x")
	}
	if _, err := (ErrorHandling{}).Handle(ctx, &spi.ToolCallContext{}, &spi.ToolRequest{}, handler); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !oc.set || oc.decision != DecisionPanic {
		t.Errorf("outcome = %+v, want panic decision marked", oc)
	}
}
