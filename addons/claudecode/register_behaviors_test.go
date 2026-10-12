package claudecode

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
)

// TestRegisterMCPServerContentReturnsCatalogError: a project defaults file
// the catalog rejects, or cannot parse, makes the provider return the
// catalog's load error, which names the file, wrapped with what it was doing,
// instead of panicking with a misattributed "embedded catalog" message.
// toolreg.Default then reports it. Not parallel: it swaps the global catalog.
func TestRegisterMCPServerContentReturnsCatalogError(t *testing.T) {
	tests := []struct {
		name    string
		overlay string
		want    string
	}{
		{"rejected", "keep_vars: [AWS_SECRET_ACCESS_KEY]\n", "may only add or tighten"},
		{"malformed", "tools: [\n", "defaults.yaml"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			useProjectOverlay(t, tt.overlay)

			err := registerToolBehaviors(toolreg.NewRegistry())
			if err == nil {
				t.Fatal("registerToolBehaviors() = nil, want the catalog load error")
			}
			for _, want := range []string{"registering MCP server content", tt.want} {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("error %q does not contain %q", err, want)
				}
			}
			if strings.Contains(err.Error(), "embedded") {
				t.Errorf("error misattributes the failure to the embedded catalog: %v", err)
			}
		})
	}
}

// useProjectOverlay makes the global catalog load overlay as the project
// defaults file of a fresh project, and restores the catalog afterwards.
func useProjectOverlay(t *testing.T, overlay string) {
	t.Helper()
	root := t.TempDir()
	path := catalog.ProjectConfigPath(root)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(overlay), 0o644); err != nil {
		t.Fatal(err)
	}
	// ResetDefault also clears the project root, so it goes first.
	prevRoot := catalog.ProjectRoot()
	catalog.ResetDefault()
	if err := catalog.SetProjectRoot(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		catalog.ResetDefault()
		if err := catalog.SetProjectRoot(prevRoot); err != nil {
			t.Error(err)
		}
	})
}
