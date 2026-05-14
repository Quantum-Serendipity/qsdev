package devenv

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/ecosystem"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/internal/version"
	"github.com/Quantum-Serendipity/gdev-secure-devenv-bootstrap/pkg/types"
)

// DevenvNixTemplateData holds all data required to render the devenv.nix template.
type DevenvNixTemplateData struct {
	Packages          []string              // Base + extra packages.
	EnvVars           map[string]string     // Non-sensitive env vars (always includes DEVENV_SECURITY_HARDENED).
	UnsetEnvVars      []string              // Credential-bearing vars stripped from the shell.
	LanguageFragments []LanguageFragment    // Pre-rendered Nix from ecosystem modules.
	Services          []ServiceTemplateData // Structured service configs.
	GitHooksEnabled   bool                  // Whether the git-hooks block appears.
	SecurityHooks     []string              // Always-present hooks (ripsecrets, etc.).
	BuiltInHooks      []string              // Ecosystem hooks using .enable = true syntax.
	CustomHooks       []CustomHookData      // Ecosystem hooks needing full attribute sets.
	EnterShell        string                // Shell script body for enterShell.
	EnterTest         string                // Test script body for enterTest.
	Tasks             []ecosystem.TaskDefinition // Development task definitions from ecosystem modules.
}

// LanguageFragment holds a pre-rendered Nix code block from an ecosystem module.
type LanguageFragment struct {
	DisplayName string // Human-readable name (e.g. "Go", "Python").
	NixFragment string // Raw Nix code from DevenvNixFragment().
}

// ServiceTemplateData holds structured data for rendering a service block in devenv.nix.
type ServiceTemplateData struct {
	DisplayName string   // Human-readable name (e.g. "PostgreSQL").
	NixName     string   // Nix attribute name (e.g. "postgres").
	ConfigLines []string // Nix attribute lines inside the service block.
}

// CustomHookData holds all fields needed to render a custom pre-commit hook
// as a full Nix attribute set in devenv.nix.
type CustomHookData struct {
	ID            string
	Name          string
	Description   string
	Entry         string
	RawEntry      bool // When true, Entry is emitted as raw Nix (no double-quoting).
	Language      string
	Types         []string
	Stages        []string
	Files         string
	PassFilenames bool
}

