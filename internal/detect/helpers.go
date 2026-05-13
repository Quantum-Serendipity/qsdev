package detect

import (
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/fileutil"
)

// fileExists returns true if the path formed by joining parts exists and is a regular file.
func fileExists(parts ...string) bool {
	return fileutil.FileExists(parts...)
}

// dirExists returns true if the path formed by joining parts exists and is a directory.
func dirExists(parts ...string) bool {
	return fileutil.DirExists(parts...)
}

// readFirstLine reads the first non-empty line from the file at path and
// returns it trimmed. Returns empty string on any error.
func readFirstLine(path string) string {
	return fileutil.ReadFirstLine(path)
}
