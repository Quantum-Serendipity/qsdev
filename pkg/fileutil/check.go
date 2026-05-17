package fileutil

import (
	"bufio"
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
// Returns an empty string if the file cannot be read or contains no non-empty lines.
func ReadFirstLine(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close() //nolint:errcheck

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			return line
		}
	}
	return ""
}

// ReadFirstLineErr reads the first line of a file, returning any error encountered.
// Unlike ReadFirstLine, this does not silently swallow errors.
func ReadFirstLineErr(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close() //nolint:errcheck

	scanner := bufio.NewScanner(f)
	if scanner.Scan() {
		return scanner.Text(), nil
	}
	if err := scanner.Err(); err != nil {
		return "", err
	}
	return "", nil
}
