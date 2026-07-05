package ecosystem

import "testing"

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
