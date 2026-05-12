package detect

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// fileExists returns true if the path formed by joining parts exists and is a regular file.
func fileExists(parts ...string) bool {
	info, err := os.Stat(filepath.Join(parts...))
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// dirExists returns true if the path formed by joining parts exists and is a directory.
func dirExists(parts ...string) bool {
	info, err := os.Stat(filepath.Join(parts...))
	if err != nil {
		return false
	}
	return info.IsDir()
}

// readFirstLine reads the first non-empty line from the file at path and
// returns it trimmed. Returns empty string on any error.
func readFirstLine(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close() //nolint:errcheck // best-effort read

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			return line
		}
	}
	return ""
}
