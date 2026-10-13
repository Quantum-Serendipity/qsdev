package devinit

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/check"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// breakProjectDefaults writes a project defaults file the catalog rejects into
// dir and points the catalog at it, so the next registry build fails as it
// does for a real run in that project. The cached catalog and registry are
// rebuilt once the test ends.
func breakProjectDefaults(t *testing.T, dir string) {
	t.Helper()
	path := catalog.ProjectConfigPath(dir)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("tier_to_compliance:\n  standard: nonexistent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	prevRoot := catalog.ProjectRoot()
	t.Cleanup(func() {
		catalog.ResetDefault() // also clears the project root
		if err := catalog.SetProjectRoot(prevRoot); err != nil {
			t.Errorf("restoring the catalog project root: %v", err)
		}
		toolreg.ResetDefaultRegistry()
	})
	catalog.ResetDefault()
	if err := catalog.SetProjectRoot(dir); err != nil {
		t.Fatal(err)
	}
	toolreg.ResetDefaultRegistry()
	if _, err := toolreg.Default(); err == nil {
		t.Fatal("test setup: the project defaults file should break the registry")
	}
}

// TestCheckCmd_BrokenProjectDefaults pins U08-WS3: with a project defaults
// file the catalog rejects, check reports a failing
// config_integrity/config_catalog result and exits 1 in every output format,
// still runs the checks that need no catalog, and applies no auto-fix (the
// fixes regenerate from the catalog it could not load).
func TestCheckCmd_BrokenProjectDefaults(t *testing.T) {
	dir := initLifecycleProject(t)
	// A missing manifest is auto-fixable without the catalog, so an
	// auto-fix that ran would rewrite it.
	manifest := filepath.Join(dir, state.ManifestFile())
	if err := os.Remove(manifest); err != nil {
		t.Fatal(err)
	}
	breakProjectDefaults(t, dir)

	tests := []struct {
		name     string
		args     []string
		want     []string
		wantExit func(error) bool
	}{
		{
			name: "human",
			args: []string{"--auto-fix"},
			want: []string{"[FAIL] config_catalog: loading " + branding.Get().AppName + " defaults:", "defaults validate", "generated_manifest"},
			wantExit: func(err error) bool {
				var cf *check.CheckFailedError
				return errors.As(err, &cf) && cf.ExitCode() == 1
			},
		},
		{
			name: "json",
			args: []string{"--format", "json", "--auto-fix"},
			want: []string{`"name": "config_catalog"`, `"status": "fail"`, "defaults validate", "generated_manifest"},
			wantExit: func(err error) bool {
				var ee *ExitError
				return errors.As(err, &ee) && ee.Code == 1
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			out, err := runLifecycleCmd(t, dir, checkCmd(), tt.args...)
			if !tt.wantExit(err) {
				t.Errorf("check error = %#v, want exit 1", err)
			}
			for _, s := range tt.want {
				if !strings.Contains(out, s) {
					t.Errorf("output lacks %q", s)
				}
			}
			if strings.Contains(out, "panic") {
				t.Error("output mentions a panic")
			}
			if _, err := os.Stat(manifest); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("auto-fix ran: manifest stat err = %v", err)
			}
			if t.Failed() {
				t.Logf("output:\n%s", out)
			}
		})
	}
}
