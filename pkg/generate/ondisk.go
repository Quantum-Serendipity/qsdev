package generate

import (
	"bytes"
	"os"
	"runtime"

	"github.com/Quantum-Serendipity/qsdev/internal/state"
)

// compareOnDisk inspects the existing regular file at path. It returns the
// content hash of the existing bytes (empty when the file cannot be read) and
// whether the file already holds exactly content with permission bits mode,
// in which case rewriting it would change nothing but its mtime.
//
// Windows does not model Unix permission bits, so only content is compared
// there.
func compareOnDisk(path string, content []byte, mode os.FileMode) (prevHash string, identical bool) {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return "", false
	}
	existing, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	sameMode := runtime.GOOS == "windows" || info.Mode().Perm() == mode.Perm()
	return state.ComputeHash(existing), sameMode && bytes.Equal(existing, content)
}
