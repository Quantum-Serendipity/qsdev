package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestSymlinkOrSkip(t *testing.T) {
	t.Parallel()

	t.Run("creates parents and link", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		link := filepath.Join(dir, "a", "b", "link")
		r := record(func(tb testing.TB) { SymlinkOrSkip(tb, "target", link) })
		if r.failed {
			t.Fatalf("SymlinkOrSkip failed: %s", r.msg)
		}
		if r.skipped {
			t.Skipf("symlinks unavailable: %s", r.msg)
		}
		got, err := os.Readlink(link)
		if err != nil {
			t.Fatalf("reading link: %v", err)
		}
		if got != "target" {
			t.Errorf("link target = %q, want %q", got, "target")
		}
	})

	t.Run("creation failure", func(t *testing.T) {
		t.Parallel()
		link := filepath.Join(t.TempDir(), "taken")
		if err := os.WriteFile(link, nil, 0o644); err != nil {
			t.Fatal(err)
		}
		r := record(func(tb testing.TB) { SymlinkOrSkip(tb, "target", link) })
		if runtime.GOOS == "windows" {
			if !r.skipped || r.failed {
				t.Errorf("skipped=%v failed=%v, want a skip on Windows", r.skipped, r.failed)
			}
			return
		}
		if !r.failed || r.skipped {
			t.Errorf("skipped=%v failed=%v, want a failure", r.skipped, r.failed)
		}
	})
}
