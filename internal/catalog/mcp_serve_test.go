package catalog

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// The embedded defaults opt into no gated MCP tool: the operator must.
func TestMCPServeOptIns_OffInEmbeddedDefaults(t *testing.T) {
	t.Parallel()
	if got := mustEmbedded(t).MCPServeOptIns(); got != (MCPServeOptIns{}) {
		t.Errorf("embedded MCPServeOptIns() = %+v, want all false", got)
	}
}

// An org overlay sets only the opt-ins it names; the rest stay off.
func TestMCPServeOptIns_OrgOverlay(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		content string
		want    MCPServeOptIns
	}{
		{name: "nix_run only", content: "mcp_serve:\n  allow_nix_run: true\n", want: MCPServeOptIns{AllowNixRun: true}},
		{
			name:    "credential_vend only",
			content: "mcp_serve:\n  allow_credential_vend: true\n",
			want:    MCPServeOptIns{AllowCredentialVend: true},
		},
		{
			name:    "both",
			content: "mcp_serve:\n  allow_nix_run: true\n  allow_credential_vend: true\n",
			want:    MCPServeOptIns{AllowNixRun: true, AllowCredentialVend: true},
		},
		{name: "explicit false", content: "mcp_serve:\n  allow_nix_run: false\n", want: MCPServeOptIns{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cat, err := Load(WithOrgConfigFile(writeUnifiedFile(t, tt.content)))
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			if got := cat.MCPServeOptIns(); got != tt.want {
				t.Errorf("MCPServeOptIns() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// A misspelled opt-in key fails strict parsing rather than being dropped.
func TestMCPServeOptIns_UnknownKeyRejected(t *testing.T) {
	t.Parallel()
	if _, err := Load(WithOrgConfigFile(writeUnifiedFile(t, "mcp_serve:\n  allow_nixrun: true\n"))); err == nil {
		t.Error("Load accepted a misspelled mcp_serve key")
	}
}

// LoadUserScope reads the embedded defaults and the user's org overlay, and
// never the project defaults file, which is repository content.
func TestLoadUserScope_HonorsOrgOverlayOptIns(t *testing.T) {
	org := writeUnifiedFile(t, "mcp_serve:\n  allow_nix_run: true\n  allow_credential_vend: true\n")
	t.Setenv(branding.Get().EnvPrefix+"ORG_CONFIG", org)

	// A project file that a project-including load would reject must not
	// matter to the user scope.
	project := t.TempDir()
	projFile := ProjectConfigPath(project)
	if err := os.MkdirAll(filepath.Dir(projFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projFile, []byte("permission_allow_rules:\n  standard_base:\n    - Bash(*)\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prevRoot := ProjectRoot()
	SetProjectRoot(project)
	t.Cleanup(func() { SetProjectRoot(prevRoot) })

	cat, err := LoadUserScope()
	if err != nil {
		t.Fatalf("LoadUserScope: %v", err)
	}
	want := MCPServeOptIns{AllowNixRun: true, AllowCredentialVend: true}
	if got := cat.MCPServeOptIns(); got != want {
		t.Errorf("MCPServeOptIns() = %+v, want %+v", got, want)
	}
}

// A broken org overlay falls back to the embedded defaults, so every opt-in
// fails closed to off.
func TestLoadUserScope_BrokenOrgOverlayFailsClosed(t *testing.T) {
	org := writeUnifiedFile(t, "mcp_serve:\n  allow_nix_run: true\nnot_a_section: 1\n")
	t.Setenv(branding.Get().EnvPrefix+"ORG_CONFIG", org)

	cat, err := LoadUserScope()
	if err != nil {
		t.Fatalf("LoadUserScope: %v", err)
	}
	if got := cat.MCPServeOptIns(); got != (MCPServeOptIns{}) {
		t.Errorf("MCPServeOptIns() = %+v after a broken overlay, want all false", got)
	}
	if len(cat.MCPServers()) != len(mustEmbedded(t).MCPServers()) {
		t.Error("fallback catalog is not the embedded defaults")
	}
}

// Without an org overlay the user scope is the embedded defaults.
func TestLoadUserScope_NoOverlay(t *testing.T) {
	t.Setenv(branding.Get().EnvPrefix+"ORG_CONFIG", "")
	cat, err := LoadUserScope()
	if err != nil {
		t.Fatalf("LoadUserScope: %v", err)
	}
	if got := cat.MCPServeOptIns(); got != (MCPServeOptIns{}) {
		t.Errorf("MCPServeOptIns() = %+v, want all false", got)
	}
}
