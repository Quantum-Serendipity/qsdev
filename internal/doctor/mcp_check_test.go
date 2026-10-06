package doctor

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
)

// TestMCPFindings covers the static MCP validation both doctors share: it
// reads .mcp.json, reports per-server status without starting anything, fails
// on an unreadable file, and carries a broken catalog as a warning.
func TestMCPFindings(t *testing.T) {
	t.Parallel()

	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	servers := func(entries map[string]any) string {
		data, err := json.Marshal(map[string]any{"mcpServers": entries})
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}

	tests := []struct {
		name        string
		mcpJSON     string // empty writes no .mcp.json
		noRoot      bool
		catalogErr  error
		wantErr     bool
		wantNil     bool
		wantWarning string
		wantStatus  map[string]string
	}{
		{name: "outside a project", noRoot: true, wantNil: true},
		{name: "missing .mcp.json", wantNil: true},
		{name: "no servers", mcpJSON: `{"mcpServers":{}}`, wantNil: true},
		{name: "invalid JSON", mcpJSON: "{not json", wantErr: true},
		{
			name: "PATH-missing command is misconfigured",
			mcpJSON: servers(map[string]any{
				"present": map[string]any{"command": exe},
				"missing": map[string]any{"command": "qsdev-u21-absent-mcp"},
			}),
			wantStatus: map[string]string{"present": MCPStatusOK, "missing": MCPStatusMisconfigured},
		},
		{
			name:        "catalog error carried as a warning",
			mcpJSON:     servers(map[string]any{"present": map[string]any{"command": exe}}),
			catalogErr:  errors.New("loading MCP server catalog: bad yaml"),
			wantWarning: "bad yaml",
			wantStatus:  map[string]string{"present": MCPStatusOK},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := t.TempDir()
			if tt.mcpJSON != "" {
				if err := os.WriteFile(filepath.Join(root, ".mcp.json"), []byte(tt.mcpJSON), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if tt.noRoot {
				root = ""
			}

			ms, err := mcpFindings(root, mcpregistry.NewRegistry(), tt.catalogErr)
			if (err != nil) != tt.wantErr {
				t.Fatalf("mcpFindings() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				if ms != nil {
					t.Errorf("mcpFindings() = %+v alongside an error, want nil", ms)
				}
				return
			}
			if (ms == nil) != tt.wantNil {
				t.Fatalf("mcpFindings() = %+v, wantNil %v", ms, tt.wantNil)
			}
			if ms == nil {
				return
			}
			switch got := strings.Join(ms.Warnings, "\n"); {
			case tt.wantWarning == "" && got != "":
				t.Errorf("warnings = %v, want none", ms.Warnings)
			case !strings.Contains(got, tt.wantWarning):
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

// TestMCPFindings_UsesRegistryCatalogErr verifies the exported entry point
// takes the catalog error from the registry: a healthy registry adds no warning.
func TestMCPFindings_UsesRegistryCatalogErr(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".mcp.json"), []byte(`{"mcpServers":{"x":{"command":"qsdev-u21-absent-mcp"}}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	ms, err := MCPFindings(root, mcpregistry.NewRegistry())
	if err != nil || ms == nil || len(ms.Warnings) != 0 || len(ms.Servers) != 1 {
		t.Fatalf("MCPFindings() = %+v, %v; want one server and no warnings", ms, err)
	}
}

func TestNewMCPSection(t *testing.T) {
	t.Parallel()

	servers := []mcphealth.ServerConfig{
		{Name: "good", Command: "/bin/good-mcp"},
		{Name: "remote", URL: "https://mcp.example/mcp"},
		{Name: "noenv", Command: "/bin/noenv-mcp"},
		{Name: "broken", Command: "missing-mcp"},
		{Name: "odd", Command: "/bin/odd-mcp"},
		{Name: "", Command: "x"},
	}
	findings := []mcphealth.ConfigWarning{
		{Server: "noenv", Severity: mcphealth.SeverityWarning, Message: "required environment variable \"TOKEN\" is not set", Remediation: "set TOKEN"},
		{Server: "broken", Severity: mcphealth.SeverityWarning, Message: "arg 0 references ${X} which is not set"},
		{Server: "broken", Severity: mcphealth.SeverityError, Message: "command \"missing-mcp\" not found on PATH", Remediation: "install it"},
		{Server: "odd", Severity: "surprising", Message: "unknown severity"},
		{Server: emptyServerName, Severity: mcphealth.SeverityError, Message: "server name is empty"},
	}

	ms := NewMCPSection(servers, findings, []string{"catalog not loaded"})
	if ms == nil || !ms.Detected {
		t.Fatalf("NewMCPSection() = %+v, want a detected section", ms)
	}
	if len(ms.Warnings) != 1 {
		t.Errorf("warnings = %v, want the catalog warning", ms.Warnings)
	}

	tests := []struct {
		name          string
		wantTransport string
		wantStatus    string
		wantIssues    int
	}{
		{"good", "stdio", MCPStatusOK, 0},
		{"remote", "http", MCPStatusOK, 0},
		{"noenv", "stdio", MCPStatusDegraded, 1},
		{"broken", "stdio", MCPStatusMisconfigured, 2},
		{"odd", "stdio", MCPStatusMisconfigured, 1},
		{emptyServerName, "stdio", MCPStatusMisconfigured, 1},
	}
	if len(ms.Servers) != len(tests) {
		t.Fatalf("got %d servers, want %d", len(ms.Servers), len(tests))
	}
	for i, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := ms.Servers[i]
			if got.Name != tt.name || got.Transport != tt.wantTransport || got.Status != tt.wantStatus || len(got.Issues) != tt.wantIssues {
				t.Errorf("server = %+v, want name %q transport %q status %q with %d issue(s)",
					got, tt.name, tt.wantTransport, tt.wantStatus, tt.wantIssues)
			}
		})
	}
}

func TestNewMCPSection_Empty(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		servers  []mcphealth.ServerConfig
		warnings []string
		wantNil  bool
	}{
		{name: "nothing configured", wantNil: true},
		{name: "unreadable .mcp.json", warnings: []string{"parsing .mcp.json: bad"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			ms := NewMCPSection(tt.servers, nil, tt.warnings)
			if (ms == nil) != tt.wantNil {
				t.Fatalf("NewMCPSection() = %+v, wantNil %v", ms, tt.wantNil)
			}
			if ms == nil {
				return
			}
			// Servers must encode as [] rather than null for JSON consumers.
			data, err := json.Marshal(ms)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(data), `"servers":[]`) {
				t.Errorf("JSON = %s, want an empty servers array", data)
			}
		})
	}
}

func TestFormatReport_MCPSection(t *testing.T) {
	t.Parallel()

	ms := NewMCPSection(
		[]mcphealth.ServerConfig{
			{Name: "context7", Command: "/bin/npx"},
			{Name: "github", Command: "/bin/github-mcp"},
			{Name: "broken", Command: "missing-mcp"},
		},
		[]mcphealth.ConfigWarning{
			{Server: "github", Severity: mcphealth.SeverityWarning, Message: "required environment variable \"GITHUB_TOKEN\" is not set", Remediation: "set GITHUB_TOKEN in your environment or .env file"},
			{Server: "broken", Severity: mcphealth.SeverityError, Message: "command \"missing-mcp\" not found on PATH"},
		},
		[]string{"catalog not loaded"},
	)
	r := &Report{QsdevVersion: "0.1.0"}
	r.SetMCPSection(ms)

	var buf bytes.Buffer
	FormatReport(&buf, r, false)
	out := buf.String()
	for _, want := range []string{
		"MCP Servers (configuration only; no server was started)",
		"context7             [OK] ok (stdio)",
		"github               [WARN] degraded (stdio)",
		"[WARN] required environment variable \"GITHUB_TOKEN\" is not set",
		"fix: set GITHUB_TOKEN in your environment or .env file",
		"broken               [FAIL] misconfigured (stdio)",
		"[FAIL] command \"missing-mcp\" not found on PATH",
		"[WARN] catalog not loaded",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("report missing %q:\n%s", want, out)
		}
	}
	if strings.Contains(out, "tools:") {
		t.Errorf("report claims a tool count although no server was started:\n%s", out)
	}
}

func TestDisplayMCPServerName(t *testing.T) {
	t.Parallel()

	for in, want := range map[string]string{
		"github":            "github",
		"evil\u001b[2K\rok": `"evil\x1b[2K\rok"`,
	} {
		if got := (MCPServerInfo{Name: in}).DisplayName(); got != want {
			t.Errorf("DisplayName(%q) = %s, want %s", in, got, want)
		}
	}
}

func TestFormatReport_MCPSectionEscapesName(t *testing.T) {
	t.Parallel()

	r := &Report{QsdevVersion: "0.1.0"}
	r.SetMCPSection(NewMCPSection([]mcphealth.ServerConfig{{Name: "x\u001b[1Ay", Command: "/bin/x"}}, nil, nil))
	var buf bytes.Buffer
	FormatReport(&buf, r, false)
	if strings.ContainsRune(buf.String(), '\u001b') {
		t.Errorf("report passes a raw escape from .mcp.json to the terminal:\n%q", buf.String())
	}
}
