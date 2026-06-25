package middleware

import (
	"context"
	"reflect"
	"testing"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// TestLayerOrders pins each built-in layer to its documented Order() value. The
// onion sequence asserted in TestOnionOrdering is only meaningful because these
// numbers are exactly what the real layers report.
func TestLayerOrders(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		mw   spi.Middleware
		want int
	}{
		{"context_injection", ContextInjection{}, 10},
		{"audit", Audit{}, 15},
		{"guardrail", Guardrail{}, 20},
		{"rate_limit", RateLimit{}, 30},
		{"content_safety", ContentSafety{}, 45},
		{"error_handling", ErrorHandling{}, 50},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := tc.mw.Order(); got != tc.want {
				t.Errorf("Order() = %d, want %d", got, tc.want)
			}
		})
	}

	if AddonOrderFloor <= 50 {
		t.Errorf("AddonOrderFloor = %d, must exceed the innermost built-in order (50)", AddonOrderFloor)
	}
}

// TestOnionOrdering proves the chain executes layers outer->inner by ascending
// Order(): 10,15,20,30,45,50 around the handler, unwinding in reverse. Recording
// probes carry the same Order() values the real layers report (asserted in
// TestLayerOrders).
func TestOnionOrdering(t *testing.T) {
	t.Parallel()

	var events []string
	// Registered deliberately out of order; the chain must sort by Order().
	chain := spi.NewChain(
		recordingMW{order: orderContentSafety, label: "45", events: &events},
		recordingMW{order: orderContextInjection, label: "10", events: &events},
		recordingMW{order: orderErrorHandling, label: "50", events: &events},
		recordingMW{order: orderAudit, label: "15", events: &events},
		recordingMW{order: orderRateLimit, label: "30", events: &events},
		recordingMW{order: orderGuardrail, label: "20", events: &events},
	)
	final := func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
		events = append(events, "handler")
		return &spi.ToolResult{Text: "ok"}, nil
	}

	if _, err := chain.Execute(context.Background(), &spi.ToolCallContext{}, &spi.ToolRequest{}, final); err != nil {
		t.Fatalf("Execute error: %v", err)
	}

	want := []string{
		"enter:10", "enter:15", "enter:20", "enter:30", "enter:45", "enter:50",
		"handler",
		"exit:50", "exit:45", "exit:30", "exit:20", "exit:15", "exit:10",
	}
	if !reflect.DeepEqual(events, want) {
		t.Errorf("onion order =\n %v\nwant\n %v", events, want)
	}
}

// TestDefaultChainComposition checks DefaultChain assembles exactly the six
// built-in layers.
func TestDefaultChainComposition(t *testing.T) {
	t.Parallel()
	if n := DefaultChain().Len(); n != 6 {
		t.Errorf("DefaultChain has %d layers, want 6", n)
	}
}

// TestNoOpChainPerformance asserts the full default chain adds negligible
// overhead around a no-op handler: well under the 5ms budget per call.
func TestNoOpChainPerformance(t *testing.T) {
	t.Parallel()

	// Unbounded limits and a no-op sink isolate the measurement to pure chain
	// traversal (no rate-limit short-circuits, no log I/O).
	chain := DefaultChain(
		WithAuditSink(noopSink{}),
		WithLimits(CategoryLimits{Default: Limit{Rate: 1e9, Burst: 1_000_000_000, Concurrency: 0}}),
	)
	cc := &spi.ToolCallContext{AgentID: "perf", ToolName: "noop", Category: CategoryStatus}
	final := func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
		return &spi.ToolResult{Text: "ok"}, nil
	}

	// Warm once (lazy bucket/semaphore creation) then time a batch.
	if _, err := chain.Execute(context.Background(), cc, &spi.ToolRequest{Name: "noop"}, final); err != nil {
		t.Fatalf("warmup Execute error: %v", err)
	}

	const iters = 200
	start := time.Now()
	for i := 0; i < iters; i++ {
		if _, err := chain.Execute(context.Background(), cc, &spi.ToolRequest{Name: "noop"}, final); err != nil {
			t.Fatalf("Execute error: %v", err)
		}
	}
	perCall := time.Since(start) / iters
	if perCall > 5*time.Millisecond {
		t.Errorf("per-call chain overhead = %v, want <= 5ms", perCall)
	}
}
