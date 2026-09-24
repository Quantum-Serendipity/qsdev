package mcpserve_test

import (
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/projectctx"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// TestMountableToolNames checks that the mcp.disabled_tools namespace covers
// every surface the server mounts (project context, security/devenv/status and
// every shipped adapter), is sorted and unique, and holds no catalog tool name.
func TestMountableToolNames(t *testing.T) {
	t.Parallel()
	all := adapters.All()
	names := mcpserve.MountableToolNames(all)

	if !slices.IsSorted(names) || len(slices.Compact(slices.Clone(names))) != len(names) {
		t.Errorf("names are not sorted and unique: %v", names)
	}

	var want []string
	want = append(want, projectctx.ToolNames()...)
	want = append(want, tools.Names()...)
	for _, a := range all {
		for _, r := range a.Tools() {
			want = append(want, r.Name)
		}
	}
	for _, name := range want {
		if !slices.Contains(names, name) {
			t.Errorf("mountable tool %q missing from %v", name, names)
		}
	}

	tests := []struct {
		name string
		tool string
		want bool
	}{
		{name: "devenv tool", tool: "qsdev_nix_run", want: true},
		{name: "security tool", tool: "qsdev_security_scan", want: true},
		{name: "project context tool", tool: "qsdev_project_info", want: true},
		{name: "catalog tool is a different namespace", tool: "gitleaks", want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := slices.Contains(names, tt.tool); got != tt.want {
				t.Errorf("contains %q = %v, want %v", tt.tool, got, tt.want)
			}
		})
	}
}

// TestMountableToolNamesMatchesMountedServer checks the static name set
// against what a real server mounts, so the two cannot drift.
func TestMountableToolNamesMatchesMountedServer(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	srv := mcpserve.New(mcpserve.WithProjectRoot(dir))
	pc, err := projectctx.NewProjectContext(dir)
	if err != nil {
		t.Fatalf("NewProjectContext: %v", err)
	}
	srv.MountProjectContext(pc)
	srv.MountTools(tools.All(dir, nil, tools.Options{
		CredentialVend: types.CredentialVendConfig{Enabled: true},
		NixRun:         true,
	}))

	mountable := mcpserve.MountableToolNames(spi.DefaultRegistry().All())
	for name := range srv.MCPServer().ListTools() {
		if !slices.Contains(mountable, name) {
			t.Errorf("mounted tool %q missing from MountableToolNames", name)
		}
	}
}
