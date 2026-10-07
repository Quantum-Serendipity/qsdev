package devinit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestInitPlanNamesProjectDefaults: init's plan names the project defaults
// file the catalog applies (the layer of the root the runtime set for the
// command), and says nothing about project defaults when none applies.
func TestInitPlanNamesProjectDefaults(t *testing.T) {
	// Not parallel: the working directory and the global catalog are
	// process-wide.
	t.Setenv(branding.Get().EnvPrefix+"ORG_CONFIG", "")
	tests := []struct {
		name    string
		overlay bool
	}{
		{name: "project defaults applied", overlay: true},
		{name: "no project defaults"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := testutil.IsolatedDir(t)
			overlay := catalog.ProjectConfigPath(dir)
			if tt.overlay {
				if err := os.MkdirAll(filepath.Dir(overlay), 0o755); err != nil {
					t.Fatal(err)
				}
				content := "permission_deny_rules:\n  npx:\n    - Bash(project-only-deny *)\n"
				if err := os.WriteFile(overlay, []byte(content), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			catalog.ResetDefault()
			t.Cleanup(catalog.ResetDefault)
			if err := catalog.SetProjectRoot(dir); err != nil {
				t.Fatal(err)
			}

			out, err := executeInitCmd(t, dir, "--yes", "--dry-run", "--lang", "go")
			if err != nil {
				t.Fatalf("init --dry-run: %v\n%s", err, out)
			}
			want := "Project defaults: " + overlay
			if got := strings.Contains(out, want); got != tt.overlay {
				t.Errorf("plan names %q = %v, want %v:\n%s", want, got, tt.overlay, out)
			}
			if !tt.overlay && strings.Contains(out, "Project defaults:") {
				t.Errorf("plan names project defaults when none applies:\n%s", out)
			}
		})
	}
}
