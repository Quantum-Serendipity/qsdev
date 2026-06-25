package workspace

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

// GlobResolver resolves ecosystem workspace patterns into concrete package
// directories. It wraps github.com/bmatcuk/doublestar/v4, which uniformly
// understands "**" recursion and "{a,b}" brace expansion, and adds the
// ecosystem-specific normalizations the workspace formats require:
//
//   - pnpm/npm "!"-prefixed include entries are treated as exclude patterns.
//   - Cargo's prefix-match exclusion (an entry names a directory whose whole
//     subtree is excluded) is honored alongside true glob excludes.
//   - "**" recursion and "{packages,libs}/*" brace expansion are passed through
//     to doublestar.
//
// All patterns are interpreted relative to the workspace root.
type GlobResolver struct{}

// NewGlobResolver constructs a GlobResolver. It is stateless and safe to share.
func NewGlobResolver() *GlobResolver { return &GlobResolver{} }

// ResolvePatterns expands includes against the directory tree rooted at root and
// removes any directory matched by excludes, returning the surviving package
// directories as cleaned, forward-slash relative paths sorted for determinism.
//
// Only directories are returned: workspace members are always directories, so a
// pattern that also matches files (e.g. "packages/*" matching a stray
// "packages/README.md") contributes only its directory matches.
//
// An include entry beginning with "!" is reinterpreted as an exclude (defensive
// handling of pnpm/npm negation even when a parser has not already split it).
func (r *GlobResolver) ResolvePatterns(root string, includes, excludes []string) ([]string, error) {
	includePats, negatedExcludes := splitNegations(includes)
	allExcludes := append(negatedExcludes, normalizePatterns(excludes)...)

	fsys := os.DirFS(root)
	seen := map[string]struct{}{}
	for _, pat := range normalizePatterns(includePats) {
		matches, err := doublestar.Glob(fsys, pat)
		if err != nil {
			return nil, fmt.Errorf("globbing include pattern %q: %w", pat, err)
		}
		for _, m := range matches {
			rel := normalizeRelDir(m)
			if !isDir(filepath.Join(root, filepath.FromSlash(rel))) {
				continue
			}
			if matchesAnyExclude(rel, allExcludes) {
				continue
			}
			seen[rel] = struct{}{}
		}
	}

	out := make([]string, 0, len(seen))
	for d := range seen {
		out = append(out, d)
	}
	sort.Strings(out)
	return out, nil
}

// splitNegations partitions include entries into true includes and the excludes
// implied by a leading "!" (pnpm/npm negation). The "!" prefix is stripped from
// negated entries.
func splitNegations(includes []string) (pos, neg []string) {
	for _, p := range includes {
		t := strings.TrimSpace(p)
		if t == "" {
			continue
		}
		if strings.HasPrefix(t, "!") {
			neg = append(neg, strings.TrimSpace(t[1:]))
			continue
		}
		pos = append(pos, t)
	}
	return pos, neg
}

// normalizePatterns trims, drops empties, and strips leading "./" from patterns
// so they are anchored at the workspace root the way doublestar expects.
func normalizePatterns(pats []string) []string {
	out := make([]string, 0, len(pats))
	for _, p := range pats {
		t := strings.TrimSpace(p)
		t = strings.TrimPrefix(t, "./")
		t = strings.TrimSuffix(t, "/")
		if t == "" {
			continue
		}
		out = append(out, t)
	}
	return out
}

// matchesAnyExclude reports whether the relative directory rel is excluded by
// any pattern in excludes. Each exclude is tested two ways to cover both glob
// excludes and Cargo's prefix-match semantics: a direct doublestar match, and a
// prefix/subtree match (the excluded directory itself and everything under it).
func matchesAnyExclude(rel string, excludes []string) bool {
	for _, ex := range excludes {
		if ex == "" {
			continue
		}
		if ok, err := doublestar.Match(ex, rel); err == nil && ok {
			return true
		}
		// Cargo prefix-match / pnpm directory exclusion: excluding "crates/old"
		// also excludes "crates/old/sub". A plain (meta-free) exclude likewise
		// names a directory subtree.
		if rel == ex || strings.HasPrefix(rel, ex+"/") {
			return true
		}
		if ok, err := doublestar.Match(ex+"/**", rel); err == nil && ok {
			return true
		}
	}
	return false
}

// isDir reports whether path exists and is a directory.
func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
