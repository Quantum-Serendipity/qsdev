package devenv

import (
	"fmt"
	"maps"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/internal/version"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/lsp"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// DevenvNixTemplateData holds all data required to render the devenv.nix template.
type DevenvNixTemplateData struct {
	Overlays           []string              // Nix path expressions for overlay files (e.g. "./nix/go-overlay.nix").
	Packages           []string              // Base + extra packages (rendered as pkgs.NAME).
	PackageExprs       []string              // Raw Nix expressions that produce derivations.
	EnvVars            map[string]string     // Non-sensitive env vars (always includes DEVENV_SECURITY_HARDENED).
	UnsetEnvVars       []string              // Credential-bearing vars stripped from the shell.
	CachixPull         []string              // Project Cachix caches (infrastructure.nix_cache).
	LanguageFragments  []LanguageFragment    // Pre-rendered Nix from ecosystem modules.
	Services           []ServiceTemplateData // Structured service configs.
	GitHooksEnabled    bool                  // Whether the git-hooks block appears.
	SecurityHooks      []string              // Always-present hooks (ripsecrets, etc.).
	BuiltInHooks       []BuiltInHookData     // Ecosystem hooks provided by git-hooks.nix.
	HookOverrides      []HookOverrideData    // File-type scoping for always-on security hooks.
	CustomHooks        []CustomHookData      // Ecosystem hooks needing full attribute sets.
	NeedsNativeLibPath bool                  // True when uv-tool MCP servers need LD_LIBRARY_PATH (NixOS).
	EnterShell         string                // Shell script body for enterShell.
	EnterTest          string                // Test script body for enterTest.
	TaskScripts        []TaskScript          // Development tasks rendered as devenv scripts.
	ServiceScripts     []ServiceScript       // Convenience scripts from services.
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

// TaskScript is a development task (build, test, lint, ...) emitted as a
// devenv script so it is an executable on PATH, which works in direnv-activated
// shells too (direnv exports variables only, never shell functions).
type TaskScript struct {
	Name        string // Script name (e.g. "qsdev-test").
	Description string // Human-readable description.
	Exec        string // Bash body; runs with errexit so any failing command fails the task.
}

// ServiceScript defines a convenience script emitted as scripts.<Name>.exec in devenv.nix.
type ServiceScript struct {
	Name string // Script name (e.g. "open-keycloak").
	Exec string // Shell command body.
}

// BuiltInHookData is an ecosystem hook provided by git-hooks.nix. Its entry is
// defined by git-hooks.nix; a hook with no TypesOr, ExcludeTypes, Excludes or
// Settings renders as `<id>.enable = true;`.
type BuiltInHookData struct {
	ID           string
	TypesOr      []string
	ExcludeTypes []string
	Excludes     []string      // Path regexes the hook skips (git-hooks.nix excludes).
	Settings     []HookSetting // Sorted by Key.
}

// HookSetting is one git-hooks.nix `settings.<Key> = "<Value>";` option.
type HookSetting struct {
	Key   string
	Value string
}

// hookSettingKeyRe restricts setting keys to plain Nix attribute names, since
// keys are rendered unquoted.
var hookSettingKeyRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

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
	TypesOr       []string
	ExcludeTypes  []string
	Stages        []string
	Files         string
	Excludes      []string
	PassFilenames bool
	// Package is the nixpkgs attribute rendered as the hook's `package`. A
	// custom hook whose ID matches a git-hooks.nix built-in merges with that
	// definition, whose default package would otherwise be evaluated (and may
	// no longer exist in nixpkgs) even though Entry names another binary.
	Package string
}

// HookOverrideData scopes an enabled git-hooks.nix built-in hook: the listed
// fields replace upstream's (mkDefault) values.
type HookOverrideData struct {
	ID           string
	TypesOr      []string
	ExcludeTypes []string
}

// languageHookResult holds the collected fragments and hooks from ecosystem modules.
type languageHookResult struct {
	Fragments     []LanguageFragment
	BuiltInHooks  []BuiltInHookData
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
	overlays, err := overlayPathExprs(answers.Overlays)
	if err != nil {
		return nil, err
	}
	data.Overlays = overlays

	// 1. Packages: base + extras. Extras come from answers files and config
	// that can be edited outside the add-package command, so re-validate them
	// here before they are rendered verbatim into devenv.nix.
	for _, name := range answers.ExtraPackages {
		if err := validateNixPackageName(name); err != nil {
			return nil, fmt.Errorf("extra packages: %w", err)
		}
	}
	basePkgs := defaultBasePackages()
	data.Packages = make([]string, 0, len(basePkgs)+len(answers.ExtraPackages))
	data.Packages = append(data.Packages, basePkgs...)
	data.Packages = append(data.Packages, answers.ExtraPackages...)

	// 2. Environment variables.
	data.EnvVars = buildEnvVars(answers)

	// 2b. LSP enforcement tier read by the lsp-first-guard PreToolUse hook.
	// buildEnvVars always returns a non-nil map; guard anyway for safety.
	if data.EnvVars == nil {
		data.EnvVars = make(map[string]string, 1)
	}
	data.EnvVars["QSDEV_LSP_ENFORCEMENT"] = answers.LSP.EnforcementTier()

	// 3. Unset env vars: credential-bearing variables.
	data.UnsetEnvVars = defaultUnsetEnvVars()

	// 3b. The project's Cachix binary cache, from the effective infrastructure
	// (an explicit infra profile's nix cache, or infrastructure.nix_cache).
	data.CachixPull = cachixPullCaches(answers.Infrastructure)

	// 4. Language fragments and hooks from ecosystem modules.
	hookResult, err := collectLanguageFragmentsAndHooks(answers, registry)
	if err != nil {
		return nil, err
	}
	data.LanguageFragments = hookResult.Fragments
	data.BuiltInHooks = hookResult.BuiltInHooks
	data.CustomHooks = hookResult.CustomHooks
	data.Packages = append(data.Packages, hookResult.ExtraPackages...)

	// 4b. Collect packages from modules that implement PackageProvider and
	// package expressions from modules that implement PackageExprProvider.
	modPkgs, modExprs := collectModulePackages(answers, registry)
	data.Packages = append(data.Packages, modPkgs...)
	data.PackageExprs = append(data.PackageExprs, modExprs...)

	// 4c. Collect packages for enabled tools that need binaries on PATH.
	toolPkgs, toolExprs := collectToolPackages(answers)
	data.Packages = append(data.Packages, toolPkgs...)
	data.PackageExprs = append(data.PackageExprs, toolExprs...)

	// 4d. MCP server runtime dependencies (e.g. pkgs.uv for semble's uvx).
	mcpPkgs := collectMCPPackages(answers)
	data.Packages = append(data.Packages, mcpPkgs...)
	data.NeedsNativeLibPath = needsNativeLibPath(answers)

	// 4e. LSP servers: a single, registry-driven section that emits explicit
	// enable/disable lines for every detected ecosystem (plus always-on nixd).
	// Analyzer config lives in the generated .lsp.json, not devenv.
	collectLSPSection(answers, data)

	// 4f. Sections enabled tools contribute to devenv.nix (e.g. the starship
	// env var, commit-ticket/branch-naming hooks). Rendering them here makes
	// the generator their single source: init, update and `enable` all emit
	// them, so a routine update no longer drops a tool that is still enabled.
	toolFragments, err := collectToolNixSections(answers)
	if err != nil {
		return nil, err
	}
	data.LanguageFragments = append(data.LanguageFragments, toolFragments...)

	// 5. Services.
	serviceEnv := make(map[string]bool)
	for _, svc := range answers.Services {
		svcData, err := serviceToTemplateData(svc)
		if err != nil {
			return nil, fmt.Errorf("configuring service %s: %w", svc.Name, err)
		}
		data.Services = append(data.Services, svcData)
		for k, v := range svcData.EnvVars {
			data.EnvVars[k] = v
			serviceEnv[k] = true
		}
		data.ServiceScripts = append(data.ServiceScripts, svcData.Scripts...)
	}

	// A variable a service sets on purpose (e.g. MinIO's local AWS_* client
	// credentials) must not also be unset: devenv unsets after exporting env,
	// so the service's value would silently disappear from the shell.
	data.UnsetEnvVars = slices.DeleteFunc(data.UnsetEnvVars, func(name string) bool {
		return serviceEnv[name]
	})

	if err := validateEnvVarNames(data.EnvVars); err != nil {
		return nil, err
	}

	// A variable set explicitly (--env, .qsdev.yaml env:, or a service)
	// overrides the default an ecosystem module or tool writes as env.NAME;
	// keeping both would define env.NAME twice.
	if err := dropOverriddenEnv(data.LanguageFragments, data.EnvVars); err != nil {
		return nil, err
	}

	// 6. Security hooks are always present. An ecosystem module may declare
	// the same hook (shellcheck for shell, statix for nix); rendering both
	// defines one git-hooks attribute twice and devenv.nix fails to evaluate,
	// so the always-on security entry wins.
	data.SecurityHooks = defaultSecurityHooks()
	seenHookIDs := hookResult.SeenHookIDs
	securityHookIDs := make(map[string]bool, len(data.SecurityHooks))
	for _, id := range data.SecurityHooks {
		securityHookIDs[id] = true
		seenHookIDs[id] = true
	}
	data.CustomHooks = slices.DeleteFunc(data.CustomHooks, func(h CustomHookData) bool { return securityHookIDs[h.ID] })

	// A selected module's scoping for a security hook survives as an
	// override of the always-on entry; the registry supplies the rest.
	overridden := make(map[string]bool, len(data.SecurityHooks))
	for _, h := range data.BuiltInHooks {
		if !securityHookIDs[h.ID] || (len(h.TypesOr) == 0 && len(h.ExcludeTypes) == 0) {
			continue
		}
		overridden[h.ID] = true
		data.HookOverrides = append(data.HookOverrides, HookOverrideData{ID: h.ID, TypesOr: h.TypesOr, ExcludeTypes: h.ExcludeTypes})
	}
	data.HookOverrides = append(data.HookOverrides, securityHookOverrides(registry, data.SecurityHooks, overridden)...)
	data.BuiltInHooks = slices.DeleteFunc(data.BuiltInHooks, func(h BuiltInHookData) bool { return securityHookIDs[h.ID] })

	// Specialized security custom hooks (always present), deduped against ecosystem hooks.
	for _, hook := range defaultSpecializedHooks(projectLockFiles(answers.Languages)) {
		if !seenHookIDs[hook.ID] {
			seenHookIDs[hook.ID] = true
			data.CustomHooks = append(data.CustomHooks, hook)
		}
	}

	// 7. Git hooks are always enabled (security hooks are mandatory).
	data.GitHooksEnabled = true

	// 8. Shell scripts.
	data.EnterShell = buildEnterShellScript(data.UnsetEnvVars)
	data.EnterTest = buildEnterTestScript(data.UnsetEnvVars)

	// 9. Task definitions from ecosystem modules.
	data.TaskScripts = buildTaskScripts(collectTaskDefinitions(answers, registry))

	// Sort built-in hooks for deterministic output.
	slices.SortFunc(data.BuiltInHooks, func(a, b BuiltInHookData) int { return strings.Compare(a.ID, b.ID) })

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

// dropOverriddenEnv removes env.NAME bindings from fragments for every NAME
// in env, which is rendered in the env block and takes precedence.
func dropOverriddenEnv(fragments []LanguageFragment, env map[string]string) error {
	overridden := func(path []string) bool {
		if len(path) != 2 || path[0] != "env" {
			return false
		}
		_, ok := env[path[1]]
		return ok
	}
	for i := range fragments {
		out, err := dropNixBindings(fragments[i].NixFragment, overridden)
		if err != nil {
			return fmt.Errorf("applying env overrides to the %s section: %w", fragments[i].DisplayName, err)
		}
		fragments[i].NixFragment = out
	}
	return nil
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

		cfg := ecosystem.ToModuleConfigWithInfra(lang, answers.Infrastructure)
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
				builtIn, err := builtInHookData(hook)
				if err != nil {
					return result, fmt.Errorf("pre-commit hook for %s: %w", lang.Name, err)
				}
				result.BuiltInHooks = append(result.BuiltInHooks, builtIn)
			} else {
				entry := hook.Entry
				rawEntry, needsToString := false, false

				switch {
				case hook.Script != "":
					entry = scriptHookEntry(hook)
					rawEntry, needsToString = true, true
					if hook.NixPackage != "" {
						result.ExtraPackages = append(result.ExtraPackages, hook.NixPackage)
					}
				case hook.NixPackage != "":
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
					NeedsToString: needsToString,
					Language:      hook.Language,
					Types:         hook.Types,
					TypesOr:       hook.TypesOr,
					ExcludeTypes:  hook.ExcludeTypes,
					Stages:        hook.Stages,
					Files:         hook.Files,
					Excludes:      hook.Excludes,
					PassFilenames: hook.PassFilenames,
					Package:       hook.NixPackage,
				})
			}
		}
	}

	return result, nil
}

