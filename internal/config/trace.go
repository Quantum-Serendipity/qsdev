package config

import (
	"fmt"
	"io"
)

// ResolutionTrace records a single field's value change during config resolution.
type ResolutionTrace struct {
	Field    string
	Value    any
	Source   string
	Previous any
	Note     string
}

// ResolutionTracer collects resolution traces when enabled. When disabled,
// Record is a no-op to avoid allocation overhead.
type ResolutionTracer struct {
	enabled bool
	traces  []ResolutionTrace
}

// NewTracer returns a tracer that records traces only when enabled is true.
func NewTracer(enabled bool) *ResolutionTracer {
	return &ResolutionTracer{enabled: enabled}
}

// Record appends a trace entry. It is a no-op when the tracer is disabled.
func (t *ResolutionTracer) Record(field string, value any, source string, prev any, note string) {
	if !t.enabled {
		return
	}
	t.traces = append(t.traces, ResolutionTrace{
		Field:    field,
		Value:    value,
		Source:   source,
		Previous: prev,
		Note:     note,
	})
}

// Traces returns the collected traces.
func (t *ResolutionTracer) Traces() []ResolutionTrace {
	return t.traces
}

// FormatTraces writes a human-readable table of resolution traces to w.
func FormatTraces(w io.Writer, traces []ResolutionTrace) {
	if len(traces) == 0 {
		return
	}

	fmt.Fprintf(w, "%-30s %-15s %-20s %-20s %s\n", "FIELD", "SOURCE", "VALUE", "PREVIOUS", "NOTE")
	fmt.Fprintf(w, "%-30s %-15s %-20s %-20s %s\n", "-----", "------", "-----", "--------", "----")

	for _, tr := range traces {
		prev := fmt.Sprintf("%v", tr.Previous)
		if tr.Previous == nil {
			prev = "<nil>"
		}
		val := fmt.Sprintf("%v", tr.Value)
		fmt.Fprintf(w, "%-30s %-15s %-20s %-20s %s\n", tr.Field, tr.Source, val, prev, tr.Note)
	}
}
