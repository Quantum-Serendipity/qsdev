package bwrap

import (
	"fmt"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/sandbox/denylist"
)

// ValidateMountPath checks that a path is safe to use as a bind-mount source
// or target inside a sandbox. It rejects relative paths and paths on the deny
// list (checking both the literal path and its symlink-resolved form).
func ValidateMountPath(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("mount path must be absolute: %q", path)
	}

	candidates := denylist.CandidatePaths(path)
	denyPaths := denylist.AllDenyPaths()

	if deny, ok := matchedDenyPath(candidates, denyPaths); ok {
		return fmt.Errorf("mount path %q is denied: overlaps sensitive path %q", path, deny)
	}

	// Also reject a path that is an ANCESTOR of a deny entry. Binding such a path
	// (e.g. $HOME, which contains ~/.ssh, or /etc, which contains /etc/shadow)
	// would re-expose the sensitive descendant inside the sandbox, defeating the
	// deny list. Descendants and exact matches are handled by matchedDenyPath.
	if deny, ok := ancestorOfDenyPath(candidates, denyPaths); ok {
		return fmt.Errorf("mount path %q is denied: would re-expose sensitive path %q", path, deny)
	}

	return nil
}

// IsDenyPath reports whether path (or its symlink-resolved form) overlaps a
// sensitive deny-list entry that must never be bind-mounted into a sandbox.
func IsDenyPath(path string) bool {
	_, ok := matchedDenyPath(denylist.CandidatePaths(path), denylist.AllDenyPaths())
	return ok
}

// matchedDenyPath returns the deny-list entry that any candidate equals or
// descends from, if any.
func matchedDenyPath(candidates, denyPaths []string) (string, bool) {
	for _, clean := range candidates {
		for _, deny := range denyPaths {
			if denylist.Overlaps(clean, deny) {
				return deny, true
			}
		}
	}

	return "", false
}

// ancestorOfDenyPath returns the deny-list entry that any candidate is a strict
// ancestor of, if any.
func ancestorOfDenyPath(candidates, denyPaths []string) (string, bool) {
	for _, clean := range candidates {
		for _, deny := range denyPaths {
			if denylist.IsStrictAncestor(clean, deny) {
				return deny, true
			}
		}
	}

	return "", false
}
