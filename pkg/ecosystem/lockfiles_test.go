package ecosystem

import (
	"slices"
	"testing"
)

// TestManifestAndLockfileCatalogsShareKeys pins the invariant the
// ManifestsByEcosystem doc promises: it shares its keys with
// LockFilesByEcosystem, so every catalog ecosystem has both a manifest and a
// lockfile entry. Drift detection derives its coverage by iterating the
// manifest map and looking lockfiles up in the other, so an ecosystem present
// in only one map would be silently half-covered (or not covered at all).
func TestManifestAndLockfileCatalogsShareKeys(t *testing.T) {
	t.Parallel()

	for eco, manifests := range ManifestsByEcosystem {
		if len(manifests) == 0 {
			t.Errorf("ecosystem %q has an empty manifest list", eco)
		}
		if len(LockFilesByEcosystem[eco]) == 0 {
			t.Errorf("ecosystem %q has manifests but no LockFilesByEcosystem entry", eco)
		}
	}
	for eco := range LockFilesByEcosystem {
		if _, ok := ManifestsByEcosystem[eco]; !ok {
			t.Errorf("ecosystem %q has lockfiles but no ManifestsByEcosystem entry", eco)
		}
	}
}

// TestManifestLockfilePairsMatchCatalog keeps the drift pairs consistent with
// the ecosystem catalog: every pair must be a catalog manifest/lockfile of the
// same ecosystem, and a pair's lockfile must never be the manifest itself.
func TestManifestLockfilePairsMatchCatalog(t *testing.T) {
	t.Parallel()

	for _, p := range ManifestLockfilePairs {
		t.Run(p.Manifest+"->"+p.Lockfile, func(t *testing.T) {
			t.Parallel()
			if p.Manifest == p.Lockfile {
				t.Fatal("a manifest must not be paired with itself as its lockfile")
			}
			for eco, manifests := range ManifestsByEcosystem {
				if slices.Contains(manifests, p.Manifest) && slices.Contains(LockFilesByEcosystem[eco], p.Lockfile) {
					return
				}
			}
			t.Errorf("pair is not a manifest/lockfile of any single catalog ecosystem")
		})
	}
}

// TestLockfileCatalogsCoverPackageManagers pins lockfiles each package
// manager writes by default, so their presence is recognised everywhere the
// catalog is consulted (e.g. Bun >= 1.2 writes the text bun.lock).
func TestLockfileCatalogsCoverPackageManagers(t *testing.T) {
	t.Parallel()

	tests := []struct {
		eco, manifest, lockfile string
	}{
		{NameJavaScript, "package.json", "npm-shrinkwrap.json"},
		{NameJavaScript, "package.json", "bun.lock"},
		{NameJavaScript, "package.json", "bun.lockb"},
		{NamePython, "Pipfile", "Pipfile.lock"},
		{NameRuby, "Gemfile", "Gemfile.lock"},
		{NamePHP, "composer.json", "composer.lock"},
		{NameNix, "flake.nix", "flake.lock"},
	}
	for _, tt := range tests {
		t.Run(tt.lockfile, func(t *testing.T) {
			t.Parallel()
			if !slices.Contains(LockFilesByEcosystem[tt.eco], tt.lockfile) {
				t.Errorf("LockFilesByEcosystem[%q] missing %q", tt.eco, tt.lockfile)
			}
			if !slices.Contains(ManifestLockfilePairs, LockFilePair{tt.manifest, tt.lockfile}) {
				t.Errorf("ManifestLockfilePairs missing {%q, %q}", tt.manifest, tt.lockfile)
			}
		})
	}
}

// TestGroupedManifestLockfiles_Workspace checks that every workspace lockfile
// is a catalog lockfile (a typo would silently disable the workspace-root
// lookup) and that grouping marks it on its manifest only.
func TestGroupedManifestLockfiles_Workspace(t *testing.T) {
	t.Parallel()

	for _, lf := range WorkspaceLockfiles {
		if !slices.ContainsFunc(ManifestLockfilePairs, func(p LockFilePair) bool { return p.Lockfile == lf }) {
			t.Errorf("workspace lockfile %q is not in ManifestLockfilePairs", lf)
		}
	}

	want := map[string][]string{
		"package.json":   {"npm-shrinkwrap.json", "package-lock.json", "pnpm-lock.yaml", "yarn.lock", "bun.lock", "bun.lockb"},
		"pyproject.toml": {"uv.lock"},
		"Cargo.toml":     {"Cargo.lock"},
		"go.mod":         nil,
		"Gemfile":        nil,
	}
	for _, g := range GroupedManifestLockfiles() {
		w, ok := want[g.Manifest]
		if !ok {
			continue
		}
		if !slices.Equal(g.Workspace, w) {
			t.Errorf("%s: Workspace = %v, want %v", g.Manifest, g.Workspace, w)
		}
	}
}