// BuildDevenvNixData assembles all template data from wizard answers and ecosystem
// modules. It calls into each selected module to collect Nix fragments and hooks,
// then merges them with security defaults.
func BuildDevenvNixData(answers types.WizardAnswers, registry *ecosystem.Registry) (*DevenvNixTemplateData, error) {
	data := &DevenvNixTemplateData{}

	// 1. Packages: base + extras.
	data.Packages = make([]string, 0, len(defaultBasePackages)+len(answers.ExtraPackages))
	data.Packages = append(data.Packages, defaultBasePackages...)
	data.Packages = append(data.Packages, answers.ExtraPackages...)

	// 2. Environment variables: always include the security-hardened flag.
	data.EnvVars = make(map[string]string, len(answers.EnvVars)+6)
	data.EnvVars["DEVENV_SECURITY_HARDENED"] = "true"

	// gdev context environment variables.
	projectName := answers.ProjectName
	if projectName == "" {
		projectName = "unknown"
	}
	data.EnvVars["GDEV_PROJECT_NAME"] = projectName

	securityProfile := answers.ComplianceLevel
	if securityProfile == "" {
		securityProfile = "standard"
	}
	data.EnvVars["GDEV_SECURITY_PROFILE"] = securityProfile

	data.EnvVars["GDEV_VERSION"] = version.Info().Version
	data.EnvVars["GDEV_ECOSYSTEMS"] = buildEcosystemsList(answers)
	data.EnvVars["GDEV_TOOL_COUNT"] = strconv.Itoa(countEnabledTools(answers))

	for k, v := range answers.EnvVars {
		data.EnvVars[k] = v
	}

	// 3. Unset env vars: credential-bearing variables.
	data.UnsetEnvVars = make([]string, len(defaultUnsetEnvVars))
	copy(data.UnsetEnvVars, defaultUnsetEnvVars)

	// 4. Language fragments from ecosystem modules.
	seenHookIDs := make(map[string]bool)
	for _, lang := range answers.Languages {
		mod, ok := registry.ByName(lang.Name)
		if !ok {
			return nil, fmt.Errorf("unknown language module: %q", lang.Name)
		}

		cfg := toModuleConfig(lang)
		fragment, err := mod.DevenvNixFragment(cfg)
		if err != nil {
			return nil, fmt.Errorf("generating Nix fragment for %s: %w", lang.Name, err)
		}

		data.LanguageFragments = append(data.LanguageFragments, LanguageFragment{
			DisplayName: mod.DisplayName(),
			NixFragment: fragment,
		})

		// Collect hooks from each module.
		for _, hook := range mod.PreCommitHooks(cfg) {
			if seenHookIDs[hook.ID] {
				continue
			}
			seenHookIDs[hook.ID] = true

			if hook.BuiltIn {
				data.BuiltInHooks = append(data.BuiltInHooks, hook.ID)
			} else {
				data.CustomHooks = append(data.CustomHooks, CustomHookData{
					ID:            hook.ID,
					Name:          hook.Name,
					Description:   hook.Description,
					Entry:         hook.Entry,
					Language:      hook.Language,
					Types:         hook.Types,
					Stages:        hook.Stages,
					Files:         hook.Files,
					PassFilenames: hook.PassFilenames,
				})
			}
		}
	}

	// 5. Services.
	for _, svc := range answers.Services {
		svcData, err := serviceToTemplateData(svc)
		if err != nil {
			return nil, fmt.Errorf("configuring service %s: %w", svc.Name, err)
		}
		data.Services = append(data.Services, svcData)
	}

	// 6. Security hooks are always present.
	data.SecurityHooks = make([]string, len(defaultSecurityHooks))
	copy(data.SecurityHooks, defaultSecurityHooks)

	// Specialized security custom hooks (always present).
	for _, hook := range defaultSpecializedHooks() {
		if !seenHookIDs[hook.ID] {
			seenHookIDs[hook.ID] = true
			data.CustomHooks = append(data.CustomHooks, hook)
		}
	}

	// 7. Git hooks are always enabled (security hooks are mandatory).
	data.GitHooksEnabled = true

	// 8. Shell scripts.
	data.EnterShell = buildEnterShellScript()
	data.EnterTest = buildEnterTestScript()

	// 9. Task definitions from ecosystem modules.
	var modules []ecosystem.EcosystemModule
	configForFunc := func(mod ecosystem.EcosystemModule) ecosystem.ModuleConfig {
		for _, lang := range answers.Languages {
			if lang.Name == mod.Name() {
				return toModuleConfig(lang)
			}
		}
		return ecosystem.ModuleConfig{}
	}
	for _, lang := range answers.Languages {
		if mod, ok := registry.ByName(lang.Name); ok {
			modules = append(modules, mod)
		}
	}
	data.Tasks = ecosystem.AggregateTaskDefinitions(modules, configForFunc, answers.EnabledTools)

	// Sort built-in hooks for deterministic output.
	sort.Strings(data.BuiltInHooks)

	return data, nil
}

// buildEcosystemsList returns a sorted, comma-separated list of selected
// ecosystem names. Returns "none" when no languages are selected.
func buildEcosystemsList(answers types.WizardAnswers) string {
	if len(answers.Languages) == 0 {
		return "none"
	}
	names := make([]string, 0, len(answers.Languages))
	for _, lang := range answers.Languages {
		names = append(names, lang.Name)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}

// countEnabledTools returns the number of tools that are explicitly enabled
// in the wizard answers.
func countEnabledTools(answers types.WizardAnswers) int {
	count := 0
	for _, enabled := range answers.EnabledTools {
		if enabled {
			count++
		}
	}
	return count
}
