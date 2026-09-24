package teardown

import (
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"runtime"
	"strings"
)

// checkTrackedPath rejects a state-recorded path that is not a plain relative
// path inside the project (absolute, empty, or containing ".." components).
// State files can be committed to a repository, so their contents are
// untrusted input.
func checkTrackedPath(relPath string) error {
	if !filepath.IsLocal(filepath.FromSlash(relPath)) {
		return fmt.Errorf("unsafe tracked path %q: must be relative and inside the project", relPath)
	}
	return nil
}

// resolveTrackedPath joins relPath onto projectRoot after checking it, and
// verifies that its parent directory does not resolve (through symlinks)
// outside the project root. A missing parent is not an error: there is then
// nothing on disk to act on.
func resolveTrackedPath(projectRoot, relPath string) (string, error) {
	if err := checkTrackedPath(relPath); err != nil {
		return "", err
	}
	absPath := filepath.Join(projectRoot, filepath.FromSlash(relPath))
	if err := checkWithinRoot(projectRoot, filepath.Dir(absPath)); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return absPath, nil
		}
		return "", fmt.Errorf("%s: %w", relPath, err)
	}
	return absPath, nil
}

// checkWithinRoot resolves symlinks in path and projectRoot and returns an
// error unless the resolved path is the root or lies beneath it.
func checkWithinRoot(projectRoot, path string) error {
	root, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		return fmt.Errorf("resolving project root: %w", err)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return fmt.Errorf("resolving %s: %w", path, err)
	}
	if !isWithin(resolved, root) {
		return fmt.Errorf("path resolves outside the project root: %s", resolved)
	}
	return nil
}

// isWithin reports whether path equals root or lies beneath it, comparing on
// separator boundaries (case-insensitively on Windows).
func isWithin(path, root string) bool {
	if runtime.GOOS == "windows" {
		path, root = strings.ToLower(path), strings.ToLower(root)
	}
	if path == root {
		return true
	}
	return strings.HasPrefix(path, strings.TrimSuffix(root, string(filepath.Separator))+string(filepath.Separator))
}
