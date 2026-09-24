package nixlang_test

import (
	"testing"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem/modules/nixlang"
)

// TestManifestFiles_ReportsFlakeLock verifies Nix flake manifests reach
// Version-Sentinel coverage (as uncovered) instead of being omitted.
func TestManifestFiles_ReportsFlakeLock(t *testing.T) {
	t.Parallel()
	var mod ecosystem.EcosystemModule = &nixlang.Module{}
	report := ecosystem.AggregateManifestCoverage([]ecosystem.EcosystemModule{mod},
		func(ecosystem.EcosystemModule) ecosystem.ModuleConfig { return ecosystem.ModuleConfig{} })
	if len(report.Uncovered) != 1 {
		t.Fatalf("coverage Uncovered = %v, want the flake manifest listed", report.Uncovered)
	}
	got := report.Uncovered[0]
	if got.Path != "flake.nix" || got.LockFile != "flake.lock" || got.LockFilePolicy != ecosystem.LockFilePolicyRequired {
		t.Errorf("manifest = %+v, want flake.nix -> flake.lock (required)", got)
	}
}