// scriptHookEntry renders a module hook's Script as a pkgs.writeShellScript
// derivation. The script is plain bash: it is escaped for the Nix indented
// string, so shell antiquotes and quote pairs reach bash unchanged. NixPackage's bin
// directory is prepended to PATH so the script runs the pinned tool.
func scriptHookEntry(hook ecosystem.HookConfig) string {
	body := strings.TrimSpace(hook.Script)
	body = strings.ReplaceAll(body, "''", "'''")
	body = strings.ReplaceAll(body, "${", "''${")

	var b strings.Builder
	fmt.Fprintf(&b, "pkgs.writeShellScript %s ''\n", nixStr(hook.ID))
	if hook.NixPackage != "" {
		fmt.Fprintf(&b, "        export PATH=${pkgs.%s}/bin:$PATH\n", hook.NixPackage)
	}
	b.WriteString(indentBlock(body, "        "))
	b.WriteString("\n      ''")
	return b.String()
}

// hookOverride returns the file-type scoping a built-in hook declares, if any.
func hookOverride(hook ecosystem.HookConfig) (HookOverrideData, bool) {
	if len(hook.TypesOr) == 0 && len(hook.ExcludeTypes) == 0 {
		return HookOverrideData{}, false
	}
	return HookOverrideData{ID: hook.ID, TypesOr: hook.TypesOr, ExcludeTypes: hook.ExcludeTypes}, true
}

