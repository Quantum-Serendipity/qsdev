package mcpserve

import (
	"context"
	"encoding/json"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/container"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools"
)

// newModuleServer builds the server `qsdev mcp serve --module <modules>` runs
// for the project in dir, with the Guardrail policy derived from its config.
func newModuleServer(t *testing.T, dir string, modules ...string) *Server {
	t.Helper()
	policy, err := policyFromDir(dir)
	if err != nil {
		t.Fatalf("deriving policy: %v", err)
	}
	regs, err := tools.Select(modules, dir, policy, tools.Options{})
	if err != nil {
		t.Fatalf("selecting modules: %v", err)
	}
	srv := newServeServer(dir, container.DeployNative, policy, serveOptions{modules: modules})
	srv.MountTools(regs)
	return srv
}

// TestServeModuleMountsOnlyTheModule proves a --module server exposes exactly
// the selected module's tools: no project context surface, no framework
// adapter tools and no other module.
func TestServeModuleMountsOnlyTheModule(t *testing.T) {
	t.Parallel()
	tests := []struct {
		module string
		want   []string
	}{
		{"agent-postmortem", []string{"analyze_session", "generate_verification_checklist", "list_failure_patterns"}},
		{"version-sentinel", []string{"check_versions", "detect_drift", "manifest_coverage", "version_history"}},
	}
	for _, tt := range tests {
		t.Run(tt.module, func(t *testing.T) {
			t.Parallel()
			srv := newModuleServer(t, t.TempDir(), tt.module)
			got := keys(srv.MCPServer().ListTools())
			slices.Sort(got)
			if !slices.Equal(got, tt.want) {
				t.Errorf("mounted tools = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestServeModuleEnforcesDisabledTools is the F525 regression: the
// agent-postmortem tools used to be served by a bespoke stdio server with no
// middleware, so mcp.disabled_tools could not stop them. Served as a module of
// the universal server, a disabled tool is refused by the Guardrail before its
// handler runs, while the module's other tools still answer.
func TestServeModuleEnforcesDisabledTools(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeDisablingConfig(t, dir, "analyze_session")
	srv := newModuleServer(t, dir, "agent-postmortem")

	denied := callTool(t, srv, "analyze_session", map[string]any{"session_path": "/etc/passwd.jsonl"})
	if !denied.IsError || !strings.Contains(strings.ToLower(resultText(denied)), "denied") {
		t.Errorf("disabled analyze_session: result = %+v, want a guardrail denial", denied)
	}

	allowed := callTool(t, srv, "generate_verification_checklist", nil)
	if strings.Contains(strings.ToLower(resultText(allowed)), "guardrail") {
		t.Errorf("generate_verification_checklist was refused by the guardrail: %s", resultText(allowed))
	}
}

// callTool sends a tools/call request through the server's full handler path
// (visibility filter, middleware chain, bridge) and decodes the result.
func callTool(t *testing.T, srv *Server, name string, args map[string]any) mcp.CallToolResult {
	t.Helper()
	req, err := json.Marshal(map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params":  map[string]any{"name": name, "arguments": args},
	})
	if err != nil {
		t.Fatal(err)
	}
	resp := srv.MCPServer().HandleMessage(context.Background(), req)
	raw, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Result mcp.CallToolResult `json:"result"`
		Error  json.RawMessage    `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("decoding %s: %v", raw, err)
	}
	if len(envelope.Error) > 0 {
		t.Fatalf("tools/call %s: protocol error %s", name, envelope.Error)
	}
	return envelope.Result
}

// resultText concatenates a tool result's text content.
func resultText(res mcp.CallToolResult) string {
	var b strings.Builder
	for _, c := range res.Content {
		if tc, ok := c.(mcp.TextContent); ok {
			b.WriteString(tc.Text)
		}
	}
	return b.String()
}

// TestLegacyModuleCommandsServeTheModule proves the hidden `mcp <module>`
// aliases an older .mcp.json still launches run the universal server
// restricted to that module with the serve defaults, so their tool calls pass
// through the middleware chain rather than a bespoke server.
func TestLegacyModuleCommandsServeTheModule(t *testing.T) {
	t.Parallel()
	serve := Command()
	for _, module := range tools.LegacyServerModules() {
		t.Run(module, func(t *testing.T) {
			t.Parallel()
			var got serveOptions
			cmd := legacyModuleCommand(module, func(_ context.Context, opts serveOptions) error {
				got = opts
				return nil
			})
			var stderr strings.Builder
			cmd.SetErr(&stderr)
			cmd.SetArgs(nil)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("executing mcp %s: %v", module, err)
			}
			if !cmd.Hidden || cmd.Name() != module {
				t.Errorf("alias %q hidden=%v, want a hidden %q command", cmd.Name(), cmd.Hidden, module)
			}
			if !slices.Equal(got.modules, []string{module}) {
				t.Errorf("modules = %v, want [%s]", got.modules, module)
			}
			if def := serve.Flags().Lookup("transport").DefValue; got.transport != def {
				t.Errorf("transport = %q, want the serve default %q", got.transport, def)
			}
			if def := serve.Flags().Lookup("port").DefValue; strconv.Itoa(got.port) != def {
				t.Errorf("port = %d, want the serve default %s", got.port, def)
			}
			if !strings.Contains(stderr.String(), "--module "+module) {
				t.Errorf("deprecation notice = %q, want it to name the replacement", stderr.String())
			}
		})
	}
}
