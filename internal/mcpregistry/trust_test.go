package mcpregistry

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// testTrusted is a trusted-definition table with stdio and remote entries.
func testTrusted() map[string][]LaunchSpec {
	return map[string][]LaunchSpec{
		"context7": {{Command: "npx", Args: []string{"-y", "@upstash/context7-mcp"}}},
		"github":   {{Command: "gh-mcp", Args: []string{"stdio"}, Env: map[string]string{"GITHUB_TOKEN": "${GITHUB_TOKEN}"}}},
		"remote": {{
			URL:     "https://mcp.example.invalid/mcp",
			Headers: map[string]string{"Authorization": "Bearer ${REMOTE_TOKEN}"},
		}},
	}
}

func TestMatchesTrusted(t *testing.T) {
	t.Parallel()

	specs := []LaunchSpec{
		{Command: "srv", Args: []string{"--stdio"}, Env: map[string]string{"K": "${K}"}},
		{URL: "https://mcp.example.invalid/mcp", Headers: map[string]string{"X-Key": "${KEY}"}},
	}
	tests := []struct {
		name string
		cfg  mcphealth.ServerConfig
		want bool
	}{
		{"stdio identical", mcphealth.ServerConfig{Command: "srv", Args: []string{"--stdio"}, Env: map[string]string{"K": "${K}"}}, true},
		{"stdio different command", mcphealth.ServerConfig{Command: "evil", Args: []string{"--stdio"}, Env: map[string]string{"K": "${K}"}}, false},
		{"stdio extra arg", mcphealth.ServerConfig{Command: "srv", Args: []string{"--stdio", "-x"}, Env: map[string]string{"K": "${K}"}}, false},
		{"stdio different env", mcphealth.ServerConfig{Command: "srv", Args: []string{"--stdio"}, Env: map[string]string{"K": "v"}}, false},
		{"empty command never matches", mcphealth.ServerConfig{}, false},
		{"http identical", mcphealth.ServerConfig{URL: "https://mcp.example.invalid/mcp", Headers: map[string]string{"X-Key": "${KEY}"}}, true},
		{"http different url", mcphealth.ServerConfig{URL: "https://evil.invalid/mcp", Headers: map[string]string{"X-Key": "${KEY}"}}, false},
		{"http header template changed", mcphealth.ServerConfig{URL: "https://mcp.example.invalid/mcp", Headers: map[string]string{"X-Key": "${AWS_SECRET_ACCESS_KEY}"}}, false},
		{"http added env", mcphealth.ServerConfig{URL: "https://mcp.example.invalid/mcp", Headers: map[string]string{"X-Key": "${KEY}"}, Env: map[string]string{"A": "b"}}, false},
		{"http url is not a stdio match", mcphealth.ServerConfig{Command: "srv", Args: []string{"--stdio"}, Env: map[string]string{"K": "${K}"}, URL: "https://evil.invalid/mcp"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := MatchesTrusted(tc.cfg, specs); got != tc.want {
				t.Errorf("MatchesTrusted = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestLaunchVariants(t *testing.T) {
	t.Parallel()

	env := map[string]string{"K": "${K}"}
	tests := []struct {
		name string
		def  catalog.MCPServerDef
		want []LaunchSpec
	}{
		{
			name: "stdio launcher",
			def:  catalog.MCPServerDef{Command: "npx", Args: []string{"pkg@1.0.0"}, Env: env, Transport: "stdio"},
			want: []LaunchSpec{{Command: "npx", Args: []string{"pkg@1.0.0"}, Env: env}},
		},
		{
			name: "http",
			def:  catalog.MCPServerDef{URL: "https://mcp.example.invalid/", Transport: "http", Env: env},
			want: []LaunchSpec{{URL: "https://mcp.example.invalid/", Env: env}},
		},
		{
			name: "installed bin",
			def:  catalog.MCPServerDef{Command: "uvx", Args: []string{"srv==1.0.0", "--db", "x"}, Version: "1.0.0", Bin: "srv", BinArgs: []string{"--db", "x"}},
			want: []LaunchSpec{
				{Command: "uvx", Args: []string{"srv==1.0.0", "--db", "x"}},
				{Command: "srv", Args: []string{"--db", "x"}},
			},
		},
		{
			name: "bin without version is not installable",
			def:  catalog.MCPServerDef{Command: "uvx", Args: []string{"srv"}, Bin: "srv"},
			want: []LaunchSpec{{Command: "uvx", Args: []string{"srv"}}},
		},
		{
			name: "optional args",
			def: catalog.MCPServerDef{
				Command: "uvx", Args: []string{"srv==1.0.0"}, Version: "1.0.0", Bin: "srv",
				OptionalArgs: [][]string{{"--text"}, {"--a", "b"}},
			},
			want: []LaunchSpec{
				{Command: "uvx", Args: []string{"srv==1.0.0"}},
				{Command: "uvx", Args: []string{"srv==1.0.0", "--text"}},
				{Command: "uvx", Args: []string{"srv==1.0.0", "--a", "b"}},
				{Command: "srv"},
				{Command: "srv", Args: []string{"--text"}},
				{Command: "srv", Args: []string{"--a", "b"}},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := LaunchVariants(tc.def); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("LaunchVariants =\n%#v\nwant\n%#v", got, tc.want)
			}
		})
	}
}

// TestLaunchVariants_DoesNotAliasDef guards against appending optional
// arguments into the catalog definition's own Args backing array.
func TestLaunchVariants_DoesNotAliasDef(t *testing.T) {
	t.Parallel()
	args := make([]string, 1, 8)
	args[0] = "srv==1.0.0"
	def := catalog.MCPServerDef{Command: "uvx", Args: args, OptionalArgs: [][]string{{"--x"}, {"--y"}}}
	got := LaunchVariants(def)
	if want := []string{"srv==1.0.0", "--x"}; !reflect.DeepEqual(got[1].Args, want) {
		t.Errorf("first optional variant = %v, want %v", got[1].Args, want)
	}
}

// TestTrustedDefinitions_ExcludesProjectOverlay: the trusted set is the
// embedded catalog, the user's organization overlay and the binary's own
// servers. The project defaults file is repository content and never
// contributes, even when it is broken.
func TestTrustedDefinitions_ExcludesProjectOverlay(t *testing.T) {
	tmp := t.TempDir()
	org := filepath.Join(tmp, "org.yaml")
	orgYAML := "mcp_servers:\n  orgsrv:\n    display_name: Org\n    category: agent\n    description: org server\n    command: org-mcp\n    args: [\"--stdio\"]\n    transport: stdio\n"
	if err := os.WriteFile(org, []byte(orgYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(branding.Get().EnvPrefix+"ORG_CONFIG", org)

	project := filepath.Join(tmp, "project")
	projYAML := "mcp_servers:\n  evil:\n    command: evil\n    transport: stdio\n"
	projFile := catalog.ProjectConfigPath(project)
	if err := os.MkdirAll(filepath.Dir(projFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projFile, []byte(projYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	catalog.ResetDefault()
	t.Cleanup(catalog.ResetDefault)
	if err := catalog.SetProjectRoot(project); err != nil {
		t.Fatalf("SetProjectRoot: %v", err)
	}

	extra := map[string][]LaunchSpec{"bin-srv": {{Command: "bin-srv"}}}
	trusted := TrustedDefinitions(extra)

	if _, ok := trusted["evil"]; ok {
		t.Error("project overlay server is trusted")
	}
	if want := []LaunchSpec{{Command: "org-mcp", Args: []string{"--stdio"}}}; !reflect.DeepEqual(trusted["orgsrv"], want) {
		t.Errorf("org server specs = %#v, want %#v", trusted["orgsrv"], want)
	}
	if !reflect.DeepEqual(trusted["bin-srv"], extra["bin-srv"]) {
		t.Errorf("extra specs = %#v, want %#v", trusted["bin-srv"], extra["bin-srv"])
	}
	embedded, err := catalog.LoadEmbeddedOnly()
	if err != nil {
		t.Fatal(err)
	}
	for name, def := range embedded.MCPServers() {
		if !reflect.DeepEqual(trusted[name], LaunchVariants(def)) {
			t.Errorf("%s specs = %#v, want LaunchVariants %#v", name, trusted[name], LaunchVariants(def))
		}
	}
}

// TestTrustedDefinitions_BrokenOrgOverlayKeepsEmbedded: TrustedDefinitions
// reads the user scope through catalog.LoadUserScope, so a broken org overlay
// drops only the overlay's servers; the embedded ones stay trusted.
func TestTrustedDefinitions_BrokenOrgOverlayKeepsEmbedded(t *testing.T) {
	org := filepath.Join(t.TempDir(), "org.yaml")
	orgYAML := "mcp_servers:\n  orgsrv:\n    command: org-mcp\n    transport: stdio\nnot_a_section: 1\n"
	if err := os.WriteFile(org, []byte(orgYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(branding.Get().EnvPrefix+"ORG_CONFIG", org)

	trusted := TrustedDefinitions(nil)
	if _, ok := trusted["orgsrv"]; ok {
		t.Error("server from a broken org overlay is trusted")
	}
	embedded, err := catalog.LoadEmbeddedOnly()
	if err != nil {
		t.Fatal(err)
	}
	for name, def := range embedded.MCPServers() {
		if !reflect.DeepEqual(trusted[name], LaunchVariants(def)) {
			t.Errorf("%s specs = %#v, want LaunchVariants %#v", name, trusted[name], LaunchVariants(def))
		}
	}
}
