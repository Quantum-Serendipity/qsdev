package ecosystem

import "github.com/Quantum-Serendipity/qsdev/pkg/types"

// SetupWarnings returns the warnings each language's module reports for the
// project at projectRoot (see SetupWarner), in language order, each prefixed
// with the module's display name. Languages without a registered module, and
// modules that do not implement SetupWarner, contribute nothing.
func (r *Registry) SetupWarnings(projectRoot string, langs []types.LanguageChoice) []string {
	var warnings []string
	for _, lang := range langs {
		m, ok := r.ByName(lang.Name)
		if !ok {
			continue
		}
		w, ok := m.(SetupWarner)
		if !ok {
			continue
		}
		for _, msg := range w.SetupWarnings(projectRoot, ToModuleConfig(lang)) {
			warnings = append(warnings, m.DisplayName()+": "+msg)
		}
	}
	return warnings
}
