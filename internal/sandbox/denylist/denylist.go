// Package denylist provides shared deny-path lists for sandbox mount validation.
// Both the bwrap backend and the policy compiler import these lists so that
// sensitive paths are defined in exactly one place.
package denylist

import (
	"os"
	"path/filepath"
	"strings"
)

// SystemDenyPaths returns absolute paths that must never be bind-mounted into
// a sandbox. These cover system credential stores and privilege-escalation
// vectors.
func SystemDenyPaths() []string {
	return []string{
		"/etc/shadow",
		"/etc/sudoers",
		"/etc/sudoers.d",
		"/root",
	}
}

// HomeDenyPaths returns home-relative paths that must never be bind-mounted
// into a sandbox. Each entry is joined with the current user's home directory.
// If the home directory cannot be determined, "/home/unknown" is used as a
// fallback so that the deny list is never empty.
func HomeDenyPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/home/unknown"
	}

	return []string{
		filepath.Join(home, ".ssh"),
		filepath.Join(home, ".gnupg"),
		filepath.Join(home, ".aws"),
		filepath.Join(home, ".azure"),
		filepath.Join(home, ".config", "gcloud"),
		filepath.Join(home, ".kube"),
		filepath.Join(home, ".docker", "config.json"),
		filepath.Join(home, ".netrc"),
	}
}

// HomeDenyRelPaths returns the home-relative deny paths without the home
// directory prefix. Callers that already have the home directory can use this
// to avoid redundant os.UserHomeDir calls.
func HomeDenyRelPaths() []string {
	return []string{
		".ssh",
		".gnupg",
		".aws",
		".azure",
		".config/gcloud",
		".kube",
		".docker/config.json",
		".netrc",
	}
}

// AllDenyPaths returns the union of SystemDenyPaths and HomeDenyPaths.
func AllDenyPaths() []string {
	paths := SystemDenyPaths()
	paths = append(paths, HomeDenyPaths()...)
	return paths
}

// CandidatePaths returns the deny-comparison candidates for path: the cleaned
// literal path plus its symlink-resolved form when that differs, so a symlink
// to (or toward) a sensitive location is caught. Both mount validators build
// their candidates here, so they can never disagree on which paths were
// examined.
func CandidatePaths(path string) []string {
	candidates := []string{filepath.Clean(path)}
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		if r := filepath.Clean(resolved); r != candidates[0] {
			candidates = append(candidates, r)
		}
	}
	return candidates
}

// Overlaps reports whether path equals deny or is a descendant of it. It is
// the complement of IsStrictAncestor: together they cover every way a mount
// path can conflict with a deny entry.
func Overlaps(path, deny string) bool {
	return path == deny || strings.HasPrefix(path, deny+"/")
}

// IsStrictAncestor reports whether ancestor is a proper parent directory of
// descendant (not equal to it). The filesystem root "/" is an ancestor of every
// absolute path. Both mount validators use it to reject binding an ancestor of
// a deny path (e.g. $HOME, which contains ~/.ssh), which would re-expose the
// sensitive descendant inside the sandbox.
func IsStrictAncestor(ancestor, descendant string) bool {
	if ancestor == descendant {
		return false
	}
	if ancestor == "/" {
		return true
	}
	return strings.HasPrefix(descendant, ancestor+"/")
}
