package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/fileutil"
)

// SymlinkOrSkip creates link -> target, making link's parent directories
// first. On Windows, where creating a symlink needs a privilege the test
// process may lack, a failure skips the test; elsewhere it fails the test.
func SymlinkOrSkip(t testing.TB, target, link string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(link), fileutil.ModeDirDefault); err != nil {
		t.Fatalf("creating parent of %s: %v", link, err)
	}
	if err := os.Symlink(target, link); err != nil {
		if runtime.GOOS == "windows" {
			t.Skipf("cannot create symlink %s: %v", link, err)
		}
		t.Fatalf("creating symlink %s -> %s: %v", link, target, err)
	}
}
