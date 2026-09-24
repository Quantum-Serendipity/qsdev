package fileutil

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileExists reports whether a regular file exists at the joined path components.
func FileExists(parts ...string) bool {
	info, err := os.Stat(filepath.Join(parts...))
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DirExists reports whether a directory exists at the joined path components.
func DirExists(parts ...string) bool {
	info, err := os.Stat(filepath.Join(parts...))
	if err != nil {
		return false
	}
	return info.IsDir()
}

// ReadFirstLine reads and returns the first non-empty trimmed line from a file.
// Returns an empty string if the file cannot be read or contains no non-empty
// lines; use ReadFirstLineErr to distinguish those cases.
func ReadFirstLine(path string) string {
	line, _ := ReadFirstLineErr(path)
	return line
}

// ReadFirstLineErr returns the first non-empty trimmed line from a file, like
// ReadFirstLine, but reports open and scan errors (such as a line longer than
// the scanner's buffer) instead of returning an empty string.
func ReadFirstLineErr(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("opening %s: %w", path, err)
	}
	defer f.Close() //nolint:errcheck

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if line := strings.TrimSpace(scanner.Text()); line != "" {
			return line, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("reading %s: %w", path, err)
	}
	return "", nil
}
