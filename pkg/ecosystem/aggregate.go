package ecosystem

import "github.com/Quantum-Serendipity/qsdev/internal/sliceutil"

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
		manifests := mod.ManifestFiles(configFor(mod))
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
