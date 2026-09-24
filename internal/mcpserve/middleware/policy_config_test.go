package middleware

import (
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestPolicyFromConfig checks that the MCP deny set comes from
// mcp.disabled_tools alone: tools.disabled names qsdev catalog tools, and a
// catalog disable must never become a phantom MCP deny (F228).
func TestPolicyFromConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		cfg      *types.QsdevConfig
		wantNil  bool
		wantDeny []string
	}{
		{name: "nil config", cfg: nil, wantNil: true},
		{name: "nothing disabled", cfg: &types.QsdevConfig{}, wantNil: true},
		{
			name:    "catalog disable only",
			cfg:     &types.QsdevConfig{Tools: types.ToolsConfig{Disabled: []string{"gitleaks"}}},
			wantNil: true,
		},
		{
			name: "mcp disables",
			cfg: &types.QsdevConfig{
				Tools: types.ToolsConfig{Disabled: []string{"gitleaks"}},
				MCP:   types.MCPConfig{DisabledTools: []string{"qsdev_security_scan", "qsdev_nix_run"}},
			},
			wantDeny: []string{"qsdev_nix_run", "qsdev_security_scan"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			p := PolicyFromConfig(tt.cfg)
			if (p == nil) != tt.wantNil {
				t.Fatalf("policy = %+v, want nil=%v", p, tt.wantNil)
			}
			if got := p.DenyToolSet(); !slices.Equal(got, tt.wantDeny) {
				t.Errorf("deny set = %v, want %v", got, tt.wantDeny)
			}
		})
	}
}

// TestPolicyFromConfigCopiesDenyList checks the policy does not alias the
// config's slice, so a later edit of the parsed config cannot change what the
// running Guardrail enforces.
func TestPolicyFromConfigCopiesDenyList(t *testing.T) {
	t.Parallel()
	cfg := &types.QsdevConfig{MCP: types.MCPConfig{DisabledTools: []string{"qsdev_nix_run"}}}
	p := PolicyFromConfig(cfg)
	cfg.MCP.DisabledTools[0] = "qsdev_status"
	if got := p.DenyToolSet(); !slices.Equal(got, []string{"qsdev_nix_run"}) {
		t.Errorf("deny set = %v, want [qsdev_nix_run]", got)
	}
}
