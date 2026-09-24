package claudecode

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
)

// TestMcpGrade_GradesConfiguredDefinitionNotRegistry pins F092/F258: a .mcp.json
// entry that reuses a registry name must be graded from what is configured,
// not silently replaced by the pristine registry definition.
func TestMcpGrade_GradesConfiguredDefinitionNotRegistry(t *testing.T) {
	root := t.TempDir()
	mcpJSON := `{"mcpServers": {"version-sentinel": {
		"command": "npx",
		"args": ["-y", "@evil/pkg"],
		"env": {"TOKEN": "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijkl"}
	}}}`
	if err := os.WriteFile(filepath.Join(root, ".mcp.json"), []byte(mcpJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })

	cmd := mcpGradeCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"--json"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("mcp grade: %v", err)
	}

	var entries []gradeJSONEntry
	if err := json.Unmarshal(buf.Bytes(), &entries); err != nil {
		t.Fatalf("parsing output: %v\n%s", err, buf.String())
	}
	if len(entries) != 1 {
		t.Fatalf("graded %d servers, want only the 1 configured server: %+v", len(entries), entries)
	}
	got := entries[0]
	if got.Name != "version-sentinel" || !got.Configured {
		t.Fatalf("entry = %+v, want the configured version-sentinel", got)
	}
	if got.Grade != mcpregistry.ComplianceBasic.String() {
		t.Errorf("grade = %q, want %q for a plaintext-token npx -y server", got.Grade, mcpregistry.ComplianceBasic)
	}
	want := map[string]bool{
		"no-plaintext-secrets":    false,
		"local-only":              false,
		"no-runtime-auto-install": false,
		"verified-provenance":     false,
		criterionMatchesRegistry:  false,
	}
	for _, c := range got.Criteria {
		if wantPassed, ok := want[c.Name]; ok && c.Passed != wantPassed {
			t.Errorf("criterion %s passed = %v, want %v (%s)", c.Name, c.Passed, wantPassed, c.Detail)
		}
		delete(want, c.Name)
	}
	for name := range want {
		t.Errorf("criterion %s missing from the report", name)
	}
}

func TestSelectGradeTargets(t *testing.T) {
	t.Parallel()

	registryDef := &mcpregistry.McpServerDefinition{Name: "github", Command: "github-mcp-server", Transport: mcpregistry.TransportStdio}
	registryOnly := &mcpregistry.McpServerDefinition{Name: "context7", Command: "context7-mcp", Transport: mcpregistry.TransportStdio}
	known := []*mcpregistry.McpServerDefinition{registryDef, registryOnly}
	configured := map[string]mcpregistry.McpServerDefinition{
		"github": {Name: "github", Command: "npx", Args: []string{"-y", "evil-pkg"}, Transport: mcpregistry.TransportStdio},
		"custom": {Name: "custom", Command: "custom-server", Transport: mcpregistry.TransportStdio},
	}

	tests := []struct {
		name           string
		only           string
		all            bool
		wantNames      []string
		wantConfigured []bool
		wantErr        bool
	}{
		{name: "configured only by default", wantNames: []string{"custom", "github"}, wantConfigured: []bool{true, true}},
		{name: "all adds unconfigured registry servers", all: true, wantNames: []string{"context7", "custom", "github"}, wantConfigured: []bool{false, true, true}},
		{name: "named configured server", only: "github", wantNames: []string{"github"}, wantConfigured: []bool{true}},
		{name: "named registry-only server", only: "context7", wantNames: []string{"context7"}, wantConfigured: []bool{false}},
		{name: "unknown server", only: "nope", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			targets, err := selectGradeTargets(configured, known, tt.only, tt.all)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected an error")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var names []string
			var configuredFlags []bool
			for _, target := range targets {
				names = append(names, target.name)
				configuredFlags = append(configuredFlags, target.configured)
				if target.name == "github" && target.def.Command != "npx" {
					t.Errorf("github graded from command %q, want the configured npx", target.def.Command)
				}
			}
			if !slices.Equal(names, tt.wantNames) || !slices.Equal(configuredFlags, tt.wantConfigured) {
				t.Errorf("targets = %v configured=%v, want %v configured=%v", names, configuredFlags, tt.wantNames, tt.wantConfigured)
			}
		})
	}
}

func TestRegistryMatchCriterion(t *testing.T) {
	t.Parallel()

	registry := &mcpregistry.McpServerDefinition{Name: "github", Command: "github-mcp-server", Args: []string{"stdio"}}
	tests := []struct {
		name       string
		configured *mcpregistry.McpServerDefinition
		want       bool
	}{
		{name: "identical", configured: &mcpregistry.McpServerDefinition{Command: "github-mcp-server", Args: []string{"stdio"}}, want: true},
		{name: "different command", configured: &mcpregistry.McpServerDefinition{Command: "npx", Args: []string{"stdio"}}},
		{name: "different args", configured: &mcpregistry.McpServerDefinition{Command: "github-mcp-server", Args: []string{"stdio", "--evil"}}},
		{name: "remote url", configured: &mcpregistry.McpServerDefinition{Command: "github-mcp-server", Args: []string{"stdio"}, URL: "https://evil.example"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := registryMatchCriterion(tt.configured, registry)
			if got.Name != criterionMatchesRegistry || got.Passed != tt.want {
				t.Errorf("criterion = %+v, want passed=%v", got, tt.want)
			}
		})
	}
}

// TestConfiguredGradeDef_CommandlessEntryIsNotStdio pins that a remote
// .mcp.json entry (no command) is not graded as a local stdio server.
func TestConfiguredGradeDef_CommandlessEntryIsNotStdio(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		def       mcpregistry.McpServerDefinition
		wantStdio bool
	}{
		{name: "remote entry parsed as stdio", def: mcpregistry.McpServerDefinition{Name: "socket", Transport: mcpregistry.TransportStdio}},
		{name: "http entry", def: mcpregistry.McpServerDefinition{Name: "socket", URL: "https://mcp.example/", Transport: mcpregistry.TransportHTTP}},
		{name: "local stdio entry", def: mcpregistry.McpServerDefinition{Name: "vs", Command: "qsdev", Args: []string{"mcp", "serve", "--module", "version-sentinel"}, Transport: mcpregistry.TransportStdio}, wantStdio: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			def := configuredGradeDef(tt.def)
			if got := def.Transport == mcpregistry.TransportStdio; got != tt.wantStdio {
				t.Errorf("transport = %q, want stdio=%v", def.Transport, tt.wantStdio)
			}
		})
	}
}
