package spi

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

// recordingMW records entry/exit around the next call so tests can assert the
// nesting order produced by a Chain.
type recordingMW struct {
	order  int
	label  string
	events *[]string
}

func (m recordingMW) Order() int { return m.order }

func (m recordingMW) Handle(ctx context.Context, cc *ToolCallContext, req *ToolRequest, next ToolHandler) (*ToolResult, error) {
	*m.events = append(*m.events, "enter:"+m.label)
	res, err := next(ctx, cc, req)
	*m.events = append(*m.events, "exit:"+m.label)
	return res, err
}

func TestChainExecuteOrdering(t *testing.T) {
	t.Parallel()

	var events []string
	// Registered out of Order() sequence on purpose: 30, 10, 20.
	chain := NewChain(
		recordingMW{order: 30, label: "c", events: &events},
		recordingMW{order: 10, label: "a", events: &events},
		recordingMW{order: 20, label: "b", events: &events},
	)

	final := func(_ context.Context, _ *ToolCallContext, _ *ToolRequest) (*ToolResult, error) {
		events = append(events, "handler")
		return &ToolResult{Text: "ok"}, nil
	}

	res, err := chain.Execute(context.Background(), &ToolCallContext{}, &ToolRequest{}, final)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if res == nil || res.Text != "ok" {
		t.Fatalf("Execute result = %+v, want Text=ok", res)
	}

	// Lowest Order() is outermost: a wraps b wraps c wraps handler.
	want := []string{
		"enter:a", "enter:b", "enter:c",
		"handler",
		"exit:c", "exit:b", "exit:a",
	}
	if !reflect.DeepEqual(events, want) {
		t.Errorf("event order = %v, want %v", events, want)
	}
}

func TestChainExecuteEmpty(t *testing.T) {
	t.Parallel()

	called := false
	final := func(_ context.Context, _ *ToolCallContext, _ *ToolRequest) (*ToolResult, error) {
		called = true
		return &ToolResult{Text: "x"}, nil
	}
	res, err := NewChain().Execute(context.Background(), &ToolCallContext{}, &ToolRequest{}, final)
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if !called || res.Text != "x" {
		t.Errorf("empty chain did not invoke final handler correctly: called=%v res=%+v", called, res)
	}
}

func TestChainExecuteNilHandler(t *testing.T) {
	t.Parallel()

	_, err := NewChain().Execute(context.Background(), &ToolCallContext{}, &ToolRequest{}, nil)
	if !errors.Is(err, errNilHandler) {
		t.Errorf("Execute with nil handler error = %v, want errNilHandler", err)
	}
}

func TestChainStableOrderOnTie(t *testing.T) {
	t.Parallel()

	var events []string
	// Two middlewares share Order() 10; stable sort must preserve registration
	// order (first -> outermost).
	chain := NewChain(
		recordingMW{order: 10, label: "first", events: &events},
		recordingMW{order: 10, label: "second", events: &events},
	)
	final := func(_ context.Context, _ *ToolCallContext, _ *ToolRequest) (*ToolResult, error) {
		return &ToolResult{}, nil
	}
	if _, err := chain.Execute(context.Background(), &ToolCallContext{}, &ToolRequest{}, final); err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	want := []string{"enter:first", "enter:second", "exit:second", "exit:first"}
	if !reflect.DeepEqual(events, want) {
		t.Errorf("tie order = %v, want %v", events, want)
	}
}
