package claudecode

import "testing"

func TestMcpCmd_Structure(t *testing.T) {
	t.Parallel()

	cmd := mcpCmd()
	if cmd.Use != "mcp" {
		t.Errorf("mcpCmd().Use = %q, want %q", cmd.Use, "mcp")
	}
	if cmd.Short == "" {
		t.Error("mcpCmd().Short is empty")
	}

	subs := []string{"agent-postmortem", "version-sentinel"}
	for _, name := range subs {
		sub, _, err := cmd.Find([]string{name})
		if err != nil {
			t.Fatalf("finding %s subcommand: %v", name, err)
		}
		if sub.Use != name {
			t.Errorf("subcommand Use = %q, want %q", sub.Use, name)
		}
		if sub.Short == "" {
			t.Errorf("%s subcommand Short is empty", name)
		}
		if sub.RunE == nil {
			t.Errorf("%s subcommand RunE is nil", name)
		}
	}
}
