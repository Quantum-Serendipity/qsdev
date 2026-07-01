package middleware

import (
	"context"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// ContextInjection is the outermost layer (Order 10). It publishes the resolved
// *spi.ToolCallContext onto the Go context via spi.WithToolCallContext so that
// nested handler code holding only a context.Context can recover the caller
// identity, project root, and tool metadata via spi.FromContext without
// re-deriving them. It never short-circuits and never mutates cc.
type ContextInjection struct{}

// Order returns 10 — the outermost built-in layer.
func (ContextInjection) Order() int { return orderContextInjection }

// Handle stores cc onto ctx (when not already present) and continues the chain.
func (ContextInjection) Handle(ctx context.Context, cc *spi.ToolCallContext, req *spi.ToolRequest, next spi.ToolHandler) (*spi.ToolResult, error) {
	if cc != nil {
		if existing, ok := spi.FromContext(ctx); !ok || existing != cc {
			ctx = spi.WithToolCallContext(ctx, cc)
		}
	}
	return next(ctx, cc, req)
}
