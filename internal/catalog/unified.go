package catalog

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// UnifiedDefaults is the user-facing schema for ~/.config/qsdev/defaults.yaml.
// It flattens the 9 internal catalog files into a single file with intuitive
// top-level keys. Users edit this file to customize defaults across all projects.
type UnifiedDefaults struct {
	// Tiers
	Tiers      map[string]TierDef            `yaml:"tiers,omitempty"`
	Compliance map[string]ComplianceLevelDef `yaml:"compliance,omitempty"`

	// Profiles
	Profiles        map[string]ProfileDef        `yaml:"profiles,omitempty"`
	ProfileAliases  map[string]string            `yaml:"profile_aliases,omitempty"`
	ProjectProfiles map[string]ProjectProfileDef `yaml:"project_profiles,omitempty"`

	// Tools
	Tools map[string]ToolDef `yaml:"tools,omitempty"`

	// MCP Servers
	MCPServers map[string]MCPServerDef `yaml:"mcp_servers,omitempty"`

	// Security
	SecurityHooks []string        `yaml:"security_hooks,omitempty"`
	BasePackages  []string        `yaml:"base_packages,omitempty"`
	UnsetVars     []string        `yaml:"unset_vars,omitempty"`
	KeepVars      []string        `yaml:"keep_vars,omitempty"`
	CustomHooks   []CustomHookDef `yaml:"custom_hooks,omitempty"`

	// Hook Tiers
	HookTierOrder []string            `yaml:"hook_tier_order,omitempty"`
	HookTiers     map[string][]string `yaml:"hook_tiers,omitempty"`

	// Derivations
	TierToCompliance   map[string]string   `yaml:"tier_to_compliance,omitempty"`
	TierToEnabledTools map[string][]string `yaml:"tier_to_enabled_tools,omitempty"`
	DefaultMCPServers  []string            `yaml:"default_mcp_servers,omitempty"`
	DefaultAgentTools  *DefaultAgentTools  `yaml:"default_agent_tools,omitempty"`

	// Validation
	Languages           *ValidationLanguages `yaml:"languages,omitempty"`
	Services            []string             `yaml:"services,omitempty"`
	PermissionPresets   []string             `yaml:"permission_presets,omitempty"`
	HookPresets         []string             `yaml:"hook_presets,omitempty"`
	SecurityLevels      []string             `yaml:"security_levels,omitempty"`
	DataClassifications []string             `yaml:"data_classifications,omitempty"`
	PackageManagers     map[string][]string  `yaml:"package_managers,omitempty"`
	ToolCategories      []ToolCategoryDef    `yaml:"tool_categories,omitempty"`

	// Permission rules
	PermissionDenyRules           map[string][]string            `yaml:"permission_deny_rules,omitempty"`
	PermissionSupplyChainDenySets []string                       `yaml:"permission_supply_chain_deny_sets,omitempty"`
	PermissionAllDenySets         []string                       `yaml:"permission_all_deny_sets,omitempty"`
	PermissionAllowRules          map[string][]string            `yaml:"permission_allow_rules,omitempty"`
	PermissionAskRules            map[string][]string            `yaml:"permission_ask_rules,omitempty"`
	PermissionPackageAskSets      []string                       `yaml:"permission_package_ask_sets,omitempty"`
	PermissionPresetDefs          map[string]PermissionPresetDef `yaml:"permission_preset_defs,omitempty"`
}

// ToCatalog maps each UnifiedDefaults field to the corresponding Catalog field,
// producing a Catalog suitable for use with MergeCatalogs.
func (u *UnifiedDefaults) ToCatalog() *Catalog {
	cat := &Catalog{}

	// Tiers
	cat.tiers.Tiers = u.Tiers
	cat.compliance.Levels = u.Compliance

	// Profiles
	cat.profiles.Profiles = u.Profiles
	cat.profiles.Aliases = u.ProfileAliases
	cat.projectProfiles.Profiles = u.ProjectProfiles

	// Tools
	cat.tools.Tools = u.Tools

	// MCP Servers
	cat.mcpServers = u.MCPServers

	// Security
	cat.security.Hooks.Default = u.SecurityHooks
	cat.security.BasePackages = u.BasePackages
	cat.security.CleanEnvironment.UnsetVars = u.UnsetVars
	cat.security.CleanEnvironment.KeepVars = u.KeepVars
	cat.security.CustomHooks = u.CustomHooks

	// Hook Tiers
	cat.hookTiers.TierOrder = u.HookTierOrder
	cat.hookTiers.Tiers = u.HookTiers

	// Derivations
	cat.derivations.TierToCompliance = u.TierToCompliance
	cat.derivations.TierToEnabledTools = u.TierToEnabledTools
	cat.derivations.DefaultMCPServers = u.DefaultMCPServers
	if u.DefaultAgentTools != nil {
		cat.derivations.DefaultAgentTools = *u.DefaultAgentTools
	}

	// Validation
	if u.Languages != nil {
		cat.validation.Languages = *u.Languages
	}
	cat.validation.Services = u.Services
	cat.validation.PermissionPresets = u.PermissionPresets
	cat.validation.HookPresets = u.HookPresets
	cat.validation.SecurityLevels = u.SecurityLevels
	cat.validation.DataClassifications = u.DataClassifications
	cat.validation.PackageManagers = u.PackageManagers
	cat.validation.ToolCategories = u.ToolCategories

	// Permission rules
	cat.permissionRules.DenyRules = u.PermissionDenyRules
	cat.permissionRules.SupplyChainDenySets = u.PermissionSupplyChainDenySets
	cat.permissionRules.AllDenySets = u.PermissionAllDenySets
	cat.permissionRules.AllowRules = u.PermissionAllowRules
	cat.permissionRules.AskRules = u.PermissionAskRules
	cat.permissionRules.PackageAskSets = u.PermissionPackageAskSets
	cat.permissionRules.PresetDefs = u.PermissionPresetDefs

	return cat
}

