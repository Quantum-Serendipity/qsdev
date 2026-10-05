package testutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

func TestMarkedAncestor(t *testing.T) {
	t.Parallel()
	root := MarkerFreeTempDir(t)
	marked := filepath.Join(root, "proj")
	leaf := filepath.Join(marked, "a", "b")
	if err := os.MkdirAll(leaf, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(marked, branding.Get().ConfigFile), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A bare data directory is not a project marker.
	other := filepath.Join(root, "other", "x")
	for _, d := range []string{filepath.Join(root, "."+branding.Get().AppName), other} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	tests := []struct {
		name string
		dir  string
		want string
	}{
		{name: "marker above", dir: leaf, want: marked},
		{name: "marker is the dir itself, not an ancestor", dir: marked, want: ""},
		{name: "bare data dir above is no marker", dir: other, want: ""},
		{name: "no marker", dir: root, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := markedAncestor(tt.dir); got != tt.want {
				t.Errorf("markedAncestor(%q) = %q, want %q", tt.dir, got, tt.want)
			}
		})
	}
}

func TestMarkerFreeTempDir_ReturnsResolvedDir(t *testing.T) {
	t.Parallel()
	got := MarkerFreeTempDir(t)
	info, err := os.Stat(got)
	if err != nil || !info.IsDir() {
		t.Fatalf("MarkerFreeTempDir() = %q, not an existing directory (err %v)", got, err)
	}
	if resolved, err := filepath.EvalSymlinks(got); err != nil || resolved != got {
		t.Errorf("MarkerFreeTempDir() = %q, want its symlink-resolved form %q (err %v)", got, resolved, err)
	}
}
