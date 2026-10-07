package devinit

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/check"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpconfig"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/surgery"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// breakProjectDefaults writes a project defaults file the catalog rejects into
// dir and points the catalog at it, so the next registry build fails as it
// does for a real run in that project. The cached catalog and registry are
// rebuilt once the test ends.
func breakProjectDefaults(t *testing.T, dir string) {
	t.Helper()
	path := catalog.ProjectConfigPath(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("tier_to_compliance:\n  standard: nonexistent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prevRoot := catalog.ProjectRoot()
	t.Cleanup(func() {
		catalog.ResetDefault() // also clears the project root
		if err := catalog.SetProjectRoot(prevRoot); err != nil {
			t.Errorf("restoring the catalog project root: %v", err)
		}
		toolreg.ResetDefaultRegistry()
	})
	catalog.ResetDefault()
	if err := catalog.SetProjectRoot(dir); err != nil {
		t.Fatal(err)
	}
	toolreg.ResetDefaultRegistry()
	if _, err := toolreg.Default(); err == nil {
		t.Fatal("test setup: the project defaults file should break the registry")
	}
}

// TestCheckCmd_BrokenProjectDefaults pins U08-WS3: with a project defaults
// file the catalog rejects, check reports a failing
// config_integrity/config_catalog result and exits 1 in every output format,
// still runs the checks that need no catalog, and applies no auto-fix (the
// fixes regenerate from the catalog it could not load).
func TestCheckCmd_BrokenProjectDefaults(t *testing.T) {
	dir := initLifecycleProject(t)
	// A missing manifest is auto-fixable without the catalog, so an
	// auto-fix that ran would rewrite it.
	manifest := filepath.Join(dir, state.ManifestFile())
	if err := os.Remove(manifest); err != nil {
		t.Fatal(err)
	}
	breakProjectDefaults(t, dir)

	tests := []struct {
		name     string
		args     []string
		want     []string
		wantExit func(error) bool
	}{
		{
			name: "human",
			args: []string{"--auto-fix"},
			want: []string{"[FAIL] config_catalog: loading " + branding.Get().AppName + " defaults:", "defaults validate", "generated_manifest"},
			wantExit: func(err error) bool {
				var cf *check.CheckFailedError
				return errors.As(err, &cf) && cf.ExitCode() == 1
			},
		},
		{
			name: "json",
			args: []string{"--format", "json", "--auto-fix"},
			want: []string{`"name": "config_catalog"`, `"status": "fail"`, "defaults validate", "generated_manifest"},
			wantExit: func(err error) bool {
				var ee *ExitError
				return errors.As(err, &ee) && ee.Code == 1
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runLifecycleCmd(t, dir, checkCmd(), tt.args...)
			if !tt.wantExit(err) {
				t.Errorf("check error = %#v, want exit 1", err)
			}
			for _, s := range tt.want {
				if !strings.Contains(out, s) {
					t.Errorf("output lacks %q", s)
				}
			}
			if strings.Contains(out, "panic") {
				t.Error("output mentions a panic")
			}
			if _, err := os.Stat(manifest); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("auto-fix ran: manifest stat err = %v", err)
			}
			if t.Failed() {
				t.Logf("output:\n%s", out)
			}
		})
	}
}

// mcpServerFailures returns the failing tool_mcp_server_* checks in report.
func mcpServerFailures(report check.CheckReport) []check.CheckResult {
	var out []check.CheckResult
	for _, c := range report.Checks {
		if strings.HasPrefix(c.Name, "tool_mcp_server_") && c.Status == check.StatusFail {
			out = append(out, c)
		}
	}
	return out
}

// TestCheck_FreshGenerationPasses verifies the .mcp.json qsdev generates
// configures every always-on MCP tool's server (from the tool registry) as a
// definition check vouches for: no tool_mcp_server_* check fails.
func TestCheck_FreshGenerationPasses(t *testing.T) {
	dir := initStandardProject(t)
	content, err := os.ReadFile(filepath.Join(dir, check.MCPConfigRelPath))
	if err != nil {
		t.Fatalf("reading .mcp.json: %v", err)
	}
	for _, tool := range alwaysOnMCPTools(t) {
		if !surgery.JSONHasMCPServer(content, tool.MCPServer) {
			t.Fatalf(".mcp.json lacks %s's server %q; the check is not exercised:\n%s", tool.Name, tool.MCPServer, content)
		}
	}
	if failed := mcpServerFailures(runCheckJSON(t, dir)); len(failed) != 0 {
		t.Errorf("freshly generated project fails %+v", failed)
	}
}

// TestCheck_ReplacedAlwaysOnMCPServerFails is the W0N-08 regression: an
// always-on MCP tool's server entry replaced with another command fails
// check as untrusted, naming the command, rather than passing as present.
func TestCheck_ReplacedAlwaysOnMCPServerFails(t *testing.T) {
	for _, tool := range alwaysOnMCPTools(t) {
		t.Run(tool.Name, func(t *testing.T) {
			dir := initStandardProject(t)
			replaced := `{"mcpServers":{` + strconv.Quote(tool.MCPServer) + `:{"command":"true"}}}`
			if err := os.WriteFile(filepath.Join(dir, check.MCPConfigRelPath), []byte(replaced), 0o644); err != nil {
				t.Fatal(err)
			}
			var got *check.CheckResult
			for _, c := range mcpServerFailures(runCheckJSON(t, dir)) {
				if c.Name == "tool_mcp_server_untrusted_"+tool.Name {
					got = &c
				}
			}
			if got == nil {
				t.Fatalf("check did not fail tool_mcp_server_untrusted_%s", tool.Name)
			}
			if !strings.Contains(got.Message, `command "true"`) {
				t.Errorf("message %q does not name the replaced command", got.Message)
			}
		})
	}
}

// TestCheck_CommandBesideTrustedURLFails is the transport-confusion
// regression for check: each always-on remote MCP server's generated entry
// (from the generated .mcp.json, not a fixed list) given a command beside its
// trusted URL, which Claude Code launches as stdio, fails check as untrusted.
func TestCheck_CommandBesideTrustedURLFails(t *testing.T) {
	dir := initStandardProject(t)
	path := filepath.Join(dir, check.MCPConfigRelPath)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	generated, err := mcpconfig.Parse(content)
	if err != nil {
		t.Fatal(err)
	}
	exercised := 0
	for _, tool := range alwaysOnMCPTools(t) {
		entry, ok := generated.MCPServers[tool.MCPServer]
		if !ok || entry.URL == "" {
			continue
		}
		exercised++
		for _, typ := range []string{mcpconfig.TransportStdio, ""} {
			t.Run(tool.Name+"/type="+typ, func(t *testing.T) {
				evil := entry
				evil.Type, evil.Command = typ, "./evil.sh"
				data, err := json.Marshal(mcpconfig.File{MCPServers: map[string]mcpconfig.Server{tool.MCPServer: evil}})
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, data, 0o644); err != nil {
					t.Fatal(err)
				}
				var got *check.CheckResult
				for _, c := range mcpServerFailures(runCheckJSON(t, dir)) {
					if c.Name == "tool_mcp_server_untrusted_"+tool.Name {
						got = &c
					}
				}
				if got == nil {
					t.Fatalf("check did not fail tool_mcp_server_untrusted_%s for %s", tool.Name, data)
				}
				if !strings.Contains(got.Message, `command "./evil.sh"`) {
					t.Errorf("message %q does not name the command", got.Message)
				}
			})
		}
	}
	if exercised == 0 {
		t.Fatalf("no always-on MCP tool has a remote entry in the generated .mcp.json; the check is not exercised:\n%s", content)
	}
}

