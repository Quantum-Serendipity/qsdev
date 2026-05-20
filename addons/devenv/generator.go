package devenv

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/profile"
	"github.com/Quantum-Serendipity/qsdev/internal/tier"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Compile-time interface check.
var _ types.Generator = (*DevenvGenerator)(nil)

// DevenvGenerator orchestrates all devenv sub-generators to produce the
// complete set of files for a security-hardened development environment.
type DevenvGenerator struct {
	registry        *ecosystem.Registry
	profileRegistry *profile.ProfileRegistry
}

// DevenvGeneratorOption configures a DevenvGenerator.
type DevenvGeneratorOption func(*DevenvGenerator)

// WithProfileRegistry returns an option that sets the profile registry on the generator.
func WithProfileRegistry(pr *profile.ProfileRegistry) DevenvGeneratorOption {
	return func(g *DevenvGenerator) {
		g.profileRegistry = pr
	}
}

// NewDevenvGenerator creates a DevenvGenerator backed by the given ecosystem
// module registry. The registry may be nil for minimal (no-language) generation.
// Optional DevenvGeneratorOption values configure additional features.
func NewDevenvGenerator(registry *ecosystem.Registry, opts ...DevenvGeneratorOption) *DevenvGenerator {
	g := &DevenvGenerator{registry: registry}
	for _, opt := range opts {
		opt(g)
	}
	return g
}

// Generate produces the full set of generated files from wizard answers:
//  1. devenv.yaml — hardened devenv configuration
//  2. devenv.nix  — Nix expression with packages, services, hooks
//  3. .envrc      — direnv activation (when enabled)
//  4. Per-language security configs (e.g. .npmrc, pip.conf)
//  5. nix.conf hardening guide (opt-in)
//  6. Profile-driven configs (CI workflow, Renovate/Dependabot, security docs)
func (g *DevenvGenerator) Generate(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	var files []types.GeneratedFile

	// 1. devenv.yaml
	yamlFile, err := GenerateDevenvYaml(answers, g.registry)
	if err != nil {
		return nil, fmt.Errorf("generating devenv.yaml: %w", err)
	}
	if yamlFile != nil {
		files = append(files, *yamlFile)
	}

	// 2. devenv.nix
	nixFile, err := GenerateDevenvNix(answers, g.registry)
	if err != nil {
		return nil, fmt.Errorf("generating devenv.nix: %w", err)
	}
	if nixFile != nil {
		files = append(files, *nixFile)
	}

	// 3. .envrc (only when direnv is enabled)
	envrcFile := GenerateEnvrc(answers)
	if envrcFile != nil {
		files = append(files, *envrcFile)
	}

	// 4. Per-language security configuration files
	if g.registry != nil {
		for _, lang := range answers.Languages {
			mod, ok := g.registry.ByName(lang.Name)
			if !ok {
				return nil, fmt.Errorf("unknown language module: %q", lang.Name)
			}
			cfg := ecosystem.ToModuleConfigWithProxy(lang, answers.Infrastructure)
			secFiles := mod.SecurityConfigs(cfg)
			files = append(files, secFiles...)
		}
	}

	// 5. nix.conf hardening guide (opt-in)
	nixHardeningFile, err := GenerateNixHardeningGuide(answers)
	if err != nil {
		return nil, fmt.Errorf("generating nix-conf-hardening guide: %w", err)
	}
	if nixHardeningFile != nil {
		files = append(files, *nixHardeningFile)
	}

	// 6. Profile-driven configs (CI workflow, Renovate/Dependabot, security docs)
	// Requires tier >= Standard: these are opinionated workflow configs.
	t, _ := tier.ParseTier(answers.Tier)
	if t >= tier.Standard && g.profileRegistry != nil {
		profileName := answers.ProfileName
		if profileName == "" {
			profileName = "consulting-default"
		}
		p, ok := g.profileRegistry.Get(profileName)
		if ok {
			files = append(files, p.ConfigFiles()...)
		}
	}

	return files, nil
}
