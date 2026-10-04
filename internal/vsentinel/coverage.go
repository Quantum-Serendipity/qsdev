package vsentinel

import (
	"path"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// CoverageLevel is how thoroughly Version-Sentinel can check a manifest: the
// capability counterpart of the Verification a DetectDrift run records.
type CoverageLevel = Verification

// Coverage reports the level at which DetectDrift checks manifest when it is
// locked by lockFile. It is derived from driftPairs, so from the catalog and
// specificCheckers: a manifest is diffed only when it belongs to an ecosystem
// with a dedicated checker and lockFile is that checker's primary lockfile
// (empty means the default); any other catalog manifest — generic ecosystems,
// non-primary lockfiles such as pnpm-lock.yaml for package.json — gets a
// lockfile-presence check; and a manifest outside the catalog (mix.exs, *.tf,
// ...) is uncovered. Both names may carry a directory; only the base names
// are compared.
func Coverage(manifest, lockFile string) CoverageLevel {
	base := path.Base(filepath.ToSlash(manifest))
	lock := ""
	if lockFile != "" {
		lock = path.Base(filepath.ToSlash(lockFile))
	}
	level := VerificationUncovered
	for _, pair := range driftPairs() {
		if manifest == "" || !matchesManifest(pair.manifest, base) {
			continue
		}
		if pair.checker != nil && (lock == "" || lock == pair.primaryLock) {
			return VerificationDiffed
		}
		level = VerificationPresenceOnly
	}
	return level
}

// matchesManifest reports whether the file name base matches the catalog
// manifest pattern, which may be a glob (e.g. "*.csproj"), or whether base
// is the pattern itself.
func matchesManifest(pattern, base string) bool {
	ok, err := path.Match(pattern, base)
	return pattern == base || (err == nil && ok)
}

// ManifestCoverage partitions manifests by Coverage: the one classification
// behind every Version-Sentinel coverage surface (the manifest_coverage MCP
// tool, the generated CLAUDE.md section and .version-sentinel/ignore), so
// they cannot disagree about a manifest.
type ManifestCoverage struct {
	// Diffed manifests have their declared versions compared with the
	// lockfile.
	Diffed []ecosystem.ManifestFileInfo
	// PresenceOnly manifests are checked only for a lockfile.
	PresenceOnly []ecosystem.ManifestFileInfo
	// Uncovered manifests are not checked at all.
	Uncovered []ecosystem.ManifestFileInfo
	// NotDiffed is PresenceOnly and Uncovered together, in input order: the
	// manifests whose versions are never compared.
	NotDiffed []ecosystem.ManifestFileInfo
}

// ClassifyManifests sorts manifests by Coverage, keeping input order within
// each class.
func ClassifyManifests(manifests []ecosystem.ManifestFileInfo) ManifestCoverage {
	var c ManifestCoverage
	for _, m := range manifests {
		switch Coverage(m.Path, m.LockFile) {
		case VerificationDiffed:
			c.Diffed = append(c.Diffed, m)
			continue
		case VerificationPresenceOnly:
			c.PresenceOnly = append(c.PresenceOnly, m)
		default:
			c.Uncovered = append(c.Uncovered, m)
		}
		c.NotDiffed = append(c.NotDiffed, m)
	}
	return c
}

// ManifestPaths returns the Path of each manifest.
func ManifestPaths(manifests []ecosystem.ManifestFileInfo) []string {
	paths := make([]string, len(manifests))
	for i, m := range manifests {
		paths[i] = m.Path
	}
	return paths
}
