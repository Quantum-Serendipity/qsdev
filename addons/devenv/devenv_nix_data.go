package devenv

import (
	"fmt"
	"maps"
	"sort"
	"strconv"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// DevenvNixTemplateData holds all data required to render the devenv.nix template.
type DevenvNixTemplateData struct {
	Overlays          []string                   // Nix overlay file paths (e.g. "./nix/go-overlay.nix").
	Packages          []string                   // Base + extra packages (rendered as pkgs.NAME).
	PackageExprs      []string                   // Raw Nix expressions that produce derivations.
	EnvVars           map[string]string          // Non-sensitive env vars (always includes DEVENV_SECURITY_HARDENED).
	UnsetEnvVars      []string                   // Credential-bearing vars stripped from the shell.
	LanguageFragments []LanguageFragment         // Pre-rendered Nix from ecosystem modules.
	Services          []ServiceTemplateData      // Structured service configs.
	GitHooksEnabled   bool                       // Whether the git-hooks block appears.
	SecurityHooks     []string                   // Always-present hooks (ripsecrets, etc.).
	BuiltInHooks      []string                   // Ecosystem hooks using .enable = true syntax.
	CustomHooks       []CustomHookData           // Ecosystem hooks needing full attribute sets.
	EnterShell        string                     // Shell script body for enterShell.
	EnterTest         string                     // Test script body for enterTest.
	Tasks             []ecosystem.TaskDefinition // Development task definitions from ecosystem modules.
	ServiceScripts    []ServiceScript            // Convenience scripts from services.
}

// LanguageFragment holds a pre-rendered Nix code block from an ecosystem module.
type LanguageFragment struct {
	DisplayName string // Human-readable name (e.g. "Go", "Python").
	NixFragment string // Raw Nix code from DevenvNixFragment().
}

// ServiceTemplateData holds structured data for rendering a service block in devenv.nix.
type ServiceTemplateData struct {
	DisplayName string            // Human-readable name (e.g. "PostgreSQL").
	NixName     string            // Nix attribute name (e.g. "postgres").
	ConfigLines []string          // Nix attribute lines inside the service block.
	EnvVars     map[string]string // Service-specific env vars merged into the global env block.
	Scripts     []ServiceScript   // Convenience scripts (e.g. open-keycloak).
}

// ServiceScript defines a convenience script emitted as scripts.<Name>.exec in devenv.nix.
type ServiceScript struct {
	Name string // Script name (e.g. "open-keycloak").
	Exec string // Shell command body.
}

// CustomHookData holds all fields needed to render a custom pre-commit hook
// as a full Nix attribute set in devenv.nix.
type CustomHookData struct {
	ID            string
	Name          string
	Description   string
	Entry         string
	RawEntry      bool // When true, Entry is emitted as raw Nix (no double-quoting).
	NeedsToString bool // When true, wrap Entry in toString() (for Nix derivations like writeShellScript).
	Language      string
	Types         []string
	Stages        []string
	Files         string
	PassFilenames bool
}

// languageHookResult holds the collected fragments and hooks from ecosystem modules.
type languageHookResult struct {
	Fragments     []LanguageFragment
	BuiltInHooks  []string
	CustomHooks   []CustomHookData
	ExtraPackages []string
	SeenHookIDs   map[string]bool
}

// BuildDevenvNixData assembles all template data from wizard answers and ecosystem
// modules. It calls into each selected module to collect Nix fragments and hooks,
// then merges them with security defaults.
func BuildDevenvNixData(answers types.WizardAnswers, registry *ecosystem.Registry) (*DevenvNixTemplateData, error) {
	data := &DevenvNixTemplateData{}

	// 0. Overlays from user configuration.
	data.Overlays = answers.Overlays

	// 1. Packages: base + extras.
	basePkgs := defaultBasePackages()
	data.Packages = make([]string, 0, len(basePkgs)+len(answers.ExtraPackages))
	data.Packages = append(data.Packages, basePkgs...)
	data.Packages = append(data.Packages, answers.ExtraPackages...)

	// 2. Environment variables.
	data.EnvVars = buildEnvVars(answers)

	// 3. Unset env vars: credential-bearing variables.
	data.UnsetEnvVars = defaultUnsetEnvVars()

	// 4. Language fragments and hooks from ecosystem modules.
	hookResult, err := collectLanguageFragmentsAndHooks(answers, registry)
	if err != nil {
		return nil, err
	}
	data.LanguageFragments = hookResult.Fragments
	data.BuiltInHooks = hookResult.BuiltInHooks
	data.CustomHooks = hookResult.CustomHooks
	data.Packages = append(data.Packages, hookResult.ExtraPackages...)

	// 4b. Collect packages from modules that implement PackageProvider.
	data.Packages = append(data.Packages, collectModulePackages(answers, registry)...)

	// 4c. Collect packages for enabled tools that need binaries on PATH.
	toolPkgs, toolExprs := collectToolPackages(answers)
	data.Packages = append(data.Packages, toolPkgs...)
	data.PackageExprs = append(data.PackageExprs, toolExprs...)

	// 5. Services.
	for _, svc := range answers.Services {
		svcData, err := serviceToTemplateData(svc)
		if err != nil {
			return nil, fmt.Errorf("configuring service %s: %w", svc.Name, err)
		}
		data.Services = append(data.Services, svcData)
		for k, v := range svcData.EnvVars {
			data.EnvVars[k] = v
		}
		data.ServiceScripts = append(data.ServiceScripts, svcData.Scripts...)
	}

	// 6. Security hooks are always present.
	data.SecurityHooks = defaultSecurityHooks()

	// Specialized security custom hooks (always present), deduped against ecosystem hooks.
	seenHookIDs := hookResult.SeenHookIDs
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
	data.Tasks = collectTaskDefinitions(answers, registry)

	// Sort built-in hooks for deterministic output.
	sort.Strings(data.BuiltInHooks)

	return data, nil
}

// buildEnvVars assembles the environment variable map from wizard answers.
// It always includes the security-hardened flag, context env vars (prefix,
// project name, security profile, version, ecosystems list, tool count),
// and user-supplied env vars.
func buildEnvVars(answers types.WizardAnswers) map[string]string {
	envVars := make(map[string]string, len(answers.EnvVars)+6)
	envVars["DEVENV_SECURITY_HARDENED"] = "true"

	prefix := branding.Get().EnvPrefix
	projectName := answers.ProjectName
	if projectName == "" {
		projectName = "unknown"
	}
	envVars[prefix+"PROJECT_NAME"] = projectName

	securityProfile := answers.ComplianceLevel
	if securityProfile == "" {
		securityProfile = "standard"
	}
	envVars[prefix+"SECURITY_PROFILE"] = securityProfile

	envVars[prefix+"VERSION"] = version.Info().Version
	envVars[prefix+"ECOSYSTEMS"] = buildEcosystemsList(answers)
	envVars[prefix+"TOOL_COUNT"] = strconv.Itoa(countEnabledTools(answers))

	maps.Copy(envVars, answers.EnvVars)
	return envVars
}

// collectLanguageFragmentsAndHooks iterates over selected languages, generates
// their Nix fragments, and collects pre-commit hooks (both built-in and custom).
func collectLanguageFragmentsAndHooks(answers types.WizardAnswers, registry *ecosystem.Registry) (languageHookResult, error) {
	result := languageHookResult{
		SeenHookIDs: make(map[string]bool),
	}

	for _, lang := range answers.Languages {
		mod, ok := registry.ByName(lang.Name)
		if !ok {
			return result, fmt.Errorf("unknown language module: %q", lang.Name)
		}

		cfg := ecosystem.ToModuleConfigWithProxy(lang, answers.Infrastructure)
		fragment, err := mod.DevenvNixFragment(cfg)
		if err != nil {
			return result, fmt.Errorf("generating Nix fragment for %s: %w", lang.Name, err)
		}

		if strings.TrimSpace(fragment) != "" {
			result.Fragments = append(result.Fragments, LanguageFragment{
				DisplayName: mod.DisplayName(),
				NixFragment: fragment,
			})
		}

		for _, hook := range mod.PreCommitHooks(cfg) {
			if result.SeenHookIDs[hook.ID] {
				continue
			}
			result.SeenHookIDs[hook.ID] = true

			if hook.BuiltIn {
				result.BuiltInHooks = append(result.BuiltInHooks, hook.ID)
			} else {
				entry := hook.Entry
				rawEntry := false

				if hook.NixPackage != "" {
					parts := strings.SplitN(hook.Entry, " ", 2)
					binary := parts[0]
					args := ""
					if len(parts) > 1 {
						args = " " + parts[1]
					}
					entry = fmt.Sprintf(`"${pkgs.%s}/bin/%s%s"`, hook.NixPackage, binary, args)
					rawEntry = true
					result.ExtraPackages = append(result.ExtraPackages, hook.NixPackage)
				}

				result.CustomHooks = append(result.CustomHooks, CustomHookData{
					ID:            hook.ID,
					Name:          hook.Name,
					Description:   hook.Description,
					Entry:         entry,
					RawEntry:      rawEntry,
					Language:      hook.Language,
					Types:         hook.Types,
					Stages:        hook.Stages,
					Files:         hook.Files,
					PassFilenames: hook.PassFilenames,
				})
			}
		}
	}

	return result, nil
}

// collectModulePackages gathers Nix packages from ecosystem modules that
// implement the PackageProvider interface.
func collectModulePackages(answers types.WizardAnswers, registry *ecosystem.Registry) []string {
	var pkgs []string
	for _, lang := range answers.Languages {
		mod, ok := registry.ByName(lang.Name)
		if !ok {
			continue
		}
		if pp, ok := mod.(ecosystem.PackageProvider); ok {
			cfg := ecosystem.ToModuleConfigWithProxy(lang, answers.Infrastructure)
			pkgs = append(pkgs, pp.DevenvPackages(cfg)...)
		}
	}
	return pkgs
}

// collectToolPackages returns Nix package names and raw Nix expressions for
// enabled tools that need binaries on PATH.
func collectToolPackages(answers types.WizardAnswers) (pkgs []string, exprs []string) {
	nixPkgs := defaultToolNixPackages()
	nixExprs := defaultToolNixExprs()
	for toolName, enabled := range answers.EnabledTools {
		if !enabled {
			continue
		}
		if nixPkg, ok := nixPkgs[toolName]; ok {
			pkgs = append(pkgs, nixPkg)
		}
		if expr, ok := nixExprs[toolName]; ok {
			exprs = append(exprs, expr)
		}
	}
	return pkgs, exprs
}

// collectTaskDefinitions builds development task definitions from ecosystem
// modules registered in the given registry.
func collectTaskDefinitions(answers types.WizardAnswers, registry *ecosystem.Registry) []ecosystem.TaskDefinition {
	var modules []ecosystem.EcosystemModule
	configForFunc := func(mod ecosystem.EcosystemModule) ecosystem.ModuleConfig {
		for _, lang := range answers.Languages {
			if lang.Name == mod.Name() {
				return ecosystem.ToModuleConfigWithProxy(lang, answers.Infrastructure)
			}
		}
		return ecosystem.ModuleConfig{}
	}
	for _, lang := range answers.Languages {
		if mod, ok := registry.ByName(lang.Name); ok {
			modules = append(modules, mod)
		}
	}
	return ecosystem.AggregateTaskDefinitions(modules, configForFunc, answers.EnabledTools)
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
