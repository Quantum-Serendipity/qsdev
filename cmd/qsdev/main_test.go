package main

import (
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/instance"
	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

// TestRegisterFrameworkAdapters proves all five framework adapters are wired into
// the default registry by the explicit registration that replaced the former
// per-package init() self-registration — and that registration does not panic on
// a duplicate (it is called twice here, as qsdev and a downstream tool both may).
// It is the regression guard for the init()->explicit-wiring refactor.
func TestRegisterFrameworkAdapters(t *testing.T) {
	instance.RegisterFrameworkAdapters()
	instance.RegisterFrameworkAdapters()

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

// TestRegisterFrameworkAdaptersInto proves all five framework adapters are wired by
// the explicit registration that replaced the former per-package init()
// self-registration. It registers into a fresh registry rather than the
// process-global default, so it is repeatable under -count=N.
func TestRegisterFrameworkAdaptersInto(t *testing.T) {
	t.Parallel()

	reg := spi.NewAdapterRegistry()
	instance.RegisterFrameworkAdaptersInto(reg)

	got := map[aiframework.FrameworkID]bool{}
	for _, a := range reg.All() {
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

// TestRegisterFrameworkAdapters_DuplicatePanics proves a doubly-listed adapter
// (a build wiring mistake) is surfaced loudly rather than silently ignored.
func TestRegisterFrameworkAdapters_DuplicatePanics(t *testing.T) {
	t.Parallel()

	reg := spi.NewAdapterRegistry()
	instance.RegisterFrameworkAdaptersInto(reg)

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("second registration into the same registry did not panic")
		}
		if msg, _ := r.(string); !strings.Contains(msg, "already registered") {
			t.Errorf("panic = %v, want an already-registered error", r)
		}
	}()
	instance.RegisterFrameworkAdaptersInto(reg)
}

// TestClassifyInvocation_MCPServers proves the embedded MCP servers, derived
// from the provider registry, are logged as automated while the user-facing
// mcp subcommands keep project logging.
func TestClassifyInvocation_MCPServers(t *testing.T) {
	t.Parallel()
	// In the binary, the claudecode addon's initialize registers the providers
	// before any command (and so initLogging) runs.
	claudecode.RegisterMCPProviders()

	tests := []struct {
		args []string
		want logging.CommandClass
	}{
		{[]string{"mcp", "agent-postmortem"}, logging.ClassAutomated},
		{[]string{"mcp", "version-sentinel"}, logging.ClassAutomated},
		{[]string{"mcp", "install", "github"}, logging.ClassProject},
		{[]string{"mcp", "serve"}, logging.ClassUnlogged},
		{[]string{"init"}, logging.ClassProject},
	}
	for _, tt := range tests {
		t.Run(strings.Join(tt.args, " "), func(t *testing.T) {
			t.Parallel()
			if got := classifyInvocation(tt.args); got != tt.want {
				t.Errorf("classifyInvocation(%q) = %v, want %v", tt.args, got, tt.want)
			}
		})
	}
}