// TestTrustedMCPServers_IncludesGeneratedEntry verifies the generated entry
// of each required server is trusted next to the catalog's definitions, so a
// server built into the binary that replaces the catalog definition of the
// same name (which generation writes, but the catalog's launch variants do not
// cover) does not fail check.
func TestTrustedMCPServers_IncludesGeneratedEntry(t *testing.T) {
	fresh := map[string]types.GeneratedFile{check.MCPConfigRelPath: {
		Content: []byte(`{"mcpServers":{"context7":{"command":"project-context7","args":["--stdio"],"env":{"K":"${K}"}},"other":{"command":"x"}}}`),
	}}
	trusted := trustedMCPServers(map[string]string{"context7": "context7"}, fresh)
	want := mcpregistry.LaunchSpec{Command: "project-context7", Args: []string{"--stdio"}, Env: map[string]string{"K": "${K}"}}
	cfg := mcphealth.ServerConfig{Name: "context7", Command: want.Command, Args: want.Args, Env: want.Env}
	if !mcpregistry.MatchesTrusted(cfg, trusted["context7"]) {
		t.Errorf("trusted[context7] = %+v lacks the generated entry %+v", trusted["context7"], want)
	}
	if mcpregistry.MatchesTrusted(mcphealth.ServerConfig{Name: "other", Command: "x"}, trusted["other"]) {
		t.Error("a generated entry of a server check does not require was trusted")
	}
	if len(trusted["context7"]) < 2 {
		t.Errorf("trusted[context7] = %+v, want the catalog's definitions as well", trusted["context7"])
	}
}
