// Package workspace implements the V2 monorepo workspace detection engine for
// the universal qsdev MCP server (Phase 32, Unit 32.7). It auto-detects
// sub-packages across the five workspace configuration formats that cover every
// major ecosystem — npm/Yarn (package.json "workspaces"), pnpm
// (pnpm-workspace.yaml), Cargo ([workspace] members/exclude), Go (go.work use),
// and Python/uv ([tool.uv.workspace]) — resolves their glob patterns through a
// doublestar-backed compatibility layer, and renders the result as a flat
// WorkspaceGraph keyed by package directory.
//
// The engine exposes per-package context through the
// qsdev://project/{package}/context resource, using {ecosystem}:{name}
// qualification for cross-ecosystem name collisions.
//
// To avoid an import cycle the package depends only on spi (never on mcpserve).
package workspace

import (
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
