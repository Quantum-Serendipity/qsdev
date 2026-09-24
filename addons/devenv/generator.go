package devenv

import (
	"fmt"
	"maps"

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

// Generate produces the full set of generated files from wizard answers,
// after applying an explicitly selected infra profile (applyInfraProfile):
//  1. devenv.yaml — hardened devenv configuration
//  2. devenv.nix  — Nix expression with packages, services, hooks
//  3. .envrc      — direnv activation (when enabled)
//  4. Per-language security configs (e.g. .npmrc, pip.conf)
//  5. nix.conf hardening guide (opt-in)
//  6. Profile-driven configs (CI workflow, Renovate/Dependabot, security docs)
func (g *DevenvGenerator) Generate(answers types.WizardAnswers) ([]types.GeneratedFile, error) {
	var files []types.GeneratedFile

	// 0. Infra profile: an explicitly selected profile's registry proxy, Nix
	// cache and build cache become the effective infrastructure every step
	// below generates from.
	answers, infraProfile, err := g.applyInfraProfile(answers)
	if err != nil {
		return nil, err
	}

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
			cfg := ecosystem.ToModuleConfigWithInfra(lang, answers.Infrastructure)
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

	// 5b. NixOS Podman rootless guide (Podman + NixOS auto-detected)
	podmanGuide, err := GenerateNixosPodmanGuide(answers)
	if err != nil {
		return nil, fmt.Errorf("generating nixos-podman-rootless guide: %w", err)
	}
	if podmanGuide != nil {
		files = append(files, *podmanGuide)
	}

	// 6. Profile-driven configs (CI workflow, Renovate/Dependabot, security docs).
	// Requires tier >= Standard: these are opinionated workflow configs.
	// This InfraProfile.ConfigFiles path is the SOLE generator of the project CI /
	// security-scan workflow (the former internal/cigeneration workflow producer was
	// dead code and has been removed).
	t := tier.Resolve(answers.Tier, answers.PermissionLevel, answers.MCPServers)
	if t >= tier.Standard && infraProfile != nil {
		profileFiles, err := infraProfile.ConfigFiles(profile.ProjectInputsFromAnswers(answers))
		if err != nil {
			return nil, fmt.Errorf("generating %s infra profile files: %w", infraProfile.Name, err)
		}
		files = append(files, profileFiles...)
	}

	return files, nil
}

// defaultInfraProfile is the profile whose CI, dependency-update and
// security-overview files are generated when none is selected.
const defaultInfraProfile = "consulting-default"

// applyInfraProfile resolves the infra profile for answers. An explicitly
// selected profile (--infra-profile or .qsdev.yaml infra_profile) is applied
// in full: its registry proxy layout, Nix cache and build cache replace
// answers.Infrastructure with the effective settings and its non-secret
// environment is added under the user's own env vars; a missing or
// placeholder endpoint is an error. The implicit default only contributes
// its config files (ConfigOnly), so projects that never chose an
// infrastructure keep exactly the endpoints they configured; a Nix cache they
// configured is still checked (profile.ResolveProjectInfrastructure). The returned profile is nil when
// the generator has no profile registry.
func (g *DevenvGenerator) applyInfraProfile(answers types.WizardAnswers) (types.WizardAnswers, *profile.InfraProfile, error) {
	if g.profileRegistry == nil {
		return answers, nil, nil
	}
	name := answers.ProfileName
	if name == "" {
		infra, err := profile.ResolveProjectInfrastructure(answers.Infrastructure)
		if err != nil {
			return answers, nil, err
		}
		answers.Infrastructure = infra
		p, ok := g.profileRegistry.Get(defaultInfraProfile)
		if !ok {
			return answers, nil, nil
		}
		return answers, p.ConfigOnly(), nil
	}
	p, ok := g.profileRegistry.Get(name)
	if !ok {
		// An explicit --infra-profile that does not resolve is a user error, not
		// a silent no-op that drops CI/renovate/dependabot configs. Mirror the
		// project --profile hard-error (addons/devinit/commands.go).
		return answers, nil, fmt.Errorf("unknown infra profile %q; use --list-profiles to see available profiles", name)
	}
	resolved, err := p.Resolve(answers.Infrastructure, profile.ProjectInputsFromAnswers(answers))
	if err != nil {
		return answers, nil, err
	}
	answers.Infrastructure = resolved.Infrastructure()
	if env := resolved.EnvironmentVars(); len(env) > 0 {
		merged := make(map[string]string, len(env)+len(answers.EnvVars))
		maps.Copy(merged, env)
		maps.Copy(merged, answers.EnvVars)
		answers.EnvVars = merged
	}
	return answers, resolved, nil
}
