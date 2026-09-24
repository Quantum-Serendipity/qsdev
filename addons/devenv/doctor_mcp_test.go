package devenv

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/doctor"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
)

// TestMCPConfigSection is the F369 regression: `devenv doctor` never filled
// its MCP section. It must now validate .mcp.json statically, and never start
// a configured server, since .mcp.json is repository content.
func TestMCPConfigSection(t *testing.T) {
	t.Parallel()

	// unsetVar is never set by the test environment.
	const unsetVar = "QSDEV_F369_DOCTOR_UNSET_VAR"

	reg := mcpregistry.NewRegistry()
	reg.MustRegister(mcpregistry.McpServerDefinition{Name: "needs-env", RequiredEnv: []string{unsetVar}})

	tests := []struct {
		name        string
		mcpJSON     func(exe string) string // nil writes no .mcp.json
		noRoot      bool
		wantNil     bool
		wantWarning string
		wantStatus  map[string]string
	}{
		{name: "outside a project", noRoot: true, wantNil: true},
		{name: "no .mcp.json", wantNil: true},
		{name: "no servers", mcpJSON: func(string) string { return `{"mcpServers":{}}` }, wantNil: true},
		{
			name:        "unparsable .mcp.json",
			mcpJSON:     func(string) string { return "{" },
			wantWarning: "MCP servers not validated: loading configured MCP servers: parsing .mcp.json",
			wantStatus:  map[string]string{},
		},
		{
			name: "static validation of each server",
			mcpJSON: func(exe string) string {
				data, err := json.Marshal(map[string]any{"mcpServers": map[string]any{
					"good":      map[string]any{"command": exe},
					"needs-env": map[string]any{"command": exe},
					"env-ref":   map[string]any{"command": exe, "args": []string{"--token=${" + unsetVar + "}"}},
					"missing":   map[string]any{"command": filepath.Join(t.TempDir(), "absent-mcp")},
					"plain":     map[string]any{"type": "http", "url": "http://mcp.example.test/mcp"},
					"remote":    map[string]any{"type": "http", "url": "https://mcp.example.test/mcp"},
				}})
				if err != nil {
					t.Fatal(err)
				}
				return string(data)
			},
			wantStatus: map[string]string{
				"good":      doctor.MCPStatusOK,
				"needs-env": doctor.MCPStatusDegraded,
				"env-ref":   doctor.MCPStatusDegraded,
				"missing":   doctor.MCPStatusMisconfigured,
				"plain":     doctor.MCPStatusMisconfigured,
				"remote":    doctor.MCPStatusOK,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			exe, marker := markerServer(t)
			if tt.mcpJSON != nil {
				if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(tt.mcpJSON(exe)), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			root := dir
			if tt.noRoot {
				root = ""
			}

			ms := mcpConfigSection(root, reg)

			if _, err := os.Stat(marker); err == nil {
				t.Fatal("doctor started a configured MCP server")
			}
			if (ms == nil) != tt.wantNil {
				t.Fatalf("mcpConfigSection() = %+v, wantNil %v", ms, tt.wantNil)
			}
			if ms == nil {
				return
			}
			if tt.wantWarning != "" && !strings.Contains(strings.Join(ms.Warnings, "\n"), tt.wantWarning) {
				t.Errorf("warnings = %v, want one containing %q", ms.Warnings, tt.wantWarning)
			}
			got := make(map[string]string, len(ms.Servers))
			for _, s := range ms.Servers {
				got[s.Name] = s.Status
			}
			if len(got) != len(tt.wantStatus) {
				t.Errorf("servers = %v, want %v", got, tt.wantStatus)
			}
			for name, want := range tt.wantStatus {
				if got[name] != want {
					t.Errorf("server %q status = %q, want %q (all: %+v)", name, got[name], want, ms.Servers)
				}
			}
		})
	}
}

// markerServer writes an executable that creates marker when it runs, so a
// test can prove the doctor never started it.
func markerServer(t *testing.T) (exe, marker string) {
	t.Helper()
	dir := t.TempDir()
	marker = filepath.Join(dir, "started")
	exe = filepath.Join(dir, "marker-mcp")
	script := "#!/bin/sh\ntouch '" + marker + "'\n"
	if runtime.GOOS == "windows" {
		exe += ".bat"
		script = "@echo off\r\ntype nul > \"" + marker + "\"\r\n"
	}
	if err := os.WriteFile(exe, []byte(script), 0o755); err != nil { //nolint:gosec // test executable
		t.Fatal(err)
	}
	return exe, marker
}
