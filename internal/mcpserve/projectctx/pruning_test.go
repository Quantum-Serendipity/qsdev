package projectctx

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// tieredTool builds a minimal registration with the given name, category, and
// tier, whose handler echoes its own name so dispatch can be verified.
func tieredTool(name, category string, tier Tier) spi.ToolRegistration {
	return spi.ToolRegistration{
		Name:     name,
		Category: category,
		Tier:     int(tier),
		Handler: func(_ context.Context, _ *spi.ToolCallContext, _ *spi.ToolRequest) (*spi.ToolResult, error) {
			return &spi.ToolResult{Text: name}, nil
		},
	}
}

func toolNames(regs []spi.ToolRegistration) map[string]bool {
	m := make(map[string]bool, len(regs))
	for _, r := range regs {
		m[r.Name] = true
	}
	return m
}

func TestPruneTierOrder(t *testing.T) {
	t.Parallel()
	p := NewToolPruner()
	tools := []spi.ToolRegistration{
		tieredTool("c1", "status", TierCritical),
		tieredTool("c2", "status", TierCritical),
		tieredTool("s1", "diagnostics", TierStandard),
		tieredTool("s2", "diagnostics", TierStandard),
		tieredTool("e1", "documentation", TierExtended),
		tieredTool("e2", "documentation", TierExtended),
	}

	t.Run("no prune when within ceiling", func(t *testing.T) {
		t.Parallel()
		got := p.Prune(tools, 10)
		if len(got) != 6 {
			t.Fatalf("len = %d, want 6", len(got))
		}
	})

	t.Run("drops extended first", func(t *testing.T) {
		t.Parallel()
		got := p.Prune(tools, 4)
		if len(got) != 4 {
			t.Fatalf("len = %d, want 4", len(got))
		}
		names := toolNames(got)
		if names["e1"] || names["e2"] {
			t.Errorf("extended tools should be pruned first, got %v", names)
		}
		if !names["c1"] || !names["c2"] {
			t.Errorf("critical tools must be retained, got %v", names)
		}
	})

	t.Run("drops standard after extended", func(t *testing.T) {
		t.Parallel()
		got := p.Prune(tools, 2)
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		names := toolNames(got)
		if !names["c1"] || !names["c2"] {
			t.Errorf("only the two critical tools should remain, got %v", names)
		}
	})
}

func TestPruneConsolidationFallback(t *testing.T) {
	t.Parallel()
	p := NewToolPruner()
	// Three critical tools all in the same category; pruning alone cannot reach a
	// ceiling of 1 without dropping a critical tool, so consolidation must fire.
	tools := []spi.ToolRegistration{
		tieredTool("a", "status", TierCritical),
		tieredTool("b", "status", TierCritical),
		tieredTool("c", "status", TierCritical),
	}
	got := p.Prune(tools, 1)
	if len(got) != 1 {
		t.Fatalf("len = %d, want 1 after consolidation", len(got))
	}
	if !strings.HasSuffix(got[0].Name, "_ops") {
		t.Errorf("consolidated tool name = %q, want a *_ops dispatcher", got[0].Name)
	}

	// The consolidated dispatcher must route to a member by name.
	res, err := got[0].Handler(context.Background(), &spi.ToolCallContext{},
		&spi.ToolRequest{Arguments: map[string]any{"tool": "b"}})
	if err != nil {
		t.Fatalf("consolidated handler error: %v", err)
	}
	if res.Text != "b" {
		t.Errorf("dispatch routed to %q, want member b", res.Text)
	}

	// An unknown member is a tool-level error, not a crash.
	bad, err := got[0].Handler(context.Background(), &spi.ToolCallContext{},
		&spi.ToolRequest{Arguments: map[string]any{"tool": "nope"}})
	if err != nil {
		t.Fatalf("unknown member should not be a Go error: %v", err)
	}
	if !bad.IsError {
		t.Errorf("unknown member should yield IsError result")
	}
}

func TestPruneCeilingFortyWithManyTools(t *testing.T) {
	t.Parallel()
	p := NewToolPruner()
	var tools []spi.ToolRegistration
	for i := 0; i < 50; i++ {
		tier := TierExtended
		switch {
		case i < 6:
			tier = TierCritical
		case i < 20:
			tier = TierStandard
		}
		tools = append(tools, tieredTool(fmt.Sprintf("t%02d", i), fmt.Sprintf("cat%d", i%5), tier))
	}
	got := p.Prune(tools, 40)
	if len(got) > 40 {
		t.Errorf("pruned len = %d, want <= 40", len(got))
	}
	// The six critical tools must survive.
	names := toolNames(got)
	for i := 0; i < 6; i++ {
		if !names[fmt.Sprintf("t%02d", i)] {
			t.Errorf("critical tool t%02d was dropped", i)
		}
	}
}

// fakeNotifier counts NotifyToolsListChanged invocations.
type fakeNotifier struct{ calls int }

func (f *fakeNotifier) NotifyToolsListChanged() { f.calls++ }

func TestApplyCeilingNotifies(t *testing.T) {
	t.Parallel()
	p := NewToolPruner()
	tools := []spi.ToolRegistration{
		tieredTool("c1", "status", TierCritical),
		tieredTool("s1", "diagnostics", TierStandard),
		tieredTool("e1", "documentation", TierExtended),
	}

	t.Run("notifies when catalog changes", func(t *testing.T) {
		t.Parallel()
		n := &fakeNotifier{}
		got := p.ApplyCeiling(n, tools, 2)
		if len(got) != 2 {
			t.Fatalf("len = %d, want 2", len(got))
		}
		if n.calls != 1 {
			t.Errorf("notifier calls = %d, want 1", n.calls)
		}
	})

	t.Run("silent when catalog unchanged", func(t *testing.T) {
		t.Parallel()
		n := &fakeNotifier{}
		got := p.ApplyCeiling(n, tools, 10)
		if len(got) != 3 {
			t.Fatalf("len = %d, want 3", len(got))
		}
		if n.calls != 0 {
			t.Errorf("notifier calls = %d, want 0", n.calls)
		}
	})
}

func TestGenericToolsTierSpread(t *testing.T) {
	t.Parallel()
	_, pc := newGoProject(t)
	tiers := map[int]bool{}
	for _, reg := range pc.Tools() {
		tiers[reg.Tier] = true
		if reg.Category == "" {
			t.Errorf("tool %q has no category; rate-limiter binding requires one", reg.Name)
		}
	}
	if !tiers[int(TierCritical)] {
		t.Errorf("expected at least one critical-tier generic tool")
	}
}
