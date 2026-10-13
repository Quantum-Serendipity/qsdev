package bwrap

import (
	"fmt"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/denylist"
)

// ValidateMountPath checks that a path is safe to use as a bind-mount source
// or target inside a sandbox. It rejects relative paths and paths that
// conflict with the deny list (checking both the literal path and its
// symlink-resolved form).
func ValidateMountPath(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("mount path must be absolute: %q", path)
	}

	switch deny, exposes, ok := denylist.Conflict(path); {
	case ok && exposes:
		// Binding an ancestor of a deny entry (e.g. $HOME, which contains
		// ~/.ssh, or /etc, which contains /etc/shadow) would re-expose the
		// sensitive descendant inside the sandbox, defeating the deny list.
		return fmt.Errorf("mount path %q is denied: would re-expose sensitive path %q", path, deny)
	case ok:
		return fmt.Errorf("mount path %q is denied: overlaps sensitive path %q", path, deny)
	}

	return nil
}

// IsDenyPath reports whether path (or its symlink-resolved form) overlaps a
// sensitive deny-list entry that must never be bind-mounted into a sandbox.
func IsDenyPath(path string) bool {
	_, ok := denylist.Covering(path)
	return ok
}
