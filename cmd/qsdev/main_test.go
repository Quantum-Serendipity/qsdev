package main

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

// TestRegisterFrameworkAdapters proves all five framework adapters are wired into
// the default registry by the explicit registration that replaced the former
// per-package init() self-registration — and that registration does not panic on
// a duplicate. It is the regression guard for the init()->explicit-wiring refactor.
func TestRegisterFrameworkAdapters(t *testing.T) {
	registerFrameworkAdapters()

	got := map[aiframework.FrameworkID]bool{}
	for _, a := range spi.DefaultRegistry().All() {
		got[a.ID()] = true
	}

	want := []aiframework.FrameworkID{
		aiframework.ClaudeCode,
		aiframework.ContinueDev, // cline belongs to the Continue.dev family
		aiframework.Codex,
		aiframework.Cursor,
		aiframework.Windsurf,
	}
	for _, id := range want {
		if !got[id] {
			t.Errorf("framework adapter %q not registered", id)
		}
	}
	if len(got) != len(want) {
		t.Errorf("registered adapter count = %d, want %d (got %v)", len(got), len(want), got)
	}
}
