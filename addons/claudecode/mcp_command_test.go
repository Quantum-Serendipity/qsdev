package claudecode

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserver"
)

func TestMcpCmd_Structure(t *testing.T) {
	t.Parallel()

	cmd := mcpCmd()
	if cmd.Use != "mcp" {
		t.Errorf("mcpCmd().Use = %q, want %q", cmd.Use, "mcp")
	}
	if cmd.Short == "" {
		t.Error("mcpCmd().Short is empty")
	}

	for _, provider := range mcpserver.DefaultRegistry().All() {
		sub, _, err := cmd.Find([]string{provider.Name()})
		if err != nil {
			t.Fatalf("finding %s subcommand: %v", provider.Name(), err)
		}
		if sub.Use != provider.Name() {
			t.Errorf("subcommand Use = %q, want %q", sub.Use, provider.Name())
		}
		if sub.RunE == nil {
			t.Errorf("%s subcommand RunE is nil", provider.Name())
		}
	}
}

func TestMcpCmd_HasDiagnosticSubcommands(t *testing.T) {
	t.Parallel()

	cmd := mcpCmd()

	for _, name := range []string{"status", "list"} {
		sub, _, err := cmd.Find([]string{name})
		if err != nil {
			t.Fatalf("finding %s subcommand: %v", name, err)
		}
		if sub.Use != name {
			t.Errorf("subcommand Use = %q, want %q", sub.Use, name)
		}
	}
}

func TestMCPProviderRegistry_HasExpectedProviders(t *testing.T) {
	t.Parallel()

	reg := mcpserver.DefaultRegistry()
	expected := []string{"agent-postmortem", "version-sentinel"}

	for _, name := range expected {
		p, ok := reg.Get(name)
		if !ok {
			t.Errorf("provider %q not registered", name)
			continue
		}
		if len(p.Tools()) == 0 {
			t.Errorf("provider %q has no tools", name)
		}
	}
}
