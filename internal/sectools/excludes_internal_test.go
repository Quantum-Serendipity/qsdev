package sectools

import (
	"slices"
	"strings"
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// TestLicenseScanIgnoresMatchScanExcludes keeps the license scan's ignore list
// in step with the shared scan exclusions: every exclusion is ignored by the
// license scan too, except the dependency directories, whose licenses are
// what the scan checks.
func TestLicenseScanIgnoresMatchScanExcludes(t *testing.T) {
	t.Parallel()
	dependencyDirs := []string{"node_modules", "third_party", "vendor", ".venv", "venv"}
	ignores := ecosystem.LicenseScanIgnores()

	for _, p := range defaultScanExcludes {
		name := strings.TrimSuffix(p, "/")
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			isDep := slices.Contains(dependencyDirs, name)
			ignored := slices.Contains(ignores, name)
			switch {
			case isDep && ignored:
				t.Errorf("license scan ignores dependency directory %q; its licenses must be scanned", name)
			case !isDep && !ignored:
				t.Errorf("license scan does not ignore shared scan exclusion %q", name)
			}
		})
	}
}
