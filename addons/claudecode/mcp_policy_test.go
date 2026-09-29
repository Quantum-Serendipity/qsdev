package claudecode_test

import (
	"encoding/json"
	"maps"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestGenerateMcpJson_ClientPolicy verifies the client MCP policy removes
// blocked servers whether they come from the answers or the addon config.
func TestGenerateMcpJson_ClientPolicy(t *testing.T) {
	t.Parallel()
	custom := claudecode.MCPServerConfig{Name: "my-tool", Command: "/usr/local/bin/my-tool"}
	tests := []struct {
		name   string
		policy types.MCPPolicy
		want   []string
	}{
		{name: "no policy", want: []string{"context7", "github", "my-tool"}},
		{name: "named block", policy: types.MCPPolicy{Blocked: []string{"github", "my-tool"}}, want: []string{"context7"}},
		{
			name:   "wildcard with allow",
			policy: types.MCPPolicy{Blocked: []string{types.MCPWildcard}, Allowed: []string{"my-tool"}},
			want:   []string{"my-tool"},
		},
		// Under a policy the file is still written, empty, so a merge over an
		// existing .mcp.json strips the forbidden servers.
		{name: "wildcard blocks everything", policy: types.MCPPolicy{Blocked: []string{types.MCPWildcard}}, want: []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			answers := types.WizardAnswers{MCPServers: []string{"context7", "github"}, MCPPolicy: tt.policy}
			gf, err := claudecode.GenerateMcpJson(answers, claudecode.NewConfig(claudecode.WithMCPServer(custom)))
			if err != nil {
				t.Fatal(err)
			}
			if gf == nil {
				t.Fatal("expected a .mcp.json")
			}
			if got := mcpServerNames(t, gf.Content); !slices.Equal(got, tt.want) {
				t.Errorf("servers = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestGenerate_ClientPolicyBlocksToolServers verifies the policy also removes
// servers an enabled tool or agent tool would inject, which never pass
// through answers.MCPServers before generation.
func TestGenerate_ClientPolicyBlocksToolServers(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	answers := types.WizardAnswers{
		Tier:         "full",
		Languages:    []types.LanguageChoice{{Name: "go"}},
		MCPServers:   []string{"context7"},
		EnabledTools: map[string]bool{"socket": true},
		AgentTools:   types.AgentToolsAnswers{SembleEnabled: true, SembleMode: "mcp"},
		MCPPolicy:    types.MCPPolicy{Blocked: []string{types.MCPWildcard}, Allowed: []string{"context7"}},
	}
	files, err := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{}).Generate(answers)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if f.Path != ".mcp.json" {
			continue
		}
		if got := mcpServerNames(t, f.Content); !slices.Equal(got, []string{"context7"}) {
			t.Errorf(".mcp.json servers = %v, want only context7", got)
		}
		return
	}
	t.Fatal("expected a .mcp.json for the allowed server")
}

func mcpServerNames(t *testing.T, content []byte) []string {
	t.Helper()
	var mcp claudecode.McpJSON
	if err := json.Unmarshal(content, &mcp); err != nil {
		t.Fatalf("parsing .mcp.json: %v\n%s", err, content)
	}
	return slices.Sorted(maps.Keys(mcp.MCPServers))
}

// TestGenerate_ClientPolicyAlwaysWritesMcpJson checks a standard-tier project
// with no MCP servers still gets a .mcp.json under a client policy, so the
// writers can strip forbidden servers from a checked-in file, and that the
// policy alone provisions no server (an enabled tool's server is only
// injected where it would be without a policy).
func TestGenerate_ClientPolicyAlwaysWritesMcpJson(t *testing.T) {
	reg := newTestRegistry(t, goMock())
	for _, policy := range []types.MCPPolicy{{}, {Blocked: []string{"github"}}} {
		answers := types.WizardAnswers{
			Tier:         "standard",
			Languages:    []types.LanguageChoice{{Name: "go"}},
			EnabledTools: map[string]bool{"socket": true},
			MCPPolicy:    policy,
		}
		files, err := claudecode.NewClaudeCodeGenerator(reg, claudecode.Config{}).Generate(answers)
		if err != nil {
			t.Fatal(err)
		}
		i := slices.IndexFunc(files, func(f types.GeneratedFile) bool { return f.Path == ".mcp.json" })
		if want := !policy.IsZero(); (i >= 0) != want {
			t.Errorf("policy %+v: .mcp.json written = %v, want %v", policy, i >= 0, want)
		}
		if i >= 0 {
			if got := mcpServerNames(t, files[i].Content); len(got) != 0 {
				t.Errorf("policy %+v: .mcp.json servers = %v, want none", policy, got)
			}
		}
	}
}
