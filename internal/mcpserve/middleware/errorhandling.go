package middleware

import (
	"context"
	"errors"
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// ErrorHandling is the innermost built-in layer (Order 50), wrapping the real
// tool handler. It converts a Go error returned by the handler into a clean
// tool-level error result (IsError) so tool failures surface as tool errors
// rather than JSON-RPC protocol errors — EXCEPT context cancellation/deadline
// errors, which propagate so the transport can unwind the request. It also
// defensively recovers a handler panic into a tool-error result (belt-and-
// suspenders alongside mcp-go's server-level WithRecovery).
type ErrorHandling struct{}

// Order returns 50.
func (ErrorHandling) Order() int { return orderErrorHandling }

// Handle invokes next with panic recovery and error normalization. Named returns
// let the deferred recover replace the result.
func (ErrorHandling) Handle(ctx context.Context, cc *spi.ToolCallContext, req *spi.ToolRequest, next spi.ToolHandler) (res *spi.ToolResult, err error) {
	defer func() {
		if r := recover(); r != nil {
			markDecision(ctx, DecisionPanic)
			res = &spi.ToolResult{
				IsError: true,
				Text:    fmt.Sprintf("tool %q panicked: %v", toolName(cc, req), r),
			}
			err = nil
		}
	}()

	res, err = next(ctx, cc, req)
	if err != nil {
		// Cancellation and deadline are transport-level signals, not tool
		// failures: propagate them so the request unwinds correctly.
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			return nil, err
		}
		return &spi.ToolResult{
			IsError: true,
			Text:    fmt.Sprintf("tool %q failed: %v", toolName(cc, req), err),
		}, nil
	}
	return res, nil
}
