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
// into a sandbox. Each HomeDenyRelPaths entry is joined with the current
// user's home directory. If the home directory cannot be determined,
// "/home/unknown" is used as a fallback so that the deny list is never empty.
func HomeDenyPaths() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/home/unknown"
	}

	rels := HomeDenyRelPaths()
	paths := make([]string, 0, len(rels))
	for _, rel := range rels {
		paths = append(paths, filepath.Join(home, filepath.FromSlash(rel)))
	}
	return paths
}

// HomeDenyRelPaths returns the home-relative deny paths, slash-separated,
// without the home directory prefix. It is the single definition of the
// per-user credential stores; HomeDenyPaths expands it.
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
		// Package-registry publish tokens and feed credentials.
		".cargo/credentials.toml",
		".cargo/credentials",
		".nuget/NuGet/NuGet.Config",
		".config/NuGet/NuGet.Config",
		// Container registry, Terraform and Helm credential stores.
		".config/containers/auth.json",
		".terraform.d/credentials.tfrc.json",
		".config/helm/registry",
		".config/helm/repositories.yaml",
	}
}

// AllDenyPaths returns the union of SystemDenyPaths and HomeDenyPaths.
func AllDenyPaths() []string {
	paths := SystemDenyPaths()
	paths = append(paths, HomeDenyPaths()...)
	return paths
}

// ExpandedDenyPaths returns AllDenyPaths together with the symlink-resolved
// form of every entry, without duplicates. Mount validators compare a path's
// CandidatePaths (literal and resolved) against this list: comparing a
// resolved mount path against only the literal deny entries misses every entry
// that sits under a symlinked directory. On macOS /etc, /var and /tmp are
// symlinks into /private, so a link to ~/.ssh under a /var home resolves to
// /private/var/.../.ssh and /private/etc contains /etc/sudoers; without the
// resolved entries both would pass validation.
func ExpandedDenyPaths() []string {
	base := AllDenyPaths()
	out := make([]string, 0, 2*len(base))
	seen := make(map[string]bool, 2*len(base))
	for _, p := range base {
		for _, c := range []string{filepath.Clean(p), resolveExistingPrefix(p)} {
			if !seen[c] {
				seen[c] = true
				out = append(out, c)
			}
		}
	}
	return out
}

// resolveExistingPrefix returns path with symlinks resolved in its deepest
// existing ancestor and the missing remainder re-appended. A deny entry need
// not exist (e.g. ~/.aws on a host without the AWS CLI), yet the directory it
// would live in may still be reached through a symlink, so the entry's
// resolved location must be known either way.
func resolveExistingPrefix(path string) string {
	clean := filepath.Clean(path)
	var rest []string
	for dir := clean; ; dir = filepath.Dir(dir) {
		if resolved, err := filepath.EvalSymlinks(dir); err == nil {
			return filepath.Join(append([]string{resolved}, rest...)...)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return clean
		}
		rest = append([]string{filepath.Base(dir)}, rest...)
	}
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