// ToUnified converts a Catalog back to the flattened UnifiedDefaults
// representation.
func (c *Catalog) ToUnified() *UnifiedDefaults {
	u := &UnifiedDefaults{}

	// Tiers
	u.Tiers = c.tiers.Tiers
	u.Compliance = c.compliance.Levels

	// Profiles
	u.Profiles = c.profiles.Profiles
	u.ProfileAliases = c.profiles.Aliases
	u.ProjectProfiles = c.projectProfiles.Profiles

	// Tools
	u.Tools = c.tools.Tools

	// MCP Servers
	u.MCPServers = c.mcpServers

	// Security
	u.SecurityHooks = c.security.Hooks.Default
	u.BasePackages = c.security.BasePackages
	u.UnsetVars = c.security.CleanEnvironment.UnsetVars
	u.KeepVars = c.security.CleanEnvironment.KeepVars
	u.CustomHooks = c.security.CustomHooks

	// Hook Tiers
	u.HookTierOrder = c.hookTiers.TierOrder
	u.HookTiers = c.hookTiers.Tiers

	// Derivations
	u.TierToCompliance = c.derivations.TierToCompliance
	u.TierToEnabledTools = c.derivations.TierToEnabledTools
	u.DefaultMCPServers = c.derivations.DefaultMCPServers
	agentTools := c.derivations.DefaultAgentTools
	u.DefaultAgentTools = &agentTools

	// Validation
	langs := c.validation.Languages
	u.Languages = &langs
	u.Services = c.validation.Services
	u.PermissionPresets = c.validation.PermissionPresets
	u.HookPresets = c.validation.HookPresets
	u.SecurityLevels = c.validation.SecurityLevels
	u.DataClassifications = c.validation.DataClassifications
	u.PackageManagers = c.validation.PackageManagers
	u.ToolCategories = c.validation.ToolCategories

	// Permission rules
	u.PermissionDenyRules = c.permissionRules.DenyRules
	u.PermissionSupplyChainDenySets = c.permissionRules.SupplyChainDenySets
	u.PermissionAllDenySets = c.permissionRules.AllDenySets
	u.PermissionAllowRules = c.permissionRules.AllowRules
	u.PermissionAskRules = c.permissionRules.AskRules
	u.PermissionPackageAskSets = c.permissionRules.PackageAskSets
	u.PermissionPresetDefs = c.permissionRules.PresetDefs

	return u
}

// SectionNames returns the valid section names for the unified defaults file.
func SectionNames() []string {
	return []string{
		"tiers", "compliance", "profiles", "profile_aliases", "project_profiles",
		"tools", "mcp_servers", "security_hooks", "base_packages", "unset_vars", "keep_vars",
		"custom_hooks", "hook_tier_order", "hook_tiers", "tier_to_compliance",
		"tier_to_enabled_tools", "default_mcp_servers", "default_agent_tools",
		"languages", "services", "permission_presets", "hook_presets",
		"security_levels", "data_classifications", "package_managers", "tool_categories",
		"permission_deny_rules", "permission_supply_chain_deny_sets",
		"permission_all_deny_sets", "permission_allow_rules", "permission_ask_rules",
		"permission_package_ask_sets", "permission_preset_defs",
	}
}

// loadUnifiedFile reads a unified defaults YAML file and returns it as a Catalog.
func loadUnifiedFile(path string) (*Catalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var ud UnifiedDefaults
	if err := yaml.Unmarshal(data, &ud); err != nil {
		return nil, fmt.Errorf("parsing unified defaults %s: %w", path, err)
	}

	return ud.ToCatalog(), nil
}

// LoadEmbeddedOnly loads only the embedded catalog defaults with no overlays.
func LoadEmbeddedOnly() (*Catalog, error) {
	return Load()
}

// defaultsTemplateHeader is prepended to generated defaults templates.
const defaultsTemplateHeader = `# qsdev user defaults
#
# Override any embedded default by uncommenting and modifying values below.
# Only non-empty sections are applied — omitted sections use built-in defaults.
#
# Commands:
#   qsdev defaults show              Show effective (merged) defaults
#   qsdev defaults show --section X  Show one section (tiers, tools, security_hooks, ...)
#   qsdev defaults validate          Validate this file
#   qsdev defaults edit              Open this file in $EDITOR
#   qsdev defaults reset             Remove this file
#
`

// GenerateDefaultsTemplate loads the embedded defaults, converts them to the
// unified representation, marshals to YAML, and comments out every line. A
// header comment is prepended explaining usage.
func GenerateDefaultsTemplate() ([]byte, error) {
	cat, err := LoadEmbeddedOnly()
	if err != nil {
		return nil, fmt.Errorf("loading embedded defaults for template: %w", err)
	}

	unified := cat.ToUnified()

	yamlBytes, err := yaml.Marshal(unified)
	if err != nil {
		return nil, fmt.Errorf("marshaling unified defaults: %w", err)
	}

	lines := strings.Split(string(yamlBytes), "\n")
	var commented []string
	for _, line := range lines {
		if line == "" {
			commented = append(commented, "#")
		} else {
			commented = append(commented, "# "+line)
		}
	}

	result := defaultsTemplateHeader + strings.Join(commented, "\n")
	return []byte(result), nil
}
