package spi

import (
	"context"
	"sort"
)

// Middleware wraps tool execution with a cross-cutting concern (auth, logging,
// rate-limiting, etc.). Middlewares are ordered by Order(): within a Chain the
// lowest Order() is the OUTERMOST wrapper (it sees the request first and the
// response last), and the highest Order() is the INNERMOST wrapper (closest to
// the real handler).
type Middleware interface {
	// Order returns this middleware's position in the chain. Lower values are
	// applied further out. Ties keep their relative registration order.
	Order() int
	// Handle processes the request and must call next to continue the chain
	// (or deliberately short-circuit by returning without calling next).
	Handle(ctx context.Context, cc *ToolCallContext, req *ToolRequest, next ToolHandler) (*ToolResult, error)
}

// Chain is an ordered collection of Middleware that wraps a final ToolHandler.
type Chain struct {
	mws []Middleware
}

// NewChain creates a Chain from the given middlewares. They may be supplied in
// any order; Execute sorts them by Order() at call time.
func NewChain(mws ...Middleware) *Chain {
	cp := make([]Middleware, len(mws))
	copy(cp, mws)
	return &Chain{mws: cp}
}

// Len reports the number of middlewares in the chain.
func (c *Chain) Len() int { return len(c.mws) }

// With returns a NEW Chain containing this chain's middlewares plus extra. The
// receiver is not modified, so a shared base chain can be safely extended for
// different deployment modes. Execute still orders every middleware by Order()
// at call time, so an added middleware with a lower Order() than any existing
// one becomes the new outermost layer regardless of its append position. The
// gateway deployment uses this to wrap the built-in chain with an outer
// authentication layer.
func (c *Chain) With(extra ...Middleware) *Chain {
	combined := make([]Middleware, 0, len(c.mws)+len(extra))
	combined = append(combined, c.mws...)
	combined = append(combined, extra...)
	return &Chain{mws: combined}
}

// Execute runs final wrapped by every middleware. The chain is ordered
// ascending by Order(): the lowest Order() ends up outermost. Sorting is stable
// so middlewares sharing an Order() retain their registration order.
func (c *Chain) Execute(ctx context.Context, cc *ToolCallContext, req *ToolRequest, final ToolHandler) (*ToolResult, error) {
	if final == nil {
		return nil, errNilHandler
	}
	if len(c.mws) == 0 {
		return final(ctx, cc, req)
	}

	ordered := make([]Middleware, len(c.mws))
	copy(ordered, c.mws)
	sort.SliceStable(ordered, func(i, j int) bool {
		return ordered[i].Order() < ordered[j].Order()
	})

	// Build from the innermost (highest Order, last in slice) outward so the
	// lowest Order ends up as the outermost call.
	next := final
	for i := len(ordered) - 1; i >= 0; i-- {
		mw := ordered[i]
		inner := next
		next = func(ctx context.Context, cc *ToolCallContext, req *ToolRequest) (*ToolResult, error) {
			return mw.Handle(ctx, cc, req, inner)
		}
	}
	return next(ctx, cc, req)
}