// securityHookOverrides returns the scoping for always-on security hooks that
// no selected module supplied. The module that declares the same git-hooks.nix
// built-in (shell for shellcheck) is the authority on how it must be scoped,
// whether or not that ecosystem is selected, so the whole registry is
// consulted. have lists the IDs already overridden.
func securityHookOverrides(registry *ecosystem.Registry, securityIDs []string, have map[string]bool) []HookOverrideData {
	if registry == nil {
		return nil
	}
	want := make(map[string]bool, len(securityIDs))
	for _, id := range securityIDs {
		if !have[id] {
			want[id] = true
		}
	}
	var out []HookOverrideData
	for _, mod := range registry.All() {
		for _, hook := range mod.PreCommitHooks(ecosystem.ModuleConfig{}) {
			if !hook.BuiltIn || !want[hook.ID] {
				continue
			}
			if override, ok := hookOverride(hook); ok {
				out = append(out, override)
				want[hook.ID] = false
			}
		}
	}
	return out
}

// lspDisplayName is the human-readable label for the synthetic LSP language
// fragment rendered into devenv.nix.
const lspDisplayName = "LSP servers (qsdev-managed)"

// collectLSPSection builds the centralized, registry-driven LSP section and
// merges it into data. It emits explicit enable/disable lines (analyzer config
// is deferred to the generated .lsp.json), adds package-list servers to
// data.Packages, and always provisions nixd regardless of selected languages.
//
// Emitting LSP lines from one place is equivalent to touching every ecosystem
// module: Nix's module system merges separate "languages.X = {...}" and
// "languages.X.lsp.enable = true" definitions.
func collectLSPSection(answers types.WizardAnswers, data *DevenvNixTemplateData) {
	reg := lsp.NewRegistry()

	var b strings.Builder
	nixSelected := false

	for _, lang := range answers.Languages {
		if lang.Name == "nix" {
			nixSelected = true
		}

		cfg, ok := reg.ByEcosystem(lang.Name)
		if !ok {
			continue
		}

		// Servers with no devenv lsp option get their binary added to the
		// packages list — but only when the server is default-on. Default-off
		// package-list servers (e.g. ansible) are opt-in and never auto-added.
		if pkg, isList := lsp.PackageListPackage(cfg); isList {
			if cfg.DefaultOn {
				data.Packages = append(data.Packages, pkg)
			}
			continue
		}

		// DevenvEnable / DevenvEnableOverridePackage / DevenvDisable emit lines;
		// DevenvSDKBundled returns "" (server ships in the base SDK package).
		b.WriteString(lsp.NixLSPFragment(cfg))
	}

	// nixd is always-on. devenv guards languages.nix.lsp.package behind
	// `lib.mkIf languages.nix.enable`, so the enable line is required to make
	// nixd available; the lsp line is emitted by the loop above when nix is a
	// selected language, otherwise we emit both here. Both merge harmlessly with
	// the nixlang module's own "languages.nix.enable = true;".
	if !nixSelected {
		if nixCfg, ok := reg.ByEcosystem("nix"); ok {
			b.WriteString(nixIndentLine("languages.nix.enable = true;"))
			b.WriteString(lsp.NixLSPFragment(nixCfg))
		}
	}

	if b.Len() == 0 {
		return
	}

	data.LanguageFragments = append(data.LanguageFragments, LanguageFragment{
		DisplayName: lspDisplayName,
		NixFragment: b.String(),
	})
}

