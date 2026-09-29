package mcpregistry

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
)

func TestConfiguredServers(t *testing.T) {
	t.Parallel()

	reg := NewRegistry()
	reg.MustRegister(McpServerDefinition{Name: "known", Command: "known-mcp", RequiredEnv: []string{"KNOWN_TOKEN"}})

	tests := []struct {
		name    string
		mcpJSON string // "" writes no .mcp.json
		reg     *McpServerRegistry
		want    []mcphealth.ServerConfig
		wantErr bool
	}{
		{name: "no .mcp.json", reg: reg, want: []mcphealth.ServerConfig{}},
		{name: "invalid JSON", mcpJSON: "{", reg: reg, wantErr: true},
		{
			name: "sorted with headers and registry env",
			mcpJSON: `{"mcpServers":{
				"remote": {"type":"http","url":"https://mcp.example/mcp","headers":{"Authorization":"Bearer ${TOKEN}"}},
				"known":  {"command":"known-mcp","args":["serve"],"env":{"A":"b"}}
			}}`,
			reg: reg,
			want: []mcphealth.ServerConfig{
				{Name: "known", Command: "known-mcp", Args: []string{"serve"}, Env: map[string]string{"A": "b"}, RequiredEnv: []string{"KNOWN_TOKEN"}},
				{Name: "remote", URL: "https://mcp.example/mcp", Headers: map[string]string{"Authorization": "Bearer ${TOKEN}"}},
			},
		},
		{
			name:    "nil registry adds no required env",
			mcpJSON: `{"mcpServers":{"known":{"command":"known-mcp"}}}`,
			want:    []mcphealth.ServerConfig{{Name: "known", Command: "known-mcp"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			if tt.mcpJSON != "" {
				if err := os.WriteFile(filepath.Join(dir, ".mcp.json"), []byte(tt.mcpJSON), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			got, err := ConfiguredServers(dir, tt.reg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ConfiguredServers() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr {
				return
			}
			if !slices.EqualFunc(got, tt.want, equalServerConfig) {
				t.Errorf("ConfiguredServers() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func equalServerConfig(a, b mcphealth.ServerConfig) bool {
	return a.Name == b.Name && a.Command == b.Command && a.URL == b.URL &&
		slices.Equal(a.Args, b.Args) && slices.Equal(a.RequiredEnv, b.RequiredEnv) &&
		mapsEqual(a.Env, b.Env) && mapsEqual(a.Headers, b.Headers)
}

// mapsEqual treats a nil and an empty map as equal.
func mapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}
