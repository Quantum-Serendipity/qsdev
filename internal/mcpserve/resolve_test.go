package mcpserve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/internal/testutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// fakeEnv builds a getenv function from a fixed map.
func fakeEnv(vals map[string]string) func(string) string {
	return func(k string) string { return vals[k] }
}

func TestPickStartDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		flagRoot string
		env      map[string]string
		wd       string
		want     string
	}{
		{
			name:     "flag wins over everything",
			flagRoot: "/flag",
			env:      map[string]string{envProjectRoot: "/qsdev", envGdevProjectRoot: "/gdev"},
			wd:       "/wd",
			want:     "/flag",
		},
		{
			name: "qsdev env beats gdev env",
			env:  map[string]string{envProjectRoot: "/qsdev", envGdevProjectRoot: "/gdev"},
			wd:   "/wd",
			want: "/qsdev",
		},
		{
			name: "gdev env used when qsdev unset",
			env:  map[string]string{envGdevProjectRoot: "/gdev"},
			wd:   "/wd",
			want: "/gdev",
		},
		{
			name: "getwd is the final fallback",
			env:  map[string]string{},
			wd:   "/wd",
			want: "/wd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			getwd := func() (string, error) { return tt.wd, nil }
			got, err := pickStartDir(tt.flagRoot, fakeEnv(tt.env), getwd)
			if err != nil {
				t.Fatalf("pickStartDir returned error: %v", err)
			}
			if got != tt.want {
				t.Errorf("pickStartDir = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveProjectRoot(t *testing.T) {
	t.Parallel()
	b := branding.Get()

	// mkProject returns a fresh repository (a .git entry bounds every walk
	// inside it) with the slash-separated paths planted below it: a trailing
	// "/" makes a directory, anything else a regular file.
	mkProject := func(t *testing.T, paths ...string) string {
		t.Helper()
		root := t.TempDir()
		for _, p := range append([]string{".git/"}, paths...) {
			full := filepath.Join(root, filepath.FromSlash(p))
			if strings.HasSuffix(p, "/") {
				if err := os.MkdirAll(full, 0o755); err != nil {
					t.Fatal(err)
				}
				continue
			}
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatal(err)
			}
			writeFile(t, full, "version: 1\n")
		}
		return root
	}

	tests := []struct {
		name  string
		paths []string
		start string // slash-separated, relative to the repository
		want  string // relative to the repository
	}{
		{name: "walks up to the config file", paths: []string{b.ConfigFile, "sub/dir/"}, start: "sub/dir", want: "."},
		{name: "state dir is a marker", paths: []string{b.StateDir + "/", "sub/"}, start: "sub", want: "."},
		{name: "nearest marker below the toplevel wins", paths: []string{"svc/" + b.ConfigFile, "svc/api/"}, start: "svc/api", want: "svc"},
		{name: "no marker resolves to the start, not the git toplevel", paths: []string{"pkg/a/"}, start: "pkg/a", want: "pkg/a"},
		{name: "go.mod is not a marker", paths: []string{"mod/go.mod", "mod/pkg/"}, start: "mod/pkg", want: "mod/pkg"},
		{name: "package.json is not a marker", paths: []string{"web/package.json", "web/src/"}, start: "web/src", want: "web/src"},
		{name: "bare data dir is not a marker", paths: []string{"sub/." + b.AppName + "/", "sub/x/"}, start: "sub/x", want: "sub/x"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root := mkProject(t, tt.paths...)
			start := filepath.Join(root, filepath.FromSlash(tt.start))
			got, err := ResolveProjectRoot(ResolveOptions{
				Getenv: fakeEnv(nil),
				Getwd:  func() (string, error) { return start, nil },
			})
			if err != nil {
				t.Fatalf("ResolveProjectRoot: %v", err)
			}
			if want := filepath.Join(root, filepath.FromSlash(tt.want)); got != want {
				t.Errorf("ResolveProjectRoot = %q, want %q", got, want)
			}
		})
	}

	t.Run("no walk past the git toplevel", func(t *testing.T) {
		t.Parallel()
		outer := t.TempDir()
		writeFile(t, filepath.Join(outer, b.ConfigFile), "version: 1\n")
		child := filepath.Join(outer, "child")
		start := filepath.Join(child, "src")
		for _, d := range []string{filepath.Join(child, ".git"), start} {
			if err := os.MkdirAll(d, 0o755); err != nil {
				t.Fatal(err)
			}
		}
		got, err := ResolveProjectRoot(ResolveOptions{FlagRoot: start, Getenv: fakeEnv(nil)})
		if err != nil {
			t.Fatalf("ResolveProjectRoot: %v", err)
		}
		if got != start {
			t.Errorf("ResolveProjectRoot = %q, want the start %q, not the marked ancestor above the git toplevel", got, start)
		}
	})

	t.Run("flag override wins and walks up from it", func(t *testing.T) {
		t.Parallel()
		flagRoot := mkProject(t, b.ConfigFile, "deep/")
		otherRoot := mkProject(t, b.ConfigFile)

		got, err := ResolveProjectRoot(ResolveOptions{
			FlagRoot: filepath.Join(flagRoot, "deep"),
			Getenv:   fakeEnv(map[string]string{envProjectRoot: otherRoot}),
			Getwd:    func() (string, error) { return otherRoot, nil },
		})
		if err != nil {
			t.Fatalf("ResolveProjectRoot: %v", err)
		}
		if got != flagRoot {
			t.Errorf("ResolveProjectRoot = %q, want flag-derived %q", got, flagRoot)
		}
	})

	t.Run("env var selects start dir", func(t *testing.T) {
		t.Parallel()
		envRoot := mkProject(t, b.ConfigFile)

		got, err := ResolveProjectRoot(ResolveOptions{
			Getenv: fakeEnv(map[string]string{envProjectRoot: envRoot}),
			Getwd:  func() (string, error) { return t.TempDir(), nil },
		})
		if err != nil {
			t.Fatalf("ResolveProjectRoot: %v", err)
		}
		if got != envRoot {
			t.Errorf("ResolveProjectRoot = %q, want env %q", got, envRoot)
		}
	})

	t.Run("relative flag is made absolute", func(t *testing.T) {
		t.Parallel()
		got, err := ResolveProjectRoot(ResolveOptions{FlagRoot: ".", Getenv: fakeEnv(nil)})
		if err != nil {
			t.Fatalf("ResolveProjectRoot: %v", err)
		}
		if !filepath.IsAbs(got) {
			t.Errorf("ResolveProjectRoot(\".\") = %q, want an absolute path", got)
		}
	})

	t.Run("missing start directory errors", func(t *testing.T) {
		t.Parallel()
		missing := filepath.Join(t.TempDir(), "absent")
		if got, err := ResolveProjectRoot(ResolveOptions{FlagRoot: missing, Getenv: fakeEnv(nil)}); err == nil {
			t.Errorf("ResolveProjectRoot(%q) = %q, want an error", missing, got)
		}
	})
}

// TestResolveProjectRoot_NoMarkerNoRepo covers a start outside any project
// and with no repository of its own (the walk ends at the isolation ceiling
// above it): the absolute start directory is the root.
func TestResolveProjectRoot_NoMarkerNoRepo(t *testing.T) {
	t.Parallel()
	start := filepath.Join(testutil.IsolatedDir(t), "loose")
	if err := os.MkdirAll(start, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveProjectRoot(ResolveOptions{FlagRoot: start, Getenv: fakeEnv(nil)})
	if err != nil {
		t.Fatalf("ResolveProjectRoot: %v", err)
	}
	if got != start {
		t.Errorf("ResolveProjectRoot = %q, want the start directory %q", got, start)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