// collectToolNixSections renders the devenv.nix section of every enabled tool
// that declares one, in registry order, as fragments labelled with the tool's
// display name.
func collectToolNixSections(answers types.WizardAnswers) ([]LanguageFragment, error) {
	sections, err := toolreg.DefaultRegistry().SharedSectionsFor(toolreg.DevenvNixFile, answers)
	if err != nil {
		return nil, fmt.Errorf("collecting tool devenv.nix sections: %w", err)
	}
	fragments := make([]LanguageFragment, 0, len(sections))
	for _, s := range sections {
		if strings.TrimSpace(string(s.Content)) == "" {
			continue
		}
		fragments = append(fragments, LanguageFragment{
			DisplayName: s.Tool.DisplayName,
			NixFragment: string(s.Content),
		})
	}
	return fragments, nil
}

// nixIndentLine returns line with the 2-space devenv.nix indentation and a
// trailing newline, matching the style emitted by pkg/lsp.NixLSPFragment.
func nixIndentLine(line string) string {
	return "  " + line + "\n"
}

// collectModulePackages gathers Nix package names from ecosystem modules that
// implement the PackageProvider interface and raw Nix package expressions from
// modules that implement the PackageExprProvider interface.
func collectModulePackages(answers types.WizardAnswers, registry *ecosystem.Registry) (pkgs []string, exprs []string) {
	for _, lang := range answers.Languages {
		mod, ok := registry.ByName(lang.Name)
		if !ok {
			continue
		}
		cfg := ecosystem.ToModuleConfigWithInfra(lang, answers.Infrastructure)
		if pp, ok := mod.(ecosystem.PackageProvider); ok {
			pkgs = append(pkgs, pp.DevenvPackages(cfg)...)
		}
		if ep, ok := mod.(ecosystem.PackageExprProvider); ok {
			exprs = append(exprs, ep.DevenvPackageExprs(cfg)...)
		}
	}
	return pkgs, exprs
}

