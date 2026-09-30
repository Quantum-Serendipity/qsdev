package vsentinel

import (
	"path"
	"path/filepath"
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
