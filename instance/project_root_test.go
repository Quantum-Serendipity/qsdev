package instance

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

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
