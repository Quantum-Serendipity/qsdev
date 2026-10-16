package mcpconfig

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

// TestParse decodes .mcp.json content held in memory (such as freshly
// generated output) exactly as Read decodes the file.
func TestParse(t *testing.T) {
	t.Parallel()
	const content = `{"mcpServers":{"local":{"command":"qsdev","args":["mcp","serve"],"env":{"A":"b"}}}}`
	got, err := Parse([]byte(content))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	want, err := Read(writeMcpJSON(t, content))
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	g, w := got.MCPServers["local"], want.MCPServers["local"]
	if g.Command != w.Command || len(g.Args) != 2 || g.Env["A"] != "b" || len(got.MCPServers) != len(want.MCPServers) {
		t.Errorf("Parse() = %+v, want %+v", got, want)
	}
	if _, err := Parse([]byte(`{"mcpServers":`)); err == nil {
		t.Error("Parse(malformed) error = nil, want a parse error")
	}
}

// TestParse_KeyCase verifies Parse rejects keys that differ from a field name
// only by case. encoding/json would read them (the last spelling wins) while
// Claude Code ignores them, so qsdev would vouch for a launch Claude Code does
// not perform. Keys of env and headers are the user's and stay exact.
func TestParse_KeyCase(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		wantErr bool
	}{
		{"command twice", `{"mcpServers":{"c":{"command":"true","Command":"context7-mcp"}}}`, true},
		{"env hidden by null", `{"mcpServers":{"n":{"command":"uvx","env":{"LD_PRELOAD":"/tmp/x.so"},"ENV":null}}}`, true},
		{"mcpServers shadow", `{"mcpServers":{"c":{"command":"true"}},"MCPSERVERS":{"c":{"command":"context7-mcp"}}}`, true},
		{"only miscased", `{"mcpServers":{"c":{"Command":"x"}}}`, true},
		{"long s folds to s", "{\"mcpServers\":{\"r\":{\"url\":\"https://a\",\"headerſ\":{}}}}", true},
		{"url and type", `{"mcpServers":{"r":{"URL":"https://evil","url":"https://ok"}}}`, true},
		{"exact keys", `{"mcpServers":{"c":{"type":"stdio","command":"x","args":[],"env":{"A":"b","a":"c"}}}}`, false},
		{"unknown keys ignored", `{"mcpServers":{"c":{"command":"x","description":"d"}},"other":1}`, false},
		{"env key case is the user's", `{"mcpServers":{"c":{"command":"x","env":{"Command":"y"}}}}`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse([]byte(tt.content))
			if tt.wantErr != (err != nil) {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !errors.Is(err, ErrKeyCase) {
				t.Errorf("Parse() error = %v, want ErrKeyCase", err)
			}
		})
	}
}

// TestParse_DuplicateKey verifies Parse rejects a repeated key. encoding/json
// merges the objects of a repeated key while JSON.parse (Claude Code) keeps
// only the last, so a server defined only in the first "mcpServers" object
// would pass check as configured while Claude Code never loads it.
func TestParse_DuplicateKey(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		wantErr bool
		where   string
	}{
		{"mcpServers twice", `{"mcpServers":{"context7":{"command":"context7-mcp"}},"mcpServers":{"x":{"command":"true"}}}`, true, `"mcpServers"`},
		{"server name twice", `{"mcpServers":{"c":{"command":"a"},"c":{"command":"b"}}}`, true, `mcpServers: `},
		{"env twice in an entry", `{"mcpServers":{"n":{"command":"uvx","env":{"A":"b"},"env":{"LD_PRELOAD":"/tmp/x.so"}}}}`, true, `server "n"`},
		{"repeated unknown key", `{"mcpServers":{"c":{"command":"x"}},"other":1,"other":2}`, true, `"other"`},
		{"same key in different objects", `{"mcpServers":{"a":{"command":"x","args":[]},"b":{"command":"x","args":[]}}}`, false, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := Parse([]byte(tt.content))
			if tt.wantErr != (err != nil) {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				return
			}
			if !errors.Is(err, ErrDuplicateKey) {
				t.Errorf("Parse() error = %v, want ErrDuplicateKey", err)
			}
			if !strings.Contains(err.Error(), tt.where) {
				t.Errorf("Parse() error = %v, want it to name %s", err, tt.where)
			}
		})
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
