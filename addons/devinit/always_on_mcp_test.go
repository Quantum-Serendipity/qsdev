package devinit

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/check"
	"github.com/Quantum-Serendipity/qsdev/internal/surgery"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
)

// initStandardProject runs `init --yes --lang go --tier standard` in a fresh
// directory with a go.mod and returns the directory. At the standard tier
// .mcp.json is generated only because servers are configured, so it is where
// a dropped server list loses the always-on MCP tools.
func initStandardProject(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/mcp\n\ngo 1.24\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := executeInitCmd(t, dir, "--yes", "--lang", "go", "--tier", "standard"); err != nil {
		t.Fatalf("init: %v\n%s", err, out)
	}
	return dir
}

// alwaysOnMCPTools returns the always-on catalog tools backed by an MCP
// server.
func alwaysOnMCPTools(t *testing.T) []*toolreg.Tool {
	t.Helper()
	var out []*toolreg.Tool
	for _, tool := range toolreg.DefaultRegistry().All() {
		if tool.Default == toolreg.AlwaysOn && tool.MCPServer != "" {
			out = append(out, tool)
		}
	}
	if len(out) == 0 {
		t.Fatal("catalog declares no always-on MCP tools")
	}
	return out
}

// TestAlwaysOnMCP_LocalServerListCannotDropThem is the U28-WS1 regression: a
// local answers edit `mcp_servers: []` neither removes the always-on MCP
// tools' servers from .mcp.json nor from the committed claude_code.mcp_servers,
// whichever committing path regenerates, and the committed servers it dropped
// are warned about.
func TestAlwaysOnMCP_LocalServerListCannotDropThem(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, dir string) (string, error)
	}{
		{name: "init --update", run: func(t *testing.T, dir string) (string, error) {
			t.Helper()
			return executeInitCmd(t, dir, "--update", "--yes")
		}},
		{name: "enable", run: func(t *testing.T, dir string) (string, error) {
			t.Helper()
			return enableTool(t, dir, "commitlint")
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := initStandardProject(t)
			committed := committedConfig(t, dir).ClaudeCode.MCPServers

			local := loadProjectAnswers(t, dir)
			local.MCPServers = []string{}
			if err := saveAnswers(dir, local); err != nil {
				t.Fatalf("saving answers: %v", err)
			}

			out, err := tt.run(t, dir)
			if err != nil {
				t.Fatalf("%s: %v\n%s", tt.name, err, out)
			}

			mcp, err := os.ReadFile(filepath.Join(dir, check.MCPConfigRelPath))
			if err != nil {
				t.Fatalf("reading .mcp.json: %v", err)
			}
			after := committedConfig(t, dir).ClaudeCode.MCPServers
			for _, tool := range alwaysOnMCPTools(t) {
				if !surgery.JSONHasMCPServer(mcp, tool.MCPServer) {
					t.Errorf(".mcp.json lacks %s's server %q:\n%s", tool.Name, tool.MCPServer, mcp)
				}
				if !slices.Contains(committed, tool.MCPServer) {
					continue
				}
				if !slices.Contains(after, tool.MCPServer) {
					t.Errorf("committed mcp_servers = %v, lost %s's server %q", after, tool.Name, tool.MCPServer)
				}
				if want := `always-on tool "` + tool.Name + `" kept enabled`; !strings.Contains(out, want) {
					t.Errorf("output lacks %q:\n%s", want, out)
				}
			}
		})
	}
}

// TestRunCheck_FailsWhenAlwaysOnMCPServerMissing verifies check fails when the
// on-disk .mcp.json lacks the server of an always-on MCP tool the project
// enables, and passes the project as generated.
func TestRunCheck_FailsWhenAlwaysOnMCPServerMissing(t *testing.T) {
	dir := initStandardProject(t)
	missing := func(report check.CheckReport) []string {
		var names []string
		for _, c := range report.Checks {
			if strings.HasPrefix(c.Name, "tool_mcp_server_missing_") && c.Status == check.StatusFail {
				names = append(names, strings.TrimPrefix(c.Name, "tool_mcp_server_missing_"))
			}
		}
		return names
	}

	if got := missing(runCheckJSON(t, dir)); len(got) != 0 {
		t.Fatalf("generated project reports missing MCP servers %v", got)
	}

	if err := os.Remove(filepath.Join(dir, check.MCPConfigRelPath)); err != nil {
		t.Fatal(err)
	}
	got := missing(runCheckJSON(t, dir))
	for _, tool := range alwaysOnMCPTools(t) {
		if !slices.Contains(got, tool.Name) {
			t.Errorf("check did not fail tool_mcp_server_missing_%s; failed %v", tool.Name, got)
		}
	}
}
