package ecosystem

import (
	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ResolveLanguageModules returns the registered modules for the given language
// choices (unknown names are skipped), plus a configFor function suitable for
// the aggregate helpers such as AggregateVerificationCommands and
// AggregateManifestCoverage. It is the single implementation shared by every
// caller that needs "the modules for this project's languages".
func ResolveLanguageModules(
	languages []types.LanguageChoice,
	registry *Registry,
) ([]EcosystemModule, func(EcosystemModule) ModuleConfig) {
	configFor := func(mod EcosystemModule) ModuleConfig {
		for _, lang := range languages {
			if lang.Name == mod.Name() {
				return ToModuleConfig(lang)
			}
		}
		return ModuleConfig{}
	}

	var modules []EcosystemModule
	for _, lang := range languages {
		if mod, ok := registry.ByName(lang.Name); ok {
			modules = append(modules, mod)
		}
	}
	return modules, configFor
}

// LanguageManifestCoverage aggregates Version-Sentinel manifest coverage for
// the modules of the given language choices.
func LanguageManifestCoverage(languages []types.LanguageChoice, registry *Registry) ManifestCoverageReport {
	return AggregateManifestCoverage(ResolveLanguageModules(languages, registry))
}

// LanguageVerificationCommands returns the de-duplicated build, test and lint
// commands of the modules for the given language choices. It backs both the
// generated agent-postmortem skill and the MCP verification checklist, so the
// two always list the same commands.
func LanguageVerificationCommands(languages []types.LanguageChoice, registry *Registry) []string {
	return sliceutil.Dedup(AggregateVerificationCommands(ResolveLanguageModules(languages, registry)).All())
}
