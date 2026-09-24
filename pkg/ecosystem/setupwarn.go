package ecosystem

import "github.com/Quantum-Serendipity/qsdev/pkg/types"

// SetupWarnings returns the warnings each of the answers' languages' module
// reports for the project at projectRoot (see SetupWarner), in language order,
// each prefixed with the module's display name. Each module sees the
// configuration generation uses (ToGenerationConfig). Languages without a
// registered module, and modules that do not implement SetupWarner,
// contribute nothing.
func (r *Registry) SetupWarnings(projectRoot string, answers types.WizardAnswers) []string {
	var warnings []string
	for _, lang := range answers.Languages {
		m, ok := r.ByName(lang.Name)
		if !ok {
			continue
		}
		w, ok := m.(SetupWarner)
		if !ok {
			continue
		}
		for _, msg := range w.SetupWarnings(projectRoot, ToGenerationConfig(lang, answers)) {
			warnings = append(warnings, m.DisplayName()+": "+msg)
		}
	}
	return warnings
}
