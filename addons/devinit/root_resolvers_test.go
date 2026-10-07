package devinit

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/cmdutil"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve"
	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// TestRootResolversAgree checks that the CLI (cmdutil), the enforce hook and
// the MCP server resolve the same project root for the same start directory,
// including a repository without a marker (none of them falls back to the git
// toplevel, which nothing vouches for).
//
// The self-protection hook has no row on purpose: it resolves no qsdev
// project root. It guards the Claude Code session, whose directory is
// CLAUDE_PROJECT_DIR (or the hook's working directory), because that is where
// Claude Code loads .claude/settings.json and runs hooks from, wherever the
// nearest qsdev marker is (internal/selfprotect/rules.hookEnvFromProcess).
func TestRootResolversAgree(t *testing.T) {
	// Not parallel: t.Chdir changes process-wide state.
	b := branding.Get()
	tests := []struct {
		name     string
		paths    []string
		shared   string // a directory made world-writable and sticky (unix only)
		start    string
		want     string
		unixOnly bool
	}{
		{name: "config file in ancestor", paths: []string{b.ConfigFile, "a/b/"}, start: "a/b", want: "."},
		{name: "state dir in ancestor", paths: []string{b.StateDir + "/", "a/"}, start: "a", want: "."},
		{name: "nearest marker wins", paths: []string{b.ConfigFile, "svc/" + b.ConfigFile, "svc/api/"}, start: "svc/api", want: "svc"},
		{name: "marker at git toplevel", paths: []string{".git/", b.ConfigFile, "x/y/"}, start: "x/y", want: "."},
		{name: "ancestor above the git toplevel is not adopted", paths: []string{b.ConfigFile, "repo/" + b.StateDir + "/", "repo/.git/", "repo/sub/"}, start: "repo/sub", want: "repo"},
		{name: "git repo without marker resolves to the start", paths: []string{".git/", "a/b/"}, start: "a/b", want: "a/b"},
		{name: "bare data dir is no marker", paths: []string{"." + b.AppName + "/defaults.yaml", "p/q/"}, start: "p/q", want: "p/q"},
		{
			name:  "markers in a world-writable ancestor are ignored",
			paths: []string{"tmp/" + b.ConfigFile, "tmp/" + b.StateDir + "/", "tmp/victim/"}, shared: "tmp",
			start: "tmp/victim", want: "tmp/victim", unixOnly: true,
		},
		{
			name:  "planted git repo and config in a world-writable ancestor are ignored",
			paths: []string{"tmp/.git/", "tmp/" + b.ConfigFile, "tmp/victim/"}, shared: "tmp",
			start: "tmp/victim", want: "tmp/victim", unixOnly: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.unixOnly && runtime.GOOS == "windows" {
				t.Skip("marker trust is ACL-based on Windows and out of scope")
			}
			root := testutil.IsolatedDir(t)
			plantPaths(t, root, tt.paths...)
			if tt.shared != "" {
				chmodShared(t, filepath.Join(root, filepath.FromSlash(tt.shared)))
			}
			start := filepath.Join(root, filepath.FromSlash(tt.start))
			want := filepath.Join(root, filepath.FromSlash(tt.want))
			t.Setenv(envClaudeProjectDir, "")
			t.Chdir(start)

			cli, err := cmdutil.Project(&cobra.Command{})
			if err != nil {
				t.Fatalf("cmdutil.Project: %v", err)
			}
			mcp, err := mcpserve.ResolveProjectRoot(mcpserve.ResolveOptions{FlagRoot: start, Getenv: func(string) string { return "" }})
			if err != nil {
				t.Fatalf("mcpserve.ResolveProjectRoot: %v", err)
			}
			got := map[string]string{
				"cli":           cli.Root,
				"enforce":       projectRootFrom(start),
				"enforce (cwd)": policyProjectRoot(""),
				"mcp":           mcp,
			}
			for who, root := range got {
				if root != want {
					t.Errorf("%s resolved %q, want %q", who, root, want)
				}
			}
		})
	}
}

// chmodShared makes dir world-writable and sticky, like /tmp, restoring a
// private mode at cleanup so the temp tree can be removed.
func chmodShared(t *testing.T, dir string) {
	t.Helper()
	if err := os.Chmod(dir, 0o1777); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
}
