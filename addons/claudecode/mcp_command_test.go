package claudecode

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/agentpostmortem"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools/versionsentinel"
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
}

// TestMcpCmd_ServesModulesOnlyThroughServe is the F525 regression: the tool
// modules (agent-postmortem, version-sentinel) are served by `mcp serve
// --module`, behind the universal server's middleware. The `mcp <module>`
// subcommands an older .mcp.json launches survive only as hidden aliases of it
// (see mcpserve.LegacyModuleCommands), never as bespoke servers.
func TestMcpCmd_ServesModulesOnlyThroughServe(t *testing.T) {
	t.Parallel()

	cmd := mcpCmd()
	serve, _, err := cmd.Find([]string{"serve"})
	if err != nil || serve.Use != "serve" {
		t.Fatalf("finding serve subcommand: %v", err)
	}
	if serve.Flags().Lookup("module") == nil {
		t.Error("serve has no --module flag")
	}
	for _, module := range []string{agentpostmortem.ModuleName, versionsentinel.ModuleName} {
		sub, _, err := cmd.Find([]string{module})
		if err != nil || sub.Name() != module {
			t.Errorf("mcp %s: not found (%v); an older .mcp.json launches it", module, err)
			continue
		}
		if !sub.Hidden {
			t.Errorf("mcp %s is a visible subcommand; it must be a hidden alias of mcp serve --module", module)
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
