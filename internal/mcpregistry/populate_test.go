package mcpregistry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildDefault_RegistersKnownServers(t *testing.T) {
	t.Cleanup(ResetDefaultRegistry)

	r := buildDefault()
	if r.Count() < 13 {
		t.Errorf("buildDefault() registry count = %d, want >= 13", r.Count())
	}
}

func TestBuildDefault_KnownServerProperties(t *testing.T) {
	t.Cleanup(ResetDefaultRegistry)

	r := buildDefault()

	tests := []struct {
		name        string
		wantDisplay string
		wantCat     McpCategory
		wantCmd     string
	}{
		{
			name:        "github",
			wantDisplay: "GitHub MCP",
			wantCat:     CategoryIntegration,
			wantCmd:     "npx",
		},
		{
			name:        "context7",
			wantDisplay: "Context7",
			wantCat:     CategoryAgent,
			wantCmd:     "npx",
		},
		{
			name:        "man-pages",
			wantDisplay: "Man Pages MCP",
			wantCat:     CategoryDocumentation,
			wantCmd:     "uvx",
		},
		{
			name:        "version-sentinel",
			wantDisplay: "Version Sentinel",
			wantCat:     CategoryAgent,
			wantCmd:     "qsdev",
		},
		{
			name:        "local-docs-zim",
			wantDisplay: "Stack Exchange ZIM Documentation",
			wantCat:     CategoryDocumentation,
			wantCmd:     "openzim-mcp",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			def, ok := r.Get(tt.name)
			if !ok {
				t.Fatalf("server %q not found in registry", tt.name)
			}
			if def.DisplayName != tt.wantDisplay {
				t.Errorf("DisplayName = %q, want %q", def.DisplayName, tt.wantDisplay)
			}
			if def.Category != tt.wantCat {
				t.Errorf("Category = %q, want %q", def.Category, tt.wantCat)
			}
			if def.Command != tt.wantCmd {
				t.Errorf("Command = %q, want %q", def.Command, tt.wantCmd)
			}
			if def.Source != SourceBuiltin {
				t.Errorf("Source = %q, want %q", def.Source, SourceBuiltin)
			}
			if def.Transport != TransportStdio {
				t.Errorf("Transport = %q, want %q", def.Transport, TransportStdio)
			}
		})
	}
}

func TestScanMcpJSON_HappyPath(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	content := mcpJSONFile{
		MCPServers: map[string]mcpJSONEntry{
			"test-server": {
				Command: "npx",
				Args:    []string{"@test/mcp-server"},
				Env:     map[string]string{"TEST_KEY": "val"},
			},
			"another-server": {
				Command: "uvx",
				Args:    []string{"another-mcp"},
			},
		},
	}

	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("marshaling test data: %v", err)
	}

	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), data, 0o644); err != nil {
		t.Fatalf("writing .mcp.json: %v", err)
	}

	result, err := ScanMcpJSON(dir)
	if err != nil {
		t.Fatalf("ScanMcpJSON() returned unexpected error: %v", err)
	}

	if len(result) != 2 {
		t.Fatalf("ScanMcpJSON() returned %d entries, want 2", len(result))
	}

	ts, ok := result["test-server"]
	if !ok {
		t.Fatal("expected test-server in result")
	}
	if ts.Command != "npx" {
		t.Errorf("test-server Command = %q, want %q", ts.Command, "npx")
	}
	if len(ts.Args) != 1 || ts.Args[0] != "@test/mcp-server" {
		t.Errorf("test-server Args = %v, want [\"@test/mcp-server\"]", ts.Args)
	}
	if ts.Source != SourceConfig {
		t.Errorf("test-server Source = %q, want %q", ts.Source, SourceConfig)
	}
	if ts.Transport != TransportStdio {
		t.Errorf("test-server Transport = %q, want %q", ts.Transport, TransportStdio)
	}
	if ts.Env["TEST_KEY"] != "val" {
		t.Errorf("test-server Env[TEST_KEY] = %q, want %q", ts.Env["TEST_KEY"], "val")
	}

	as, ok := result["another-server"]
	if !ok {
		t.Fatal("expected another-server in result")
	}
	if as.Command != "uvx" {
		t.Errorf("another-server Command = %q, want %q", as.Command, "uvx")
	}
}

func TestScanMcpJSON_FileNotFound(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	result, err := ScanMcpJSON(dir)
	if err != nil {
		t.Fatalf("ScanMcpJSON() returned unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("ScanMcpJSON() returned %d entries, want 0", len(result))
	}
}

func TestScanMcpJSON_InvalidJSON(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("writing .mcp.json: %v", err)
	}

	_, err := ScanMcpJSON(dir)
	if err == nil {
		t.Fatal("ScanMcpJSON() returned nil error for invalid JSON, want error")
	}
}

func TestScanMcpJSON_EmptyServers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()

	content := mcpJSONFile{
		MCPServers: map[string]mcpJSONEntry{},
	}
	data, err := json.Marshal(content)
	if err != nil {
		t.Fatalf("marshaling test data: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), data, 0o644); err != nil {
		t.Fatalf("writing .mcp.json: %v", err)
	}

	result, err := ScanMcpJSON(dir)
	if err != nil {
		t.Fatalf("ScanMcpJSON() returned unexpected error: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("ScanMcpJSON() returned %d entries, want 0", len(result))
	}
}
