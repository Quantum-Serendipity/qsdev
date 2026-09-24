package catalog

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Top-level unified defaults sections whose entries are deep-merged by
// MergeCatalogs (see Catalog.entryNodes). The names match the yaml tags on
// UnifiedDefaults.
const (
	sectionTiers                = "tiers"
	sectionCompliance           = "compliance"
	sectionProfiles             = "profiles"
	sectionProjectProfiles      = "project_profiles"
	sectionTools                = "tools"
	sectionMCPServers           = "mcp_servers"
	sectionPermissionPresetDefs = "permission_preset_defs"
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
	DefaultTier        string              `yaml:"default_tier,omitempty"`
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

	// Docs Corpus
	DocsCorpus *DocsCorpusConfig `yaml:"docs_corpus,omitempty"`

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
	cat.derivations.DefaultTier = u.DefaultTier
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

	// Docs Corpus
	if u.DocsCorpus != nil {
		cat.docsCorpus = *u.DocsCorpus
	}

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
	u.DefaultTier = c.derivations.DefaultTier
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

	// Docs Corpus
	dc := c.docsCorpus
	u.DocsCorpus = &dc

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
		"custom_hooks", "hook_tier_order", "hook_tiers", "default_tier", "tier_to_compliance",
		"tier_to_enabled_tools", "default_mcp_servers", "default_agent_tools",
		"languages", "services", "permission_presets", "hook_presets",
		"security_levels", "data_classifications", "package_managers", "tool_categories",
		"docs_corpus",
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

	cat, err := parseUnifiedBytes(data)
	if err != nil {
		return nil, fmt.Errorf("parsing unified defaults %s: %w", path, err)
	}
	return cat, nil
}

// parseUnifiedBytes strictly parses unified defaults YAML. Unknown or
// misspelled keys are rejected (a typo such as "permision_deny_rules" must
// not silently drop the user's intended security rules), as is any YAML
// document after the first, which would otherwise be ignored. An empty or
// comment-only file yields an empty catalog.
func parseUnifiedBytes(data []byte) (*Catalog, error) {
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	var ud UnifiedDefaults
	if err := dec.Decode(&ud); err != nil {
		if errors.Is(err, io.EOF) {
			return &Catalog{}, nil
		}
		return nil, err
	}
	more, err := hasMoreDocuments(dec)
	if err != nil {
		return nil, err
	}
	if more {
		return nil, errors.New("multiple YAML documents are not supported; remove everything after the first \"---\"")
	}

	// Keep the raw document so MergeCatalogs can tell which fields of an
	// entry the file actually sets.
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}

	cat := ud.ToCatalog()
	cat.entryNodes = sectionEntryNodes(&doc)
	return cat, nil
}

// hasMoreDocuments reports whether dec holds another YAML document with
// content. Empty trailing documents (a closing "---", optionally followed by
// comments) carry nothing that could be ignored, so they are skipped.
func hasMoreDocuments(dec *yaml.Decoder) (bool, error) {
	for {
		var doc yaml.Node
		if err := dec.Decode(&doc); err != nil {
			if errors.Is(err, io.EOF) {
				return false, nil
			}
			return false, err
		}
		if len(doc.Content) > 0 && doc.Content[0].Tag != "!!null" {
			return true, nil
		}
	}
}

// sectionEntryNodes indexes the entries of every top-level mapping section
// of a unified defaults document by section and entry name.
func sectionEntryNodes(doc *yaml.Node) map[string]map[string]*yaml.Node {
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil
	}
	out := make(map[string]map[string]*yaml.Node)
	for i := 0; i+1 < len(root.Content); i += 2 {
		section := root.Content[i+1]
		if section.Kind != yaml.MappingNode {
			continue
		}
		entries := make(map[string]*yaml.Node, len(section.Content)/2)
		for j := 0; j+1 < len(section.Content); j += 2 {
			entries[section.Content[j].Value] = section.Content[j+1]
		}
		out[root.Content[i].Value] = entries
	}
	return out
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
