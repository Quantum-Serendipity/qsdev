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

	candidates := candidatePaths(path)

	if deny, ok := matchedDenyPath(candidates); ok {
		return fmt.Errorf("mount path %q is denied: overlaps sensitive path %q", path, deny)
	}

	// Also reject a path that is an ANCESTOR of a deny entry. Binding such a path
	// (e.g. $HOME, which contains ~/.ssh, or /etc, which contains /etc/shadow)
	// would re-expose the sensitive descendant inside the sandbox, defeating the
	// deny list. Descendants and exact matches are handled by matchedDenyPath.
	if deny, ok := ancestorOfDenyPath(candidates); ok {
		return fmt.Errorf("mount path %q is denied: would re-expose sensitive path %q", path, deny)
	}

	return nil
}

// IsDenyPath reports whether path (or its symlink-resolved form) overlaps a
// sensitive deny-list entry that must never be bind-mounted into a sandbox.
func IsDenyPath(path string) bool {
	_, ok := matchedDenyPath(candidatePaths(path))
	return ok
}

// candidatePaths returns the deny-comparison candidates for path: the cleaned
// literal path plus its symlink-resolved form when that differs, so a symlink
// to (or toward) a sensitive location is caught. Both deny checks consume one
// shared resolution, so they can never disagree on which paths were examined.
func candidatePaths(path string) []string {
	candidates := []string{filepath.Clean(path)}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		if r := filepath.Clean(resolved); r != candidates[0] {
			candidates = append(candidates, r)
		}
	}
	return candidates
}

// matchedDenyPath returns the deny-list entry that any candidate equals or
// descends from, if any.
func matchedDenyPath(candidates []string) (string, bool) {
	denyPaths := denylist.AllDenyPaths()
	for _, clean := range candidates {
		for _, deny := range denyPaths {
			if clean == deny || strings.HasPrefix(clean, deny+"/") {
				return deny, true
			}
		}
	}

	return "", false
}

// ancestorOfDenyPath returns the deny-list entry that any candidate is a strict
// ancestor of, if any.
func ancestorOfDenyPath(candidates []string) (string, bool) {
	denyPaths := denylist.AllDenyPaths()
	for _, clean := range candidates {
		for _, deny := range denyPaths {
			if denylist.IsStrictAncestor(clean, deny) {
				return deny, true
			}
		}
	}

	return "", false
}
