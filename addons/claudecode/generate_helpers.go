package claudecode

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// loadYAMLManifest reads a YAML file from the embedded template filesystem and
// unmarshals it into the type parameter T. The path is relative to the embed
// root (e.g. "templates/skills/manifest.yaml").
func loadYAMLManifest[T any](path string) (*T, error) {
	data, err := templateFS.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}

	var manifest T
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}

	return &manifest, nil
}

// hookFileSpec describes a single hook file to be generated from a template.
type hookFileSpec struct {
	enabled      bool
	templatePath string
	outputPath   string
	mode         os.FileMode
	strategy     types.MergeStrategy
	owner        string
}

// generateHookFile reads an embedded template and returns a GeneratedFile when
// the spec is enabled. It returns (nil, nil) when the spec is disabled.
func generateHookFile(spec hookFileSpec) (*types.GeneratedFile, error) {
	if !spec.enabled {
		return nil, nil
	}

	content, err := templateFS.ReadFile(spec.templatePath)
	if err != nil {
		return nil, fmt.Errorf("reading hook template %s: %w", spec.templatePath, err)
	}

	return &types.GeneratedFile{
		Path:     spec.outputPath,
		Content:  content,
		Mode:     spec.mode,
		Strategy: spec.strategy,
		Owner:    spec.owner,
	}, nil
}

// resolveLanguageModules returns the ecosystem modules that correspond to the
// user's language choices, plus a configFor function suitable for passing to
// aggregate helpers like AggregateVerificationCommands and
// AggregateManifestCoverage. This eliminates the duplicated module-gathering
// boilerplate in generate_postmortem.go and generate_version_sentinel.go.
func resolveLanguageModules(
	answers types.WizardAnswers,
	registry *ecosystem.Registry,
) ([]ecosystem.EcosystemModule, func(ecosystem.EcosystemModule) ecosystem.ModuleConfig) {
	configFor := func(mod ecosystem.EcosystemModule) ecosystem.ModuleConfig {
		for _, lang := range answers.Languages {
			if lang.Name == mod.Name() {
				return ecosystem.ToModuleConfig(lang)
			}
		}
		return ecosystem.ModuleConfig{}
	}

	var modules []ecosystem.EcosystemModule
	for _, lang := range answers.Languages {
		if mod, ok := registry.ByName(lang.Name); ok {
			modules = append(modules, mod)
		}
	}

	return modules, configFor
}
