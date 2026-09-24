package mcpregistry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestScanMcpJSON_RemoteServers is the F257 regression: remote entries in
// .mcp.json carry type/url, which used to be dropped so every server was
// treated as a local stdio server and a remote endpoint graded "secure".
func TestScanMcpJSON_RemoteServers(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	data := []byte(`{"mcpServers":{
		"typed":   {"type":"http","url":"https://mcp.example/mcp","headers":{"Authorization":"Bearer ${TOKEN}"}},
		"untyped": {"url":"https://mcp.example/other"},
		"events":  {"type":"sse","url":"https://mcp.example/sse"},
		"local":   {"type":"stdio","command":"qsdev","args":["mcp","serve"]}
	}}`)
	if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), data, 0o644); err != nil {
		t.Fatalf("writing .mcp.json: %v", err)
	}

	result, err := ScanMcpJSON(dir)
	if err != nil {
		t.Fatalf("ScanMcpJSON() error: %v", err)
	}

	tests := []struct {
		name      string
		transport McpTransport
		url       string
		maxGrade  ComplianceLevel
	}{
		{name: "typed", transport: TransportHTTP, url: "https://mcp.example/mcp", maxGrade: ComplianceBasic},
		{name: "untyped", transport: TransportHTTP, url: "https://mcp.example/other", maxGrade: ComplianceBasic},
		{name: "events", transport: TransportSSE, url: "https://mcp.example/sse", maxGrade: ComplianceBasic},
		{name: "local", transport: TransportStdio, maxGrade: ComplianceAttested},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			def, ok := result[tt.name]
			if !ok {
				t.Fatalf("%s missing from result", tt.name)
			}
			if def.Transport != tt.transport {
				t.Errorf("Transport = %q, want %q", def.Transport, tt.transport)
			}
			if def.URL != tt.url {
				t.Errorf("URL = %q, want %q", def.URL, tt.url)
			}
			if got := GradeServer(&def).Level; got > tt.maxGrade {
				t.Errorf("grade = %s, want at most %s", got, tt.maxGrade)
			}
		})
	}

	if h := result["typed"].Headers["Authorization"]; h != "Bearer ${TOKEN}" {
		t.Errorf("typed Authorization header = %q", h)
	}
}

// TestHasPlaintextSecrets_BeyondEnv is part of the F257 regression: a token
// embedded in a URL, an arg or a header is as leaked as one in env.
func TestHasPlaintextSecrets_BeyondEnv(t *testing.T) {
	t.Parallel()

	// Built at runtime so secret scanners do not flag the fixture.
	fake := "token_" + strings.Repeat("A", 30)

	tests := []struct {
		name string
		def  McpServerDefinition
		want bool
	}{
		{name: "token in url query", def: McpServerDefinition{URL: "https://mcp.example/mcp?token=" + fake}, want: true},
		{name: "password in url userinfo", def: McpServerDefinition{URL: "https://user:pw@mcp.example/mcp"}, want: true},
		{name: "variable in url query", def: McpServerDefinition{URL: "https://mcp.example/mcp?token=${TOKEN}"}, want: false},
		{name: "token in url path", def: McpServerDefinition{URL: "https://mcp.example/mcp/" + fake + "/sse"}, want: true},
		{name: "plain url", def: McpServerDefinition{URL: "https://mcp.example/mcp?mode=read"}, want: false},
		{name: "plain url path", def: McpServerDefinition{URL: "https://mcp.example/v1/servers/github/mcp"}, want: false},
		{name: "token as flag value", def: McpServerDefinition{Args: []string{"--token=" + fake}}, want: true},
		{name: "token as separate arg", def: McpServerDefinition{Args: []string{"--token", fake}}, want: true},
		{name: "package args", def: McpServerDefinition{Args: []string{"-y", "@upstash/context7-mcp"}}, want: false},
		{name: "bearer header", def: McpServerDefinition{Headers: map[string]string{"Authorization": "Bearer " + fake}}, want: true},
		{name: "bearer header variable", def: McpServerDefinition{Headers: map[string]string{"Authorization": "Bearer ${TOKEN}"}}, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := hasPlaintextSecrets(&tt.def); got != tt.want {
				t.Errorf("hasPlaintextSecrets() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsLocalOnly_URL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		url  string
		want bool
	}{
		{url: "https://mcp.example/mcp", want: false},
		{url: "http://localhost:8080/mcp", want: true},
		{url: "http://127.0.0.1:8080/mcp", want: true},
		{url: "http://[::1]:8080/mcp", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			t.Parallel()
			if got := isLocalOnly(&McpServerDefinition{URL: tt.url}); got != tt.want {
				t.Errorf("isLocalOnly(%q) = %v, want %v", tt.url, got, tt.want)
			}
		})
	}
}
