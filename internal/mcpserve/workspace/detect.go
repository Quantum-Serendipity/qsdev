package workspace

import (
	"fmt"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"sync"
	"time"
)

// DetectWorkspaces scans root for the five workspace configuration formats and
// returns the flat WorkspaceGraph of every member package.
//
// Each ecosystem parser runs in its own goroutine with failure isolation: an
// absent configuration is skipped silently, and a parser that errors or panics
// is logged and dropped without blocking the others. The resolved directories
// are validated to actually contain the ecosystem's manifest before becoming
// packages, so a stray glob match never produces a phantom package.
//
// DetectWorkspaces always returns a non-nil graph (possibly empty) and a nil
// error; it degrades rather than failing so a non-monorepo project simply yields
// an empty graph. Timing is recorded at debug level to track the <100ms/50pkg
// and <1s/500pkg performance targets.
func DetectWorkspaces(root string) (*WorkspaceGraph, error) {
	start := time.Now()
	resolver := NewGlobResolver()
	specs := allEcosystems()

	type scan struct {
		spec ecosystemSpec
		dirs []string
	}
	results := make([]*scan, len(specs))

	var wg sync.WaitGroup
	for i := range specs {
		i := i
		spec := specs[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			// Failure isolation: a panic in any single parser must not crash the
			// scan or poison the others.
			defer func() {
				if r := recover(); r != nil {
					slog.Warn("workspace parser panicked; skipping ecosystem",
						"ecosystem", spec.id, "recover", r)
				}
			}()

			if !spec.configPresent(root) {
				return // configuration absent for this ecosystem
			}
			includes, excludes, err := spec.parse(root)
			if err != nil {
				slog.Warn("workspace parser failed; skipping ecosystem",
					"ecosystem", spec.id, "config", spec.configFile, "error", err)
				return
			}
			dirs, err := resolver.ResolvePatterns(root, includes, excludes)
			if err != nil {
				slog.Warn("workspace glob resolution failed; skipping ecosystem",
					"ecosystem", spec.id, "error", err)
				return
			}
			results[i] = &scan{spec: spec, dirs: dirs}
		}()
	}
	wg.Wait()

	packages := map[string]*Package{}
	// Iterate in spec order so cross-ecosystem directory collisions resolve
	// deterministically (the earlier ecosystem wins).
	for _, res := range results {
		if res == nil {
			continue
		}
		for _, dir := range res.dirs {
			pkg, err := buildPackage(root, res.spec, dir)
			if err != nil {
				slog.Debug("skipping resolved directory without a valid manifest",
					"ecosystem", res.spec.id, "dir", dir, "error", err)
				continue
			}
			if existing, ok := packages[dir]; ok {
				slog.Debug("workspace directory claimed by multiple ecosystems; keeping first",
					"dir", dir, "kept", existing.Ecosystem, "dropped", res.spec.id)
				continue
			}
			packages[dir] = pkg
		}
	}

	graph := &WorkspaceGraph{root: root, packages: packages}
	slog.Debug("workspace detection complete",
		"packages", len(packages), "duration", time.Since(start))
	return graph, nil
}

// buildPackage validates that the resolved directory relDir contains the
// ecosystem's manifest and, if so, reads the manifest to construct the Package.
// It returns an error when the manifest is missing or unreadable.
func buildPackage(root string, spec ecosystemSpec, relDir string) (*Package, error) {
	manifestRel := relManifestPath(relDir, spec.manifest)
	absManifest := filepath.Join(root, filepath.FromSlash(manifestRel))
	info, err := os.Stat(absManifest)
	if err != nil {
		return nil, fmt.Errorf("manifest %s not found: %w", manifestRel, err)
	}
	if info.IsDir() {
		return nil, fmt.Errorf("manifest %s is a directory", manifestRel)
	}

	name, deps, err := spec.readManifest(absManifest)
	if err != nil {
		return nil, fmt.Errorf("reading manifest %s: %w", manifestRel, err)
	}
	if name == "" {
		name = fallbackName(root, relDir)
	}

	return &Package{
		Name:         name,
		RelDir:       relDir,
		Ecosystem:    spec.id,
		ManifestPath: manifestRel,
		Dependencies: deps,
	}, nil
}

// relManifestPath joins a package's relative directory with its manifest
// filename, using slash paths and treating the workspace root (".") as the
// manifest sitting directly at the root.
func relManifestPath(relDir, manifest string) string {
	if relDir == "." || relDir == "" {
		return manifest
	}
	return path.Join(relDir, manifest)
}

// fallbackName derives a package name from its directory when the manifest
// declares none.
func fallbackName(root, relDir string) string {
	if relDir == "." || relDir == "" {
		return filepath.Base(root)
	}
	return path.Base(relDir)
}

// fileExists reports whether path exists and is a regular (non-directory) file.
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// configPresent reports whether root contains this ecosystem's membership
// configuration, accepting either its primary configFile or any alternate
// spelling in configAlts (e.g. pnpm's legacy "pnpm-workspace.yml"). A repo whose
// only workspace file is an alternate spelling is therefore still detected.
func (s ecosystemSpec) configPresent(root string) bool {
	if fileExists(filepath.Join(root, s.configFile)) {
		return true
	}
	for _, alt := range s.configAlts {
		if fileExists(filepath.Join(root, alt)) {
			return true
		}
	}
	return false
}
