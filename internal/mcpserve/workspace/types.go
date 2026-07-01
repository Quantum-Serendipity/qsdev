// Package workspace implements the V2 monorepo workspace detection engine for
// the universal qsdev MCP server (Phase 32, Unit 32.7). It auto-detects
// sub-packages across the five workspace configuration formats that cover every
// major ecosystem — npm/Yarn (package.json "workspaces"), pnpm
// (pnpm-workspace.yaml), Cargo ([workspace] members/exclude), Go (go.work use),
// and Python/uv ([tool.uv.workspace]) — resolves their glob patterns through a
// doublestar-backed compatibility layer, and renders the result as a flat
// WorkspaceGraph keyed by package directory.
//
// The engine watches the workspace configuration with a two-category fsnotify
// strategy (membership configs trigger a full re-scan; member manifests trigger
// a single-package re-parse) and exposes per-package context through the
// qsdev://project/{package}/context resource, using {ecosystem}:{name}
// qualification for cross-ecosystem name collisions.
//
// To avoid an import cycle the package depends only on spi (never on mcpserve);
// the watcher signals catalog changes through an injected callback rather than
// importing the server.
package workspace

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// Package is a single workspace member: one ecosystem sub-package rooted at a
// directory relative to the workspace root.
type Package struct {
	// Name is the package's declared name from its manifest (the npm "name", the
	// Cargo [package] name, the Go module path, or the Python [project] name).
	Name string
	// RelDir is the package directory relative to the workspace root, using
	// forward slashes. The workspace root itself is ".".
	RelDir string
	// Ecosystem is the workspace-tool identity that detected the package: one of
	// "npm", "pnpm", "cargo", "go", or "uv". It forms the prefix in the
	// {ecosystem}:{name} qualified name.
	Ecosystem string
	// ManifestPath is the package's manifest file relative to the workspace
	// root, using forward slashes (e.g. "packages/utils/package.json").
	ManifestPath string
	// Dependencies is the list of declared dependency names parsed from the
	// manifest (best-effort; empty when the manifest declares none).
	Dependencies []string
}

// QualifiedName returns the package's cross-ecosystem-unique identifier in
// {ecosystem}:{name} form (e.g. "npm:@myorg/utils").
func (p *Package) QualifiedName() string {
	return p.Ecosystem + ":" + p.Name
}

// WorkspaceGraph is the flat set of workspace packages keyed by relative
// directory. It is safe for concurrent use: reads take a read lock and the
// watcher's mutations take the write lock.
type WorkspaceGraph struct {
	mu       sync.RWMutex
	root     string
	packages map[string]*Package // keyed by RelDir
}

// NewWorkspaceGraph constructs an empty graph rooted at root.
func NewWorkspaceGraph(root string) *WorkspaceGraph {
	return &WorkspaceGraph{root: root, packages: map[string]*Package{}}
}

// Root returns the absolute workspace root the graph describes.
func (g *WorkspaceGraph) Root() string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.root
}

// Len returns the number of packages in the graph.
func (g *WorkspaceGraph) Len() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.packages)
}

// Packages returns every package, sorted by RelDir for determinism. Each
// returned pointer is the live entry; callers must treat them as read-only.
func (g *WorkspaceGraph) Packages() []*Package {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]*Package, 0, len(g.packages))
	for _, p := range g.packages {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RelDir < out[j].RelDir })
	return out
}

// ByRelDir returns the package rooted at dir (relative to the workspace root),
// or nil. The argument is normalized to forward-slash, cleaned form so callers
// may pass either "packages/a" or "./packages/a".
func (g *WorkspaceGraph) ByRelDir(dir string) *Package {
	key := normalizeRelDir(dir)
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.packages[key]
}

// ByQualifiedName resolves a package by name. It accepts the fully qualified
// {ecosystem}:{name} form (preferred, collision-free) and also a bare name,
// which resolves only when exactly one package carries that name. It returns nil
// when nothing matches or a bare name is ambiguous.
func (g *WorkspaceGraph) ByQualifiedName(name string) *Package {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if eco, bare, ok := splitQualified(name); ok {
		for _, p := range g.packages {
			if p.Ecosystem == eco && p.Name == bare {
				return p
			}
		}
		return nil
	}

	var match *Package
	for _, p := range g.packages {
		if p.Name == name {
			if match != nil {
				return nil // ambiguous bare name; require qualification
			}
			match = p
		}
	}
	return match
}

