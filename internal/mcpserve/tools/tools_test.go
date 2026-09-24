package tools_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/tools"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

func regNames(t *testing.T, modules []string) []string {
	t.Helper()
	regs, err := tools.Select(modules, t.TempDir(), nil, tools.Options{})
	if err != nil {
		t.Fatalf("Select(%v): %v", modules, err)
	}
	names := make([]string, len(regs))
	for i, r := range regs {
		names[i] = r.Name
	}
	return names
}

// TestSelect proves a module selection mounts exactly that module's tools,
// no selection mounts every module, and an unknown module is refused.
func TestSelect(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		modules []string
		want    []string
	}{
		{"agent-postmortem", []string{"agent-postmortem"}, []string{"analyze_session", "list_failure_patterns", "generate_verification_checklist"}},
		{"version-sentinel", []string{"version-sentinel"}, []string{"check_versions", "detect_drift", "manifest_coverage", "version_history"}},
		{"status", []string{tools.ModuleStatus}, []string{"qsdev_status", "qsdev_devenv_doctor"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := regNames(t, tt.modules); !slices.Equal(got, tt.want) {
				t.Errorf("Select(%v) = %v, want %v", tt.modules, got, tt.want)
			}
		})
	}

	t.Run("no selection is every module", func(t *testing.T) {
		t.Parallel()
		all := regNames(t, nil)
		for _, want := range []string{"qsdev_security_scan", "analyze_session", "check_versions"} {
			if !slices.Contains(all, want) {
				t.Errorf("Select(nil) = %v, missing %q", all, want)
			}
		}
	})

	t.Run("unknown module", func(t *testing.T) {
		t.Parallel()
		_, err := tools.Select([]string{"agent-postmortem", "no-such-module"}, t.TempDir(), nil, tools.Options{})
		if err == nil || !strings.Contains(err.Error(), "no-such-module") {
			t.Errorf("Select with an unknown module: err = %v, want one naming it", err)
		}
	})
}

// TestNamesCoverEveryModule proves mcp.disabled_tools can name every module's
// tools: Names is the namespace `qsdev check` validates the list against.
func TestNamesCoverEveryModule(t *testing.T) {
	t.Parallel()
	names := tools.Names()
	for _, module := range tools.ModuleNames() {
		for _, tool := range regNames(t, []string{module}) {
			if !slices.Contains(names, tool) {
				t.Errorf("Names() lacks %q of module %q", tool, module)
			}
		}
	}
}

// TestCatalogModuleServersNameKnownModules proves every catalog MCP server that
// runs qsdev's own server launches `mcp serve` (never a bespoke subcommand
// outside the middleware chain) and restricts it only to modules that exist.
func TestCatalogModuleServersNameKnownModules(t *testing.T) {
	t.Parallel()
	cat, err := catalog.Default()
	if err != nil {
		t.Fatal(err)
	}
	app := branding.Get().AppName
	found := 0
	for name, def := range cat.MCPServers() {
		if def.Command != app {
			continue
		}
		found++
		if len(def.Args) < 2 || def.Args[0] != "mcp" || def.Args[1] != "serve" {
			t.Errorf("catalog server %q runs %s %v, want %s mcp serve", name, app, def.Args, app)
			continue
		}
		for i, arg := range def.Args {
			if arg == "--module" && i+1 < len(def.Args) && !slices.Contains(tools.ModuleNames(), def.Args[i+1]) {
				t.Errorf("catalog server %q selects unknown module %q", name, def.Args[i+1])
			}
		}
	}
	if found == 0 {
		t.Errorf("no catalog MCP server runs %s; the module servers are missing", app)
	}
}
