package fileutil

import (
	"fmt"
	"path/filepath"
)

// ResolvePath returns path made absolute, with symlinks resolved in its
// longest existing prefix and the components that do not exist yet
// re-appended, so two spellings of one location compare equal whether or not
// the file exists.
func ResolvePath(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", path, err)
	}
	return resolveExistingPrefix(abs)
}

// PathWithin reports whether path is root or lies beneath it once both are
// resolved (see ResolvePath). It is false when either cannot be resolved.
func PathWithin(root, path string) bool {
	r, err := ResolvePath(root)
	if err != nil {
		return false
	}
	p, err := ResolvePath(path)
	if err != nil {
		return false
	}
	return isWithin(r, p)
}
