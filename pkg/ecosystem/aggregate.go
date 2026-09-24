package ecosystem

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
)

// ManifestCoverageReport summarizes manifest file coverage across detected
// ecosystems, partitioned by Version-Sentinel support status.
type ManifestCoverageReport struct {
	Covered      []ManifestFileInfo
	Uncovered    []ManifestFileInfo
	AllManifests []ManifestFileInfo
}

// HasUncovered returns true when at least one manifest is not covered.
func (r ManifestCoverageReport) HasUncovered() bool {
	return len(r.Uncovered) > 0
}

// AggregateVerificationCommands merges VerificationCommands from multiple
// ecosystem modules. It deduplicates commands within each category while
// preserving insertion order.
func AggregateVerificationCommands(
	modules []EcosystemModule,
	configFor func(EcosystemModule) ModuleConfig,
) VerificationCommands {
	var agg VerificationCommands

	for _, mod := range modules {
		vc := mod.VerificationCommands(configFor(mod))
		agg.Build = append(agg.Build, vc.Build...)
		agg.Test = append(agg.Test, vc.Test...)
		agg.Lint = append(agg.Lint, vc.Lint...)
		agg.TypeCheck = append(agg.TypeCheck, vc.TypeCheck...)
		agg.Format = append(agg.Format, vc.Format...)
	}

	agg.Build = sliceutil.Dedup(agg.Build)
	agg.Test = sliceutil.Dedup(agg.Test)
	agg.Lint = sliceutil.Dedup(agg.Lint)
	agg.TypeCheck = sliceutil.Dedup(agg.TypeCheck)
	agg.Format = sliceutil.Dedup(agg.Format)

	return agg
}

// AggregateManifestCoverage collects ManifestFileInfo from multiple modules
// and partitions them by Version-Sentinel support status.
func AggregateManifestCoverage(
	modules []EcosystemModule,
	configFor func(EcosystemModule) ModuleConfig,
) ManifestCoverageReport {
	var report ManifestCoverageReport

	for _, mod := range modules {
		mfp, ok := mod.(ManifestFileProvider)
		if !ok {
			continue
		}
		manifests := mfp.ManifestFiles(configFor(mod))
		for _, m := range manifests {
			report.AllManifests = append(report.AllManifests, m)
			if m.VSSupported {
				report.Covered = append(report.Covered, m)
			} else {
				report.Uncovered = append(report.Uncovered, m)
			}
		}
	}

	return report
}

// CIPhaseGroup is the CI commands of one pipeline phase, in the order the
// contributing modules returned them.
type CIPhaseGroup struct {
	Phase    CIPhase
	Commands []CICommand
}

// AggregateCICommands collects the CICommands of multiple ecosystem modules
// and groups them by phase in pipeline order (install, test, scan), so lock
// file enforcement runs before the tests and audits that depend on the
// installed dependencies. Phases without commands are omitted, and a command
// line contributed more than once runs only at its first occurrence. A
// command with an unknown phase is an error rather than being dropped: it
// would otherwise silently vanish from CI.
func AggregateCICommands(
	modules []EcosystemModule,
	configFor func(EcosystemModule) ModuleConfig,
) ([]CIPhaseGroup, error) {
	byPhase := make([][]CICommand, len(ciPhaseNames))
	seen := make(map[string]bool)

	for _, mod := range modules {
		for _, cmd := range mod.CICommands(configFor(mod)) {
			if int(cmd.Phase) < 0 || int(cmd.Phase) >= len(ciPhaseNames) {
				return nil, fmt.Errorf("%s CI command %q: unknown CI phase %d", mod.Name(), cmd.Name, int(cmd.Phase))
			}
			if seen[cmd.Command] {
				continue
			}
			seen[cmd.Command] = true
			byPhase[cmd.Phase] = append(byPhase[cmd.Phase], cmd)
		}
	}

	var groups []CIPhaseGroup
	for phase, cmds := range byPhase {
		if len(cmds) > 0 {
			groups = append(groups, CIPhaseGroup{Phase: CIPhase(phase), Commands: cmds})
		}
	}
	return groups, nil
}
