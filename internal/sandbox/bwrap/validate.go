package bwrap

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/denylist"
)

// ValidateMountPath checks that a path is safe to use as a bind-mount source
// or target inside a sandbox. It rejects relative paths and paths on the deny
// list (checking both the literal path and its symlink-resolved form).
func ValidateMountPath(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("mount path must be absolute: %q", path)
	}

	if deny, ok := matchedDenyPath(path); ok {
		return fmt.Errorf("mount path %q is denied: overlaps sensitive path %q", path, deny)
	}

	// Also reject a path that is an ANCESTOR of a deny entry. Binding such a path
	// (e.g. $HOME, which contains ~/.ssh, or /etc, which contains /etc/shadow)
	// would re-expose the sensitive descendant inside the sandbox, defeating the
	// deny list. Descendants and exact matches are handled by matchedDenyPath.
	if deny, ok := ancestorOfDenyPath(path); ok {
		return fmt.Errorf("mount path %q is denied: would re-expose sensitive path %q", path, deny)
	}

	return nil
}

// IsDenyPath reports whether path (or its symlink-resolved form) overlaps a
// sensitive deny-list entry that must never be bind-mounted into a sandbox.
func IsDenyPath(path string) bool {
	_, ok := matchedDenyPath(path)
	return ok
}

// matchedDenyPath returns the deny-list entry that path overlaps, if any. It
// checks both the cleaned literal path and its symlink-resolved form so that a
// symlink to a sensitive location is caught.
func matchedDenyPath(path string) (string, bool) {
	candidates := []string{filepath.Clean(path)}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		if r := filepath.Clean(resolved); r != candidates[0] {
			candidates = append(candidates, r)
		}
	}

	for _, clean := range candidates {
		for _, deny := range denylist.AllDenyPaths() {
			if clean == deny || strings.HasPrefix(clean, deny+"/") {
				return deny, true
			}
		}
	}

	return "", false
}

// ancestorOfDenyPath returns the deny-list entry that path is a strict ancestor
// of, if any. It checks both the cleaned literal path and its symlink-resolved
// form so that a symlink to an ancestor of a sensitive location is caught.
func ancestorOfDenyPath(path string) (string, bool) {
	candidates := []string{filepath.Clean(path)}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		if r := filepath.Clean(resolved); r != candidates[0] {
			candidates = append(candidates, r)
		}
	}

	for _, clean := range candidates {
		for _, deny := range denylist.AllDenyPaths() {
			if isStrictAncestor(clean, deny) {
				return deny, true
			}
		}
	}

	return "", false
}

// isStrictAncestor reports whether ancestor is a proper parent directory of
// descendant (not equal to it). The filesystem root "/" is an ancestor of every
// absolute path.
func isStrictAncestor(ancestor, descendant string) bool {
	if ancestor == descendant {
		return false
	}
	if ancestor == "/" {
		return true
	}
	return strings.HasPrefix(descendant, ancestor+"/")
}
