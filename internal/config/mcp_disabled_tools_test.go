package config

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// mcpToolNames stands in for mcpserve.MountableToolNames in these tests.
var mcpToolNames = []string{"qsdev_nix_run", "qsdev_security_scan", "qsdev_status"}

// TestValidateMCPDisabledTools checks mcp.disabled_tools is validated against
// the MCP tool namespace and tools.disabled against the catalog, so an MCP
// tool can be disabled without failing `qsdev check` (F228).
func TestValidateMCPDisabledTools(t *testing.T) {
	t.Parallel()
	catalog := []string{"gitleaks", "semgrep"}
	tests := []struct {
		name       string
		cfg        types.QsdevConfig
		opts       ValidateOptions
		wantFields []string
		wantValues []string
		wantMsg    string // substring of the first error's message, "" for any
	}{
		{
			name: "known MCP tool is valid",
			cfg: types.QsdevConfig{Version: 2,
				Tools: types.ToolsConfig{Disabled: []string{"gitleaks"}},
				MCP:   types.MCPConfig{DisabledTools: []string{"qsdev_nix_run"}}},
			opts: ValidateOptions{ToolNames: catalog, MCPToolNames: mcpToolNames},
		},
		{
			name:       "misspelled MCP tool",
			cfg:        types.QsdevConfig{Version: 2, MCP: types.MCPConfig{DisabledTools: []string{"qsdev_nixrun"}}},
			opts:       ValidateOptions{ToolNames: catalog, MCPToolNames: mcpToolNames},
			wantFields: []string{"mcp.disabled_tools"},
			wantValues: []string{"qsdev_nixrun"},
		},
		{
			name:       "catalog tool in mcp.disabled_tools",
			cfg:        types.QsdevConfig{Version: 2, MCP: types.MCPConfig{DisabledTools: []string{"gitleaks"}}},
			opts:       ValidateOptions{ToolNames: catalog, MCPToolNames: mcpToolNames},
			wantFields: []string{"mcp.disabled_tools"},
			wantValues: []string{"gitleaks"},
		},
		{
			name:       "MCP tool in tools.disabled",
			cfg:        types.QsdevConfig{Version: 2, Tools: types.ToolsConfig{Disabled: []string{"qsdev_nix_run"}}},
			opts:       ValidateOptions{ToolNames: catalog, MCPToolNames: mcpToolNames},
			wantFields: []string{"tools.disabled"},
			wantValues: []string{"qsdev_nix_run"},
			wantMsg:    "mcp.disabled_tools",
		},
		{
			name: "no MCP names given skips the check",
			cfg:  types.QsdevConfig{Version: 2, MCP: types.MCPConfig{DisabledTools: []string{"anything"}}},
			opts: ValidateOptions{ToolNames: catalog},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			errs := ValidateQsdevConfig(&tt.cfg, tt.opts)
			var fields, values []string
			for _, e := range errs {
				fields = append(fields, e.Field)
				values = append(values, e.Value)
			}
			if !slices.Equal(fields, tt.wantFields) || !slices.Equal(values, tt.wantValues) {
				t.Errorf("errors = %v, want fields %v values %v", errs, tt.wantFields, tt.wantValues)
			}
			if tt.wantMsg != "" && (len(errs) == 0 || !strings.Contains(errs[0].Message, tt.wantMsg)) {
				t.Errorf("errors = %v, want the first to mention %q", errs, tt.wantMsg)
			}
		})
	}
}

// TestParseMCPDisabledTools checks the strict decoder accepts the mcp block at
// both schema versions (a v1 file is migrated in memory).
func TestParseMCPDisabledTools(t *testing.T) {
	t.Parallel()
	for _, version := range []string{"1", "2"} {
		t.Run("version "+version, func(t *testing.T) {
			t.Parallel()
			data := "version: " + version + "\nmcp:\n  disabled_tools:\n    - qsdev_nix_run\n"
			cfg, err := ParseQsdevConfigBytes([]byte(data))
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			if !slices.Equal(cfg.MCP.DisabledTools, []string{"qsdev_nix_run"}) {
				t.Errorf("mcp.disabled_tools = %v, want [qsdev_nix_run]", cfg.MCP.DisabledTools)
			}
		})
	}
}

// TestMCPDisabledToolsSurvivesResolveAndRecreate checks the canonical resolver
// keeps mcp.disabled_tools (as its own copy), and that re-creating a project
// carries the committed list, which answers cannot express.
func TestMCPDisabledToolsSurvivesResolveAndRecreate(t *testing.T) {
	t.Parallel()
	committed := &types.QsdevConfig{Version: 2, MCP: types.MCPConfig{DisabledTools: []string{"qsdev_nix_run"}}}

	resolved, err := ResolveConfig(&types.QsdevConfig{}, committed, nil)
	if err != nil {
		t.Fatalf("ResolveConfig: %v", err)
	}
	if !slices.Equal(resolved.Config.MCP.DisabledTools, []string{"qsdev_nix_run"}) {
		t.Errorf("resolved mcp.disabled_tools = %v, want [qsdev_nix_run]", resolved.Config.MCP.DisabledTools)
	}
	resolved.Config.MCP.DisabledTools[0] = "changed"
	if committed.MCP.DisabledTools[0] != "qsdev_nix_run" {
		t.Error("resolved config aliases the committed mcp.disabled_tools")
	}

	fresh := types.QsdevConfig{Version: 2}
	PreserveCommittedPolicy(&fresh, committed)
	if !slices.Equal(fresh.MCP.DisabledTools, []string{"qsdev_nix_run"}) {
		t.Errorf("re-created mcp.disabled_tools = %v, want [qsdev_nix_run]", fresh.MCP.DisabledTools)
	}
}
