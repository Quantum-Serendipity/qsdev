package modules

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestManifestCoverage_ModulesWithLockfilesDeclareManifests verifies every
// module that advertises a package-manager lock file also declares its
// manifests, so Version-Sentinel coverage lists them (covered or not)
// instead of silently omitting the ecosystem.
func TestManifestCoverage_ModulesWithLockfilesDeclareManifests(t *testing.T) {
	t.Parallel()
	for _, mod := range ecosystem.DefaultRegistry().All() {
		hasLock := false
		for _, pm := range mod.PackageManagers() {
			if pm.LockFile != "" {
				hasLock = true
			}
		}
		if !hasLock {
			continue
		}
		t.Run(mod.Name(), func(t *testing.T) {
			t.Parallel()
			mfp, ok := mod.(ecosystem.ManifestFileProvider)
			if !ok {
				t.Fatalf("%s advertises a lock file but does not implement ManifestFileProvider", mod.Name())
			}
			if len(mfp.ManifestFiles(ecosystem.ModuleConfig{})) == 0 {
				t.Errorf("%s declares no manifest files", mod.Name())
			}
		})
	}
}
