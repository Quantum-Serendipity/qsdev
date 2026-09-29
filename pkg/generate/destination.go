package generate

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// ValidateDestination checks that writing (or removing) relPath under
// projectRoot stays inside the project. relPath must be relative and free of
// ".." components, and neither an existing file at the path nor its nearest
// existing ancestor directory may resolve through symlinks to a location
// outside the project root. A dangling symlink on the path is refused too: its
// target cannot be checked, and a non-atomic write through it would create
// the target wherever it points. Write paths that bypass WriteFiles (enable,
// disable) use it to get the same containment guarantee.
func ValidateDestination(projectRoot, relPath string) error {
	if filepath.IsAbs(relPath) {
		return fmt.Errorf("file path must be relative: %q", relPath)
	}
	if containsPathTraversal(relPath) {
		return fmt.Errorf("file path contains path traversal: %q", relPath)
	}
	root, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		return fmt.Errorf("resolving project root symlinks: %w", err)
	}

	full := filepath.Join(projectRoot, relPath)
	if resolved, err := filepath.EvalSymlinks(full); err == nil {
		if !pathHasPrefix(resolved, root) {
			return fmt.Errorf("resolved path escapes project root: %q", relPath)
		}
		return nil
	}

	// The target does not exist (yet): check the directory it would be
	// created in, walking up to the nearest ancestor that exists.
	if err := refuseDanglingSymlink(full, relPath); err != nil {
		return err
	}
	for dir := filepath.Dir(full); ; {
		resolved, err := filepath.EvalSymlinks(dir)
		if err == nil {
			if !pathHasPrefix(resolved, root) {
				return fmt.Errorf("resolved path escapes project root: %q", relPath)
			}
			return nil
		}
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("resolving %s: %w", relPath, err)
		}
		if err := refuseDanglingSymlink(dir, relPath); err != nil {
			return err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return nil
		}
		dir = parent
	}
}

// refuseDanglingSymlink reports an error when p exists as a directory entry
// even though resolving it failed with "not exist", i.e. p is (or passes
// through) a symlink whose target is missing.
func refuseDanglingSymlink(p, relPath string) error {
	if _, err := os.Lstat(p); err == nil {
		return fmt.Errorf("path passes through a dangling symlink: %q", relPath)
	}
	return nil
}