// collectToolPackages returns Nix package names and raw Nix expressions for
// enabled tools that need binaries on PATH.
func collectToolPackages(answers types.WizardAnswers) (pkgs []string, exprs []string) {
	nixPkgs := defaultToolNixPackages()
	nixExprs := defaultToolNixExprs()
	// Iterate in sorted order so regenerating with identical answers yields a
	// byte-identical devenv.nix (map order would reshuffle the package list).
	for _, toolName := range slices.Sorted(maps.Keys(answers.EnabledTools)) {
		if !answers.EnabledTools[toolName] {
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

// mcpServerNixDeps maps MCP server install methods to the Nix packages
// needed at runtime (e.g. uvx comes from the uv package).
var mcpServerNixDeps = map[string]string{
	"uv-tool": "uv",
}

// collectMCPPackages returns Nix packages required by the selected MCP servers.
func collectMCPPackages(answers types.WizardAnswers) []string {
	cat, err := catalog.Default()
	if err != nil {
		return nil
	}

	seen := make(map[string]bool)
	var pkgs []string
	for _, name := range answers.ConfiguredMCPServers() {
		def, ok := cat.MCPServer(name)
		if !ok {
			continue
		}
		if nixPkg, ok := mcpServerNixDeps[def.InstallMethod]; ok && !seen[nixPkg] {
			seen[nixPkg] = true
			pkgs = append(pkgs, nixPkg)
		}
		if def.NixPackage != "" && !seen[def.NixPackage] {
			seen[def.NixPackage] = true
			pkgs = append(pkgs, def.NixPackage)
		}
	}
	return pkgs
}

// needsNativeLibPath returns true when any selected MCP server uses uv-tool
// install method. On NixOS, Python packages with native C extensions (numpy)
// need LD_LIBRARY_PATH to find libstdc++.
func needsNativeLibPath(answers types.WizardAnswers) bool {
	cat, err := catalog.Default()
	if err != nil {
		return false
	}
	for _, name := range answers.ConfiguredMCPServers() {
		def, ok := cat.MCPServer(name)
		if !ok {
			continue
		}
		if def.InstallMethod == "uv-tool" {
			return true
		}
	}
	return false
}

// collectTaskDefinitions builds development task definitions from ecosystem
// modules registered in the given registry.
func collectTaskDefinitions(answers types.WizardAnswers, registry *ecosystem.Registry) []ecosystem.TaskDefinition {
	var modules []ecosystem.EcosystemModule
	configForFunc := func(mod ecosystem.EcosystemModule) ecosystem.ModuleConfig {
		for _, lang := range answers.Languages {
			if lang.Name == mod.Name() {
				return ecosystem.ToModuleConfigWithInfra(lang, answers.Infrastructure)
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

// taskScriptPrefix is prepended to task names to form script names. It must
// match the names the generated CLAUDE.md advertises, and posture reads the
// security-scan script by this name.
const taskScriptPrefix = ecosystem.TaskScriptPrefix

// buildTaskScripts turns task definitions into devenv scripts. Each script
// runs its commands under errexit, so a failing command fails the task instead
// of being masked by a later passing one, and first runs the scripts of the
// tasks it depends on (e.g. test runs build).
func buildTaskScripts(tasks []ecosystem.TaskDefinition) []TaskScript {
	defined := make(map[string]bool, len(tasks))
	for _, t := range tasks {
		defined[t.Name] = true
	}

	scripts := make([]TaskScript, 0, len(tasks))
	for _, t := range tasks {
		lines := []string{"set -euo pipefail"}
		for _, dep := range t.DependsOn {
			if defined[dep] && dep != t.Name {
				lines = append(lines, taskScriptPrefix+dep)
			}
		}
		lines = append(lines, t.Commands...)
		scripts = append(scripts, TaskScript{
			Name:        taskScriptPrefix + t.Name,
			Description: t.Description,
			Exec:        strings.Join(lines, "\n"),
		})
	}
	return scripts
}

// envVarNameRe matches names that are valid both as shell environment
// variables and as bare Nix attribute names in the env block.
var envVarNameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// validateEnvVarNames rejects env keys that would not render as a single Nix
// attribute: "API.URL" would become a nested attrset and "MY VAR" or "a-b"
// would not name a usable shell variable.
func validateEnvVarNames(env map[string]string) error {
	for _, name := range slices.Sorted(maps.Keys(env)) {
		if !envVarNameRe.MatchString(name) {
			return fmt.Errorf("invalid environment variable name %q: must match %s", name, envVarNameRe)
		}
	}
	return nil
}

// nixPathLiteralRe matches relative paths that can be written as a bare Nix
// path literal once prefixed with "./".
var nixPathLiteralRe = regexp.MustCompile(`^[A-Za-z0-9._+-]+(/[A-Za-z0-9._+-]+)*$`)

// overlayPathExprs normalizes project-relative overlay file paths and renders
// each as a Nix path expression rooted at the project (devenv.nix's directory).
// Absolute paths and paths escaping the project are rejected: they make
// devenv.nix non-portable and fail under pure evaluation.
func overlayPathExprs(overlays []string) ([]string, error) {
	exprs := make([]string, 0, len(overlays))
	for _, o := range overlays {
		slashed := filepath.ToSlash(o)
		if o == "" || path.IsAbs(slashed) || filepath.IsAbs(o) || filepath.VolumeName(o) != "" {
			return nil, fmt.Errorf("overlay %q: must be a path relative to the project root", o)
		}
		rel := path.Clean(slashed)
		if rel == "." || rel == ".." || strings.HasPrefix(rel, "../") {
			return nil, fmt.Errorf("overlay %q: must be a file inside the project", o)
		}
		if nixPathLiteralRe.MatchString(rel) {
			exprs = append(exprs, "./"+rel)
		} else {
			exprs = append(exprs, "(./. + "+nixStr("/"+rel)+")")
		}
	}
	return exprs, nil
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

// builtInHookData converts a BuiltIn ecosystem hook into template data,
// carrying the options git-hooks.nix exposes for its built-in hooks.
func builtInHookData(hook ecosystem.HookConfig) (BuiltInHookData, error) {
	data := BuiltInHookData{ID: hook.ID, TypesOr: hook.TypesOr, ExcludeTypes: hook.ExcludeTypes, Excludes: hook.Excludes}
	for _, key := range slices.Sorted(maps.Keys(hook.Settings)) {
		if !hookSettingKeyRe.MatchString(key) {
			return BuiltInHookData{}, fmt.Errorf("hook %q: invalid setting name %q", hook.ID, key)
		}
		data.Settings = append(data.Settings, HookSetting{Key: key, Value: hook.Settings[key]})
	}
	return data, nil
}
