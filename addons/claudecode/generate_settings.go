package claudecode

import (
	"encoding/json"
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
	"github.com/Quantum-Serendipity/qsdev/internal/sliceutil"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// SettingsJSON is the top-level structure that marshals to .claude/settings.json.
type SettingsJSON struct {
	Permissions Permissions              `json:"permissions"`
	Sandbox     *SandboxConfig           `json:"sandbox,omitempty"`
	Hooks       map[string][]HookMatcher `json:"hooks,omitempty"`
}

// Permissions defines the permission rules for Claude Code.
type Permissions struct {
	DefaultMode                  string   `json:"defaultMode,omitempty"`
	DisableBypassPermissionsMode string   `json:"disableBypassPermissionsMode,omitempty"`
	Allow                        []string `json:"allow"`
	Deny                         []string `json:"deny"`
	Ask                          []string `json:"ask,omitempty"`
}

// SandboxConfig defines filesystem and network sandbox restrictions.
type SandboxConfig struct {
	WriteDeny  []string `json:"writeDeny,omitempty"`
	WriteAllow []string `json:"writeAllow,omitempty"`
	ReadDeny   []string `json:"readDeny,omitempty"`
	NetAllow   []string `json:"netAllow,omitempty"`
}

// HookMatcher defines a matcher and its associated hooks within a hook event.
type HookMatcher struct {
	Matcher string      `json:"matcher"`
	Hooks   []HookEntry `json:"hooks"`
}

// HookEntry defines a single hook command within a HookMatcher.
type HookEntry struct {
	Type          string `json:"type"`
	Command       string `json:"command"`
	Timeout       int    `json:"timeout,omitempty"`
	StatusMessage string `json:"statusMessage,omitempty"`
}

// AllBaseDenyRules returns the full deny rule list. Exported for use by
// the check command to verify deny rule coverage.
func AllBaseDenyRules() []string {
	return catalog.MustDefault().AllPermissionDenyRules()
}

// ---------------------------------------------------------------------------
// Permission preset builder
// ---------------------------------------------------------------------------

// buildPermissions constructs the Permissions struct based on the selected
// permission preset, wizard answers, ecosystem registry, and addon config.
func buildPermissions(preset PermissionPreset, answers types.WizardAnswers, registry *ecosystem.Registry, cfg Config) Permissions {
	cat := catalog.MustDefault()
	ecosystemDeny := collectEcosystemDenyRules(answers, registry)

	presetName := string(preset)
	presetDef, ok := cat.PermissionPreset(presetName)
	if !ok {
		// Fall back to standard if preset not found.
		presetDef, _ = cat.PermissionPreset("standard")
	}

	// Assemble allow rules from preset's allow sets.
	var allow []string
	for _, setName := range presetDef.AllowSets {
		allow = append(allow, cat.PermissionAllowRules(setName)...)
	}

	// Assemble deny rules from preset's deny sets.
	var deny []string
	for _, setName := range presetDef.DenySets {
		deny = append(deny, cat.PermissionDenyRules(setName)...)
	}

	// Assemble ask rules from preset's ask sets.
	var ask []string
	for _, setName := range presetDef.AskSets {
		ask = append(ask, cat.PermissionAskRules(setName)...)
	}

	// Always add ecosystem deny rules.
	deny = append(deny, ecosystemDeny...)

	switch preset {
	case PermissionPresetPermissive:
		// Add Podman commands to permissive allow list when Podman is detected.
		if answers.Detected.ContainerRuntime == "podman-rootless" || answers.Detected.ContainerRuntime == "podman-rootful" {
			allow = append(allow, `Bash(podman *)`)
		}

	case PermissionPresetSupplyChainOnly:
		// Supply-chain-only returns early with no defaultMode/disableBypass.
		return Permissions{
			Allow: []string{},
			Deny:  sliceutil.Dedup(deny),
			Ask:   sliceutil.Dedup(ask),
		}

	case PermissionPresetCustom:
		// Custom: allow only what's in ExtraAllowPatterns (not preset sets).
		allow = cfg.ExtraAllowPatterns
		deny = append(deny, cfg.ExtraDenyPatterns...)
		return Permissions{
			Allow: sliceutil.Dedup(allow),
			Deny:  sliceutil.Dedup(deny),
			Ask:   sliceutil.Dedup(ask),
		}
	}

	// For non-custom, non-supply-chain-only presets, append extra patterns from config.
	allow = append(allow, cfg.ExtraAllowPatterns...)
	deny = append(deny, cfg.ExtraDenyPatterns...)

	perms := Permissions{
		Allow: sliceutil.Dedup(allow),
		Deny:  sliceutil.Dedup(deny),
	}
	if len(ask) > 0 {
		perms.Ask = sliceutil.Dedup(ask)
	}
	if presetDef.DefaultMode != "" {
		perms.DefaultMode = presetDef.DefaultMode
	}
	if presetDef.DisableBypassMode != "" {
		perms.DisableBypassPermissionsMode = presetDef.DisableBypassMode
	}
	return perms
}

// collectEcosystemDenyRules iterates the selected languages, looks up each
// module in the registry, and aggregates their DenyRules output.
func collectEcosystemDenyRules(answers types.WizardAnswers, registry *ecosystem.Registry) []string {
	if registry == nil {
		return nil
	}
	var rules []string
	for _, lang := range answers.Languages {
		mod, ok := registry.ByName(lang.Name)
		if !ok {
			continue
		}
		cfg := ecosystem.ToModuleConfig(lang)
		rules = append(rules, mod.DenyRules(cfg)...)
	}
	return sliceutil.Dedup(rules)
}

// buildSandbox returns a SandboxConfig when sandbox is enabled, or nil otherwise.
func buildSandbox(cfg Config) *SandboxConfig {
	if !cfg.SandboxEnabled {
		return nil
	}
	sandbox := &SandboxConfig{
		WriteDeny: []string{"/etc", "/usr"},
	}
	if len(cfg.AllowedDomains) > 0 {
		sandbox.NetAllow = cfg.AllowedDomains
	}
	return sandbox
}

// buildHooks returns the hooks map based on enabled hook presets.
// It delegates to the default HookRegistry which evaluates each registered
// hook's EnabledFunc against the provided answers. When sandbox is enabled,
// hook commands are wrapped with "qsdev sandbox exec".
func buildHooks(answers types.WizardAnswers) map[string][]HookMatcher {
	registry := defaultHookRegistry()
	hooks := registry.BuildHooksMap(answers)
	if hooks != nil && answers.Hooks.SandboxEnabled {
		hooks = wrapHooksForSandbox(hooks, registry)
	}
	return hooks
}

// wrapHooksForSandbox prefixes each hook command with "qsdev sandbox exec
// --category <cat> --" so hooks run inside the sandbox. The category is
// looked up from the registry's SandboxCategory field.
func wrapHooksForSandbox(hooks map[string][]HookMatcher, registry *HookRegistry) map[string][]HookMatcher {
	catMap := make(map[string]string)
	for _, def := range registry.Definitions() {
		if def.SandboxCategory != "" {
			catMap[def.Command] = def.SandboxCategory
		}
	}

	for event, matchers := range hooks {
		for i, m := range matchers {
			for j, h := range m.Hooks {
				cat := catMap[h.Command]
				if cat == "" {
					cat = "linter"
				}
				hooks[event][i].Hooks[j].Command = fmt.Sprintf(
					`qsdev sandbox exec --category %s -- %s`, cat, h.Command)
			}
		}
	}
	return hooks
}

// GenerateSettings produces a .claude/settings.json file from the wizard
// answers, ecosystem registry, and addon configuration. It returns a
// GeneratedFile ready for the generation pipeline.
func GenerateSettings(answers types.WizardAnswers, registry *ecosystem.Registry, cfg Config) (*types.GeneratedFile, error) {
	preset := cfg.DefaultPermissions
	if preset == "" {
		preset = PermissionPresetStandard
	}

	// Override from wizard answers. PermissionLevel is the most specific knob
	// (user-facing --claude-permissions flag), so it takes precedence when set
	// to something other than the tier's own default. When only Tier is set
	// (and PermissionLevel is empty because FillDefaults didn't populate it),
	// derive the preset from the tier.
	if answers.PermissionLevel != "" {
		preset = PermissionPreset(answers.PermissionLevel)
	} else if answers.Tier != "" {
		t := resolveTier(answers)
		preset = PermissionPreset(t.DefaultPermissionPreset())
	}

	settings := SettingsJSON{
		Permissions: buildPermissions(preset, answers, registry, cfg),
		Sandbox:     buildSandbox(cfg),
		Hooks:       buildHooks(answers),
	}

	jsonBytes, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling settings.json: %w", err)
	}

	// Append trailing newline for POSIX compliance.
	jsonBytes = append(jsonBytes, '\n')

	return &types.GeneratedFile{
		Path:     ".claude/settings.json",
		Content:  jsonBytes,
		Mode:     0o644,
		Strategy: types.ThreeWayMerge,
	}, nil
}
