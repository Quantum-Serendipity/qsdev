package testutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMarkedAncestor(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	marked := filepath.Join(root, "proj")
	leaf := filepath.Join(marked, "a", "b")
	if err := os.MkdirAll(leaf, 0o755); err != nil {
		t.Fatal(err)
	}
	isMarker := func(dir string) bool { return dir == marked }

	tests := []struct {
		name string
		dir  string
		want string
	}{
		{name: "marker above", dir: leaf, want: marked},
		{name: "marker is the dir itself, not an ancestor", dir: marked, want: ""},
		{name: "no marker", dir: root, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := markedAncestor(tt.dir, isMarker); got != tt.want {
				t.Errorf("markedAncestor(%q) = %q, want %q", tt.dir, got, tt.want)
			}
		})
	}
}

func TestMarkerFreeTempDir_ReturnsResolvedDir(t *testing.T) {
	t.Parallel()
	got := MarkerFreeTempDir(t, func(string) bool { return false })
	info, err := os.Stat(got)
	if err != nil || !info.IsDir() {
		t.Fatalf("MarkerFreeTempDir() = %q, not an existing directory (err %v)", got, err)
	}
	if resolved, err := filepath.EvalSymlinks(got); err != nil || resolved != got {
		t.Errorf("MarkerFreeTempDir() = %q, want its symlink-resolved form %q (err %v)", got, resolved, err)
	}
}
