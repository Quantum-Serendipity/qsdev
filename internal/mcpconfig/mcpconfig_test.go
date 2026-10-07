package mcpconfig

import (
	"os"
	"path/filepath"
	"testing"
)

func writeMcpJSON(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, FileName), []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", FileName, err)
	}
	return dir
}

func TestRead_MissingFileIsEmpty(t *testing.T) {
	t.Parallel()
	f, err := Read(t.TempDir())
	if err != nil {
		t.Fatalf("Read() error = %v, want nil", err)
	}
	if len(f.MCPServers) != 0 {
		t.Errorf("Read() servers = %v, want none", f.MCPServers)
	}
}

func TestRead_URLWithoutTypeIsHTTP(t *testing.T) {
	t.Parallel()
	dir := writeMcpJSON(t, `{"mcpServers":{
		"untyped": {"url":"https://mcp.example/x","headers":{"Authorization":"Bearer ${T}"}},
		"events":  {"type":"sse","url":"https://mcp.example/sse"},
		"local":   {"command":"qsdev","args":["mcp","serve"],"env":{"A":"b"}},
		"typed":   {"type":"stdio","command":"x"}
	}}`)
	f, err := Read(dir)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	tests := []struct {
		name, transport string
	}{
		{"untyped", TransportHTTP},
		{"events", "sse"},
		{"local", TransportStdio},
		{"typed", TransportStdio},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			s, ok := f.MCPServers[tt.name]
			if !ok {
				t.Fatalf("server %q missing", tt.name)
			}
			if got := s.Transport(); got != tt.transport {
				t.Errorf("Transport() = %q, want %q", got, tt.transport)
			}
		})
	}
	if h := f.MCPServers["untyped"].Headers["Authorization"]; h != "Bearer ${T}" {
		t.Errorf("headers not decoded: %q", h)
	}
	if l := f.MCPServers["local"]; len(l.Args) != 2 || l.Env["A"] != "b" {
		t.Errorf("stdio fields not decoded: %+v", l)
	}
}

func TestRead_Malformed(t *testing.T) {
	t.Parallel()
	dir := writeMcpJSON(t, `{"mcpServers":`)
	if _, err := Read(dir); err == nil {
		t.Fatal("Read() error = nil, want a parse error")
	}
}

func TestRead_UnreadableIsError(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	// A directory in place of the file is a read error, not "missing".
	if err := os.Mkdir(filepath.Join(dir, FileName), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := Read(dir); err == nil {
		t.Fatal("Read() error = nil, want a read error")
	}
}
