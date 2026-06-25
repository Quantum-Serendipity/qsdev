package mcpserve_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/projectctx"

	// Blank-import every framework adapter so their init() self-registration runs
	// into spi.DefaultRegistry() before the server is constructed. This mirrors
	// the wiring cmd/qsdev/main.go performs at the program entry point. Note these
	// adapter packages never import the mcpserve server root (they import only the
	// spi seam), so importing them from this external test creates no cycle.
	_ "github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/claudecode"
	_ "github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/cline"
	_ "github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/codex"
	_ "github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/cursor"
	_ "github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/windsurf"
)

// TestMultiAdapterToolCatalog proves that constructing the universal server in
// multi-adapter mode and mounting the generic project context surface yields the
// full 23-tool catalog: 6 generic + 5 Claude Code + 3 each for Cursor, Windsurf,
// Cline, and Codex. It is the integration-level assertion the Unit 32.5
// verification calls for (the multi-adapter --multi-adapter flag forces every
// registered adapter to mount regardless of project markers; see WithMultiAdapter).
func TestMultiAdapterToolCatalog(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/x\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".qsdev.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatalf("write .qsdev.yaml: %v", err)
	}

	pc, err := projectctx.NewProjectContext(dir)
	if err != nil {
		t.Fatalf("NewProjectContext: %v", err)
	}

	srv := mcpserve.New(mcpserve.WithProjectRoot(dir), mcpserve.WithMultiAdapter(true))
	srv.MountProjectContext(pc)

	tools := srv.MCPServer().ListTools()

	generic := []string{
		"qsdev_project_info", "qsdev_doctor", "qsdev_config_show",
		"qsdev_mcp_list", "qsdev_tool_list", "qsdev_detect",
	}
	claudecode := []string{
		"qsdev_cc_permissions", "qsdev_cc_hooks", "qsdev_cc_context_budget",
		"qsdev_cc_config_render", "qsdev_cc_enforcement_gaps",
	}
	cursor := []string{"qsdev_cursor_info", "qsdev_cursor_config", "qsdev_cursor_capabilities"}
	windsurf := []string{"qsdev_windsurf_info", "qsdev_windsurf_config", "qsdev_windsurf_capabilities"}
	cline := []string{"qsdev_cline_info", "qsdev_cline_config", "qsdev_cline_capabilities"}
	codex := []string{"qsdev_codex_info", "qsdev_codex_config", "qsdev_codex_capabilities"}

	families := map[string][]string{
		"generic":    generic,
		"claudecode": claudecode,
		"cursor":     cursor,
		"windsurf":   windsurf,
		"cline":      cline,
		"codex":      codex,
	}
	for family, names := range families {
		for _, name := range names {
			if _, ok := tools[name]; !ok {
				t.Errorf("family %s: tool %q not mounted; mounted=%v", family, name, toolNames(tools))
			}
		}
	}

	const want = 23 // 6 + 5 + 3 + 3 + 3 + 3
	if len(tools) != want {
		t.Errorf("mounted %d tools, want %d; mounted=%v", len(tools), want, toolNames(tools))
	}
}

func toolNames[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