// Resolve locates a package by the identifier embedded in the
// qsdev://project/{package}/context URI. It tries, in order, an exact relative
// directory match, then a qualified or bare name match. It returns nil when no
// package matches.
func (g *WorkspaceGraph) Resolve(id string) *Package {
	if p := g.ByRelDir(id); p != nil {
		return p
	}
	return g.ByQualifiedName(id)
}

// ResolveCWDPackage provides V1 backward compatibility: given an absolute
// working directory, it returns the package whose RelDir is the nearest ancestor
// of cwd (the most specific enclosing package). It returns an error when cwd
// lies outside the workspace root or no package encloses it.
func (g *WorkspaceGraph) ResolveCWDPackage(cwd string) (*Package, error) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		return nil, fmt.Errorf("resolving cwd %q: %w", cwd, err)
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	rel, err := filepath.Rel(g.root, abs)
	if err != nil {
		return nil, fmt.Errorf("relating cwd %q to root %q: %w", abs, g.root, err)
	}
	rel = filepath.ToSlash(rel)
	if rel == ".." || strings.HasPrefix(rel, "../") {
		return nil, fmt.Errorf("cwd %q is outside workspace root %q", abs, g.root)
	}

	var best *Package
	bestLen := -1
	for _, p := range g.packages {
		if !relDirEncloses(p.RelDir, rel) {
			continue
		}
		if l := relDirSpecificity(p.RelDir); l > bestLen {
			best, bestLen = p, l
		}
	}
	if best == nil {
		return nil, fmt.Errorf("no workspace package encloses %q", abs)
	}
	return best, nil
}

// ReplaceAll atomically swaps the graph's package set (used by the watcher after
// a full re-scan). The replacement map is keyed by RelDir.
func (g *WorkspaceGraph) ReplaceAll(packages map[string]*Package) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.packages = packages
}

// Upsert inserts or replaces a single package entry (used by the watcher after a
// member-manifest re-parse). A nil package or one with an empty RelDir is
// ignored.
func (g *WorkspaceGraph) Upsert(p *Package) {
	if p == nil || p.RelDir == "" {
		return
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	g.packages[p.RelDir] = p
}

// Remove deletes the package rooted at dir, returning whether an entry existed.
func (g *WorkspaceGraph) Remove(dir string) bool {
	key := normalizeRelDir(dir)
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.packages[key]; !ok {
		return false
	}
	delete(g.packages, key)
	return true
}

// normalizeRelDir cleans a directory path into the graph's canonical key form:
// forward slashes, no leading "./", with the root represented as ".".
func normalizeRelDir(dir string) string {
	d := filepath.ToSlash(filepath.Clean(dir))
	if d == "" {
		return "."
	}
	return d
}

// splitQualified parses an {ecosystem}:{name} identifier. It reports ok only for
// a non-empty ecosystem drawn from the known set, so a bare scoped npm name that
// happens to contain ":" is not mistaken for a qualification.
func splitQualified(id string) (eco, name string, ok bool) {
	i := strings.IndexByte(id, ':')
	if i <= 0 || i == len(id)-1 {
		return "", "", false
	}
	eco, name = id[:i], id[i+1:]
	if !knownEcosystem(eco) {
		return "", "", false
	}
	return eco, name, true
}

// knownEcosystem reports whether eco is one of the workspace-tool identities the
// engine assigns.
func knownEcosystem(eco string) bool {
	switch eco {
	case ecoNpm, ecoPnpm, ecoCargo, ecoGo, ecoUv:
		return true
	default:
		return false
	}
}

// relDirEncloses reports whether the package directory pkgDir encloses the
// relative path rel (pkgDir == rel, or rel is nested under pkgDir, or pkgDir is
// the workspace root ".").
func relDirEncloses(pkgDir, rel string) bool {
	if pkgDir == "." {
		return true
	}
	return rel == pkgDir || strings.HasPrefix(rel, pkgDir+"/")
}

// relDirSpecificity ranks how specific a package directory is for nearest-
// ancestor selection: the root "." is least specific, then path depth.
func relDirSpecificity(pkgDir string) int {
	if pkgDir == "." {
		return 0
	}
	return len(strings.Split(pkgDir, "/"))
}
