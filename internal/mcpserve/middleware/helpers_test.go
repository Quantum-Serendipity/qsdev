package middleware

import (
	"context"
	"sync"
	"time"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// fakeClock is a manually-advanced clock for deterministic rate-limit and audit
// timing tests (no sleeping).
type fakeClock struct {
	mu sync.Mutex
	t  time.Time
}

func newFakeClock() *fakeClock {
	return &fakeClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
}

func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// recordingSink captures audit records for assertions.
type recordingSink struct {
	mu      sync.Mutex
	records []AuditRecord
}

func (s *recordingSink) Record(_ context.Context, rec AuditRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records = append(s.records, rec)
}

func (s *recordingSink) all() []AuditRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]AuditRecord, len(s.records))
	copy(out, s.records)
	return out
}

// noopSink discards audit records (used by the performance test to avoid log
// I/O skewing the measurement).
type noopSink struct{}

func (noopSink) Record(context.Context, AuditRecord) {}

// recordingMW records entry/exit so onion ordering can be asserted.
type recordingMW struct {
	order  int
	label  string
	events *[]string
}

func (m recordingMW) Order() int { return m.order }

func (m recordingMW) Handle(ctx context.Context, cc *spi.ToolCallContext, req *spi.ToolRequest, next spi.ToolHandler) (*spi.ToolResult, error) {
	*m.events = append(*m.events, "enter:"+m.label)
	res, err := next(ctx, cc, req)
	*m.events = append(*m.events, "exit:"+m.label)
	return res, err
}

// okHandler returns a successful textual result and records that it ran.
func okHandler(ran *bool, text string) spi.ToolHandler {
	return func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
		if ran != nil {
			*ran = true
		}
		return &spi.ToolResult{Text: text}, nil
	}
}
