package cmdutil

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/spf13/cobra"

	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
)

// plant creates each slash-separated path below root: a trailing "/" makes a
// directory, anything else a small regular file.
func plant(t *testing.T, root string, paths ...string) {
	t.Helper()
	for _, m := range paths {
		p := filepath.Join(root, filepath.FromSlash(m))
		if m[len(m)-1] == '/' {
			if err := os.MkdirAll(p, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("version: 1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestProject_LazyEnclosing(t *testing.T) {
	// Not parallel: t.Chdir and t.Setenv change process-wide state.
	tests := []struct {
		name      string
		markers   []string // relative paths to create; a trailing "/" makes a directory
		shared    []string // directories made world-writable and sticky (unix only)
		unixOnly  bool     // relies on the unix ownership/permission trust check
		home      string   // relative to the temp root; "" = a directory outside it
		cwd       string   // relative to the temp root
		want      string   // relative to the temp root
		wantFound bool
	}{
		{name: "no project falls back to cwd", cwd: "a/b", want: "a/b"},
		{name: "config file at cwd", markers: []string{".qsdev.yaml"}, cwd: ".", want: ".", wantFound: true},
		{name: "config file in ancestor", markers: []string{".qsdev.yaml"}, cwd: "src/pkg", want: ".", wantFound: true},
		{name: "state dir in ancestor", markers: []string{".devinit/"}, cwd: "src", want: ".", wantFound: true},
		{name: "nearest project wins", markers: []string{".qsdev.yaml", "sub/.qsdev.yaml"}, cwd: "sub/x", want: "sub", wantFound: true},
		{name: "config name as directory is not a marker", markers: []string{"src/.qsdev.yaml/"}, cwd: "src", want: "src"},
		{name: "bare data dir in ancestor is not a marker", markers: []string{".qsdev/defaults.yaml"}, cwd: "internal/check", want: "internal/check"},
		{name: "global data dir in home is not a project", markers: []string{".qsdev/logs/"}, home: ".", cwd: "newproj", want: "newproj"},
		{name: "ancestor config above git child is ignored", markers: []string{".qsdev.yaml", "child/.git/"}, cwd: "child", want: "child"},
		{name: "marker at git toplevel found from subdirectory", markers: []string{".qsdev.yaml", ".git/"}, cwd: "sub/dir", want: ".", wantFound: true},
		{
			name:    "markers planted in a world-writable ancestor are ignored",
			markers: []string{"shared/.qsdev.yaml", "shared/.devinit/", "shared/.qsdev/defaults.yaml"},
			shared:  []string{"shared"}, unixOnly: true,
			cwd: "shared/victim", want: "shared/victim",
		},
		{
			name:    "trusted project below a world-writable ancestor still resolves",
			markers: []string{"shared/.devinit/", "shared/proj/.qsdev.yaml"},
			shared:  []string{"shared"}, unixOnly: true,
			cwd: "shared/proj/sub", want: "shared/proj", wantFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.unixOnly && runtime.GOOS == "windows" {
				t.Skip("marker trust is ACL-based on Windows and out of scope")
			}
			root := testutil.MarkerFreeTempDir(t)
			plant(t, root, tt.markers...)
			for _, d := range tt.shared {
				p := filepath.Join(root, filepath.FromSlash(d))
				if err := os.Chmod(p, 0o1777); err != nil {
					t.Fatal(err)
				}
				t.Cleanup(func() { _ = os.Chmod(p, 0o755) })
			}
			cwd := filepath.Join(root, filepath.FromSlash(tt.cwd))
			if err := os.MkdirAll(cwd, 0o755); err != nil {
				t.Fatal(err)
			}
			home := t.TempDir()
			if tt.home != "" {
				home = filepath.Join(root, filepath.FromSlash(tt.home))
			}
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home) // os.UserHomeDir reads USERPROFILE on Windows
			t.Chdir(cwd)
			want := filepath.Join(root, filepath.FromSlash(tt.want))

			pc, err := Project(&cobra.Command{})
			if err != nil {
				t.Fatalf("Project() error = %v", err)
			}
			if pc.Root != want || pc.Found != tt.wantFound {
				t.Errorf("Project() = {Root: %q, Found: %v}, want {Root: %q, Found: %v}", pc.Root, pc.Found, want, tt.wantFound)
			}
		})
	}
}

// TestProject_StoredContextWins checks that a Context already resolved for
// the command (by the process initializer) is returned as is, without a
// second resolution from the working directory.
func TestProject_StoredContextWins(t *testing.T) {
	t.Parallel()
	want := projectctx.Context{Start: "/s", Root: "/r", Found: true}
	cmd := &cobra.Command{}
	cmd.SetContext(projectctx.WithContext(context.Background(), want))
	got, err := Project(cmd)
	if err != nil {
		t.Fatalf("Project() error = %v", err)
	}
	if got.Root != want.Root || got.Start != want.Start || !got.Found {
		t.Errorf("Project() = %+v, want the stored %+v", got, want)
	}
}

// TestProject_StoredContextIgnoresMode checks that the stored Context wins
// over the command's root mode: the initializer already resolved it with
// that mode.
func TestProject_StoredContextIgnoresMode(t *testing.T) {
	t.Parallel()
	want := projectctx.Context{Start: "/s/sub", Root: "/s", Found: true}
	cmd := MarkRootHere(&cobra.Command{})
	cmd.SetContext(projectctx.WithContext(context.Background(), want))
	got, err := Project(cmd)
	if err != nil {
		t.Fatalf("Project() error = %v", err)
	}
	if got.Root != want.Root {
		t.Errorf("Project().Root = %q, want the stored %q", got.Root, want.Root)
	}
}

// TestProject_LazyFallbackUsesAnnotationMode checks that a command run
// without a stored Context (an addon unit test running a bare command)
// resolves from the working directory with its own root mode: a Here command
// acts on the working directory even below an initialized project.
func TestProject_LazyFallbackUsesAnnotationMode(t *testing.T) {
	root := testutil.MarkerFreeTempDir(t)
	plant(t, root, ".qsdev.yaml", ".devinit/", "services/api/")
	cwd := filepath.Join(root, "services", "api")
	t.Chdir(cwd)

	tests := []struct {
		name string
		cmd  *cobra.Command
		want string
	}{
		{"enclosing by default", &cobra.Command{}, root},
		{"here when marked", MarkRootHere(&cobra.Command{}), cwd},
		{"here inherited from parent", func() *cobra.Command {
			parent := MarkRootHere(&cobra.Command{Use: "init"})
			child := &cobra.Command{Use: "sub"}
			parent.AddCommand(child)
			return child
		}(), cwd},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Project(tt.cmd)
			if err != nil {
				t.Fatalf("Project() error = %v", err)
			}
			if got.Root != tt.want {
				t.Errorf("Project().Root = %q, want %q", got.Root, tt.want)
			}
			// A context without a stored project also falls back.
			tt.cmd.SetContext(context.Background())
			if got, err := Project(tt.cmd); err != nil || got.Root != tt.want {
				t.Errorf("Project() with an empty context = %q, %v; want %q", got.Root, err, tt.want)
			}
		})
	}
}

