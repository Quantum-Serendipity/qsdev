// Package denylist provides shared deny-path lists for sandbox mount validation.
// Both the bwrap backend and the policy compiler import these lists so that
// sensitive paths are defined in exactly one place.
package denylist

import (
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/pathmatch"
	"github.com/Quantum-Serendipity/qsdev/internal/projectctx"
	"github.com/Quantum-Serendipity/qsdev/internal/secrets"
)

// matchOptions is how Covering and Conflict compare names: the host
// filesystem's (a variable so tests can drive the macOS and Windows forms on
// any OS).
var matchOptions = pathmatch.Platform

// evalSymlinks resolves symlinks for resolveExistingPrefix (a variable so
// tests can count the filesystem lookups a check costs).
var evalSymlinks = filepath.EvalSymlinks

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

// HomeDenyPaths returns the per-user paths that must never be bind-mounted
// into a sandbox: each credential store in secrets.CredentialPaths joined with
// the current user's home directory. If the home directory cannot be
// determined, "/home/unknown" is used as a fallback so that the deny list is
// never empty.
func HomeDenyPaths() []string {
	home, err := projectctx.HomeDir()
	if err != nil {
		home = "/home/unknown"
	}

	rels := secrets.CredentialPaths()
	paths := make([]string, 0, len(rels))
	for _, rel := range rels {
		paths = append(paths, filepath.Join(home, filepath.FromSlash(rel)))
	}
	return paths
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
		if resolved, err := evalSymlinks(dir); err == nil {
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
// to (or toward) a sensitive location is caught. Resolution goes through the
// deepest existing ancestor, exactly as ExpandedDenyPaths does for deny
// entries: a mount path that does not exist yet (e.g. ~/.aws) under a
// symlinked directory (macOS /var -> /private/var) must still resolve to the
// same form as the deny entry. Both mount validators build their candidates
// here, so they can never disagree on which paths were examined.
func CandidatePaths(path string) []string {
	candidates := []string{filepath.Clean(path)}
	if r := resolveExistingPrefix(path); r != candidates[0] {
		candidates = append(candidates, r)
	}
	return candidates
}

// Covering returns the ExpandedDenyPaths entry that path (any CandidatePaths
// form of it) equals or lies under, as the host filesystem compares names: on
// macOS and Windows ~/.SSH/id_rsa is covered by ~/.ssh.
func Covering(path string) (deny string, ok bool) {
	return firstMatch(CandidatePaths(path), ExpandedDenyPaths(), covers)
}

// Conflict reports the ExpandedDenyPaths entry that path conflicts with, if
// any: first one that a CandidatePaths form of path equals or lies under (as
// Covering, exposes=false), otherwise one that a form of path is a strict
// ancestor of (exposes=true). Binding such an ancestor (e.g. $HOME, which
// contains ~/.ssh, or /etc, which contains /etc/shadow) would re-expose the
// sensitive descendant inside the sandbox; the filesystem root "/" exposes
// every entry. The deny list and the candidates are resolved once and shared
// by both checks, since each resolution costs filesystem lookups.
func Conflict(path string) (deny string, exposes, ok bool) {
	denyPaths := ExpandedDenyPaths()
	candidates := CandidatePaths(path)
	if deny, ok := firstMatch(candidates, denyPaths, covers); ok {
		return deny, false, true
	}
	deny, ok = firstMatch(candidates, denyPaths, exposesEntry)
	return deny, ok, ok
}

// covers reports whether candidate equals deny or lies under it.
func covers(candidate, deny string) bool {
	return matchOptions.Within(candidate, deny)
}

// exposesEntry reports whether candidate is a strict ancestor of deny.
func exposesEntry(candidate, deny string) bool {
	return matchOptions.StrictlyWithin(deny, candidate)
}

// firstMatch returns the first deny entry that match reports for any of the
// candidates, trying the candidates in order.
func firstMatch(candidates, denyPaths []string, match func(candidate, deny string) bool) (string, bool) {
	for _, candidate := range candidates {
		for _, deny := range denyPaths {
			if match(candidate, deny) {
				return deny, true
			}
		}
	}
	return "", false
}
