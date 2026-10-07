package devenv

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// genContext is what one generation run reads from: the answers (after the
// generator's infra profile is applied), the catalog and tool registry loaded
// once, the ecosystem module registry, and each selected module's
// ModuleConfig computed once. Loading both defaults up front makes every
// generator fail closed, with an error naming the broken file, instead of
// panicking or silently dropping catalog security defaults; computing the
// configs once makes every generated file agree on each module's settings.
type genContext struct {
	answers types.WizardAnswers
	cat     *catalog.Catalog
	tools   *toolreg.Registry
	modules *ecosystem.Registry               // nil for minimal (no-language) generation
	configs map[string]ecosystem.ModuleConfig // keyed by module Name()
}

// newGenContext loads the default catalog and tool registry for generating
// from answers with the given module registry, which may be nil. With a
// registry, every selected language must name a known module, and each
// module's ModuleConfig is derived once: the language entry completed from
// detection (WithSuggested), with the effective infrastructure, the
// project-level module settings and the compliance level's release-age
// window (ecosystem.ToGenerationConfig).
func newGenContext(answers types.WizardAnswers, modules *ecosystem.Registry) (*genContext, error) {
	cat, err := catalog.Default()
	if err != nil {
		return nil, fmt.Errorf("loading catalog: %w", err)
	}
	tools, err := toolreg.Default()
	if err != nil {
		return nil, fmt.Errorf("loading tool registry: %w", err)
	}
	configs := make(map[string]ecosystem.ModuleConfig, len(answers.Languages))
	if modules != nil {
		minReleaseAge := cat.AgeGate(answers.ComplianceLevel)
		for _, lang := range answers.Languages {
			mod, ok := modules.ByName(lang.Name)
			if !ok {
				return nil, fmt.Errorf("unknown language module: %q", lang.Name)
			}
			configs[mod.Name()] = ecosystem.ToGenerationConfig(answers.Detected.WithSuggested(lang), answers, minReleaseAge)
		}
	}
	return &genContext{answers: answers, cat: cat, tools: tools, modules: modules, configs: configs}, nil
}

// moduleConfig is the ModuleConfig every generator passes to mod, a selected
// language's module.
func (c *genContext) moduleConfig(mod ecosystem.EcosystemModule) ecosystem.ModuleConfig {
	return c.configs[mod.Name()]
}
