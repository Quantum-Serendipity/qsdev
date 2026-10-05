package instance

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/logging"
	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestSessionProjectRoot is the U01-01/U16-V02 regression for the session
// log: a marker planted in a world-writable ancestor, or a bare data
// directory, must not make the ancestor the project whose .qsdev/logs the
// session log is written to.
func TestSessionProjectRoot(t *testing.T) {
	// Not parallel: t.Chdir changes process-wide state.
	b := branding.Get()
	global := isolateLogging(t)
	tests := []struct {
		name     string
		dirs     []string // slash-separated directories to create
		files    []string // slash-separated regular files to create
		shared   string   // a directory made world-writable and sticky
		cwd      string
		want     string // "" for no project
		unixOnly bool
	}{
		{name: "real project from a subdirectory", files: []string{"repo/" + b.ConfigFile}, dirs: []string{"repo/sub/dir"}, cwd: "repo/sub/dir", want: "repo"},
		{name: "bare data dir ancestor is no project", dirs: []string{"." + b.AppName + "/logs", "victim"}, cwd: "victim"},
		{
			name: "markers in a world-writable ancestor are ignored", unixOnly: true,
			dirs: []string{"shared/." + b.AppName, "shared/" + b.StateDir, "shared/victim"}, files: []string{"shared/" + b.ConfigFile},
			shared: "shared", cwd: "shared/victim",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.unixOnly && runtime.GOOS == "windows" {
				t.Skip("marker trust is ACL-based on Windows and out of scope")
			}
			root := testutil.MarkerFreeTempDir(t)
			for _, d := range tt.dirs {
				if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(d)), 0o755); err != nil {
					t.Fatal(err)
				}
			}
			for _, f := range tt.files {
				p := filepath.Join(root, filepath.FromSlash(f))
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte("version: 1\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			if tt.shared != "" {
				p := filepath.Join(root, tt.shared)
				if err := os.Chmod(p, 0o1777); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(p, 0o755) })
			}
			t.Chdir(filepath.Join(root, filepath.FromSlash(tt.cwd)))
			want := global
			if tt.want != "" {
				want = logging.ProjectLogDir(filepath.Join(root, filepath.FromSlash(tt.want)))
			}
			rt := &Runtime{}
			rt.initCommand(profileTree(), []string{"interactive"}, io.Discard, false)
			t.Cleanup(rt.Finish)
			if rt.logSession == nil || rt.logSession.LogDir != want {
				t.Errorf("session log = %+v, want one in %s", rt.logSession, want)
			}
		})
	}
}

// evilOverlay is a project defaults file (.<app>/defaults.yaml) adding a
// custom hook to the baseline tier, as root-hijack.sh plants it.
const evilOverlay = `custom_hooks:
  - id: evil-hook
    name: Evil
    description: planted by an enclosing project
    entry: ./evil.sh
    language: system
    stages: [pre-commit]
hook_tiers:
  baseline:
    - evil-hook
`

// TestInitCommandSetsCatalogProjectRoot is the U01-01 residual (a)
// regression: the catalog's project defaults layer comes from the root the
// executing command resolves (cmdutil.RootMode), not from a pre-parse
// Enclosing walk. A Here command (init) in a non-git child of a trusted
// project with an overlay acts on the child and must not apply the
// ancestor's overlay; an Enclosing command from a project subdirectory keeps
// its project's overlay; a Here command in a directory with its own overlay
// applies it.
func TestInitCommandSetsCatalogProjectRoot(t *testing.T) {
	// Not parallel: t.Chdir and the global catalog are process-wide state.
	b := branding.Get()
	isolateLogging(t)
	t.Setenv(b.EnvPrefix+"ORG_CONFIG", "")
	overlay := "." + b.AppName + "/defaults.yaml"
	tests := []struct {
		name     string
		files    []string // slash-separated config files to create
		overlays []string // directories (slash-separated) given the overlay
		cwd      string
		args     []string
		wantRoot string
		wantEvil bool
	}{
		{
			name: "Here command in a non-git child of a trusted project", files: []string{"proj/" + b.ConfigFile},
			overlays: []string{"proj"}, cwd: "proj/sub", args: []string{"init"}, wantRoot: "proj/sub",
		},
		{
			name: "Enclosing command from a project subdirectory", files: []string{"repo/" + b.ConfigFile},
			overlays: []string{"repo"}, cwd: "repo/sub/dir", args: []string{"interactive"}, wantRoot: "repo", wantEvil: true,
		},
		{
			name: "Here command in a directory with its own overlay", files: []string{"proj/" + b.ConfigFile},
			overlays: []string{"proj/sub"}, cwd: "proj/sub", args: []string{"init"}, wantRoot: "proj/sub", wantEvil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			catalog.ResetDefault()
			t.Cleanup(catalog.ResetDefault)
			root := testutil.MarkerFreeTempDir(t)
			if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(tt.cwd)), 0o755); err != nil {
				t.Fatal(err)
			}
			for _, f := range tt.files {
				if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(f)), []byte("version: 1\n"), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			for _, d := range tt.overlays {
				p := filepath.Join(root, filepath.FromSlash(d), filepath.FromSlash(overlay))
				if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(p, []byte(evilOverlay), 0o644); err != nil {
					t.Fatal(err)
				}
			}
			t.Chdir(filepath.Join(root, filepath.FromSlash(tt.cwd)))

			rt := &Runtime{}
			rt.initCommand(profileTree(), tt.args, io.Discard, false)
			t.Cleanup(rt.Finish)

			if got, want := catalog.ProjectRoot(), filepath.Join(root, filepath.FromSlash(tt.wantRoot)); got != want {
				t.Errorf("catalog.ProjectRoot() = %q, want %q", got, want)
			}
			cat, err := catalog.Default()
			if err != nil {
				t.Fatalf("catalog.Default: %v", err)
			}
			if got := slices.Contains(cat.HookTiers()["baseline"], "evil-hook"); got != tt.wantEvil {
				t.Errorf("baseline tier has evil-hook = %v, want %v (%v)", got, tt.wantEvil, cat.HookTiers()["baseline"])
			}
		})
	}
}
