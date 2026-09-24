package merge

import (
	"encoding/json"
	"maps"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

func TestEnforceMCPPolicy(t *testing.T) {
	t.Parallel()
	onDisk := []byte(`{"extra":true,"mcpServers":{"github":{"command":"gh"},"context7":{"command":"c7"},"mine":{"command":"x","headers":{"a":"b"}}}}`)
	generated := []byte(`{"mcpServers":{"context7":{"command":"c7"}}}`)
	tests := []struct {
		name   string
		path   string
		policy types.MCPPolicy
		want   []string
	}{
		{name: "no policy keeps user-added servers", path: ".mcp.json", want: []string{"context7", "github", "mine"}},
		{name: "named block", path: ".mcp.json", policy: types.MCPPolicy{Blocked: []string{"github"}}, want: []string{"context7", "mine"}},
		{
			name:   "wildcard strips on-disk servers",
			path:   "sub/.mcp.json",
			policy: types.MCPPolicy{Blocked: []string{types.MCPWildcard}, Allowed: []string{"context7"}},
			want:   []string{"context7"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			out, err := MergeOnCreateWithMCPPolicy(tt.policy)(tt.path, onDisk, generated)
			if err != nil {
				t.Fatal(err)
			}
			var doc struct {
				Extra      bool                       `json:"extra"`
				MCPServers map[string]json.RawMessage `json:"mcpServers"`
			}
			if err := json.Unmarshal(out, &doc); err != nil {
				t.Fatalf("parsing result: %v\n%s", err, out)
			}
			if got := slices.Sorted(maps.Keys(doc.MCPServers)); !slices.Equal(got, tt.want) {
				t.Errorf("servers = %v, want %v", got, tt.want)
			}
			if !doc.Extra {
				t.Error("sibling top-level key dropped")
			}
		})
	}

	t.Run("other paths untouched", func(t *testing.T) {
		t.Parallel()
		in := []byte(`{"mcpServers":{"github":{}}}`)
		out, err := EnforceMCPPolicy(".claude/settings.json", in, types.MCPPolicy{Blocked: []string{types.MCPWildcard}})
		if err != nil || string(out) != string(in) {
			t.Errorf("EnforceMCPPolicy changed a non-.mcp.json path: %s, %v", out, err)
		}
	})
}