// TestRootModeInherited pins that RootMode is the nearest ancestor's mark,
// and Enclosing without one.
func TestRootModeInherited(t *testing.T) {
	t.Parallel()
	root := &cobra.Command{Use: "qsdev"}
	initCmd := MarkRootHere(&cobra.Command{Use: "init"})
	initSub := &cobra.Command{Use: "sub"}
	initCmd.AddCommand(initSub)
	status := &cobra.Command{Use: "status"}
	root.AddCommand(initCmd, status)

	tests := []struct {
		cmd  *cobra.Command
		want projectctx.Mode
	}{
		{root, projectctx.Enclosing},
		{initCmd, projectctx.Here},
		{initSub, projectctx.Here},
		{status, projectctx.Enclosing},
	}
	for _, tt := range tests {
		if got := RootMode(tt.cmd); got != tt.want {
			t.Errorf("RootMode(%q) = %v, want %v", tt.cmd.CommandPath(), got, tt.want)
		}
	}
}

// TestJoinedProject pins that JoinedProject returns the stored project when
// it is joined (or not yet initialized) and refuses an un-joined clone.
func TestJoinedProject(t *testing.T) {
	t.Parallel()
	fresh := t.TempDir()
	clone := t.TempDir()
	plant(t, clone, ".qsdev.yaml")

	for _, tt := range []struct {
		name       string
		root       string
		wantJoined bool
	}{
		{"never initialized passes", fresh, true},
		{"un-joined clone refused", clone, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cmd := &cobra.Command{}
			cmd.SetContext(projectctx.WithContext(context.Background(), projectctx.Context{Start: tt.root, Root: tt.root}))
			pc, err := JoinedProject(cmd)
			if tt.wantJoined {
				if err != nil || pc.Root != tt.root {
					t.Errorf("JoinedProject() = %q, %v; want %q, nil", pc.Root, err, tt.root)
				}
				return
			}
			if !errors.Is(err, state.ErrNotJoined) {
				t.Errorf("JoinedProject() error = %v, want ErrNotJoined", err)
			}
		})
	}
}
