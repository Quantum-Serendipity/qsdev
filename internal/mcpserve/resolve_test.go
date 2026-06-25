package mcpserve

import (
	"os"
	"path/filepath"
	"testing"
)

// fakeEnv builds a getenv function from a fixed map.
func fakeEnv(vals map[string]string) func(string) string {
	return func(k string) string { return vals[k] }
}

func TestPickStartDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		flagRoot  string
		rootsDirs []string
		env       map[string]string
		wd        string
		want      string
	}{
		{
			name:      "flag wins over everything",
			flagRoot:  "/flag",
			rootsDirs: []string{"/roots"},
			env:       map[string]string{envProjectRoot: "/qsdev", envGdevProjectRoot: "/gdev"},
			wd:        "/wd",
			want:      "/flag",
		},
		{
			name:      "roots used when no flag",
			rootsDirs: []string{"", "/roots"},
			env:       map[string]string{envProjectRoot: "/qsdev"},
			wd:        "/wd",
			want:      "/roots",
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
			got, err := pickStartDir(tt.flagRoot, tt.rootsDirs, fakeEnv(tt.env), getwd)
			if err != nil {
				t.Fatalf("pickStartDir returned error: %v", err)
			}
			if got != tt.want {
				t.Errorf("pickStartDir = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWalkUpForFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	nested := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	writeFile(t, filepath.Join(root, ".qsdev.yaml"), "qsdev_version: 1\n")

	t.Run("found in ancestor", func(t *testing.T) {
		t.Parallel()
		got, ok := walkUpForFile(nested, ".qsdev.yaml")
		if !ok {
			t.Fatalf("walkUpForFile did not find marker")
		}
		if got != root {
			t.Errorf("walkUpForFile = %q, want %q", got, root)
		}
	})

	t.Run("found in start dir", func(t *testing.T) {
		t.Parallel()
		got, ok := walkUpForFile(root, ".qsdev.yaml")
		if !ok || got != root {
			t.Errorf("walkUpForFile = (%q,%v), want (%q,true)", got, ok, root)
		}
	})

	t.Run("not found", func(t *testing.T) {
		t.Parallel()
		_, ok := walkUpForFile(nested, "definitely-not-present-9f3c.marker")
		if ok {
			t.Errorf("walkUpForFile unexpectedly reported a marker found")
		}
	})
}

func TestWalkUpForAny(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	nested := filepath.Join(root, "x", "y")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatalf("mkdir .git: %v", err)
	}

	got, ok := walkUpForAny(nested, fallbackMarkers)
	if !ok {
		t.Fatalf("walkUpForAny did not find a fallback marker")
	}
	if got != root {
		t.Errorf("walkUpForAny = %q, want %q", got, root)
	}

	if _, ok := walkUpForAny(nested, []string{"no-such-marker-zzz"}); ok {
		t.Errorf("walkUpForAny unexpectedly found a marker")
	}
}

func TestResolveProjectRoot(t *testing.T) {
	t.Parallel()

	t.Run("walks up to .qsdev.yaml from working dir", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		nested := filepath.Join(root, "sub", "dir")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		writeFile(t, filepath.Join(root, ".qsdev.yaml"), "qsdev_version: 1\n")

		got, err := ResolveProjectRoot(ResolveOptions{
			Getenv: fakeEnv(nil),
			Getwd:  func() (string, error) { return nested, nil },
		})
		if err != nil {
			t.Fatalf("ResolveProjectRoot: %v", err)
		}
		if got != root {
			t.Errorf("ResolveProjectRoot = %q, want %q", got, root)
		}
	})

	t.Run("flag override wins and walks up from it", func(t *testing.T) {
		t.Parallel()
		flagRoot := t.TempDir()
		otherRoot := t.TempDir()
		writeFile(t, filepath.Join(flagRoot, ".qsdev.yaml"), "qsdev_version: 1\n")
		writeFile(t, filepath.Join(otherRoot, ".qsdev.yaml"), "qsdev_version: 1\n")

		got, err := ResolveProjectRoot(ResolveOptions{
			FlagRoot:  filepath.Join(flagRoot, "deep"), // does not exist; walk-up still reaches flagRoot
			RootsDirs: []string{otherRoot},
			Getenv:    fakeEnv(map[string]string{envProjectRoot: otherRoot}),
			Getwd:     func() (string, error) { return otherRoot, nil },
		})
		if err != nil {
			t.Fatalf("ResolveProjectRoot: %v", err)
		}
		if got != flagRoot {
			t.Errorf("ResolveProjectRoot = %q, want flag-derived %q", got, flagRoot)
		}
	})

	t.Run("falls back to weaker marker when no qsdev config", func(t *testing.T) {
		t.Parallel()
		root := t.TempDir()
		nested := filepath.Join(root, "pkg")
		if err := os.MkdirAll(nested, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		writeFile(t, filepath.Join(root, "go.mod"), "module example.com/x\n")

		got, err := ResolveProjectRoot(ResolveOptions{
			Getenv: fakeEnv(nil),
			Getwd:  func() (string, error) { return nested, nil },
		})
		if err != nil {
			t.Fatalf("ResolveProjectRoot: %v", err)
		}
		if got != root {
			t.Errorf("ResolveProjectRoot = %q, want fallback %q", got, root)
		}
	})

	t.Run("env var selects start dir", func(t *testing.T) {
		t.Parallel()
		envRoot := t.TempDir()
		writeFile(t, filepath.Join(envRoot, ".qsdev.yaml"), "qsdev_version: 1\n")

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
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}
