package merge

import (
	"encoding/json"
	"testing"
)

// TestMergeMcpJson_ExistingServerNotInBase covers a generated server whose name
// the user already configured but which is not in base: always the case on the
// nil-base create path, and on update when the generator newly adds a name.
// Ours' modeled fields must win while the user's env keys and unmodeled
// fields (headers) survive.
func TestMergeMcpJson_ExistingServerNotInBase(t *testing.T) {
	t.Parallel()

	theirs := []byte(`{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["gh-mcp@1.0.0"],
      "env": {"GITHUB_TOKEN": "user-token"},
      "headers": {"X-Custom": "1"}
    }
  }
}`)
	ours := []byte(`{
  "mcpServers": {
    "github": {
      "command": "npx",
      "args": ["gh-mcp@1.2.3"],
      "env": {"GH_HOST": "github.com"}
    }
  }
}`)
	otherBase := []byte(`{"mcpServers": {"other": {"command": "x"}}}`)

	tests := []struct {
		name string
		base []byte
	}{
		{name: "create path (nil base)", base: nil},
		{name: "update path (name new to generator)", base: otherBase},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := MergeMcpJson(tt.base, theirs, ours)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			var doc struct {
				MCPServers map[string]map[string]any `json:"mcpServers"`
			}
			if err := json.Unmarshal(got, &doc); err != nil {
				t.Fatalf("invalid JSON output: %v", err)
			}
			gh := doc.MCPServers["github"]
			if gh == nil {
				t.Fatalf("github server missing from output: %s", got)
			}

			args, _ := gh["args"].([]any)
			if len(args) != 1 || args[0] != "gh-mcp@1.2.3" {
				t.Errorf("args = %v, want ours [gh-mcp@1.2.3]", gh["args"])
			}
			env, _ := gh["env"].(map[string]any)
			if env["GITHUB_TOKEN"] != "user-token" {
				t.Errorf("user env GITHUB_TOKEN lost: env = %v", env)
			}
			if env["GH_HOST"] != "github.com" {
				t.Errorf("generated env GH_HOST missing: env = %v", env)
			}
			if _, ok := gh["headers"]; !ok {
				t.Errorf("user headers lost: %v", gh)
			}
		})
	}
}
