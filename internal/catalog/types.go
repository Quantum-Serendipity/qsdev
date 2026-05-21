package catalog

// TiersFile represents the tiers.yaml schema.
type TiersFile struct {
	Tiers map[string]TierDef `yaml:"tiers"`
}

// TierDef defines a single security tier.
type TierDef struct {
	Order                   int             `yaml:"order"`
	Inherits                string          `yaml:"inherits,omitempty"`
	Description             string          `yaml:"description"`
	DefaultPermissionPreset string          `yaml:"default_permission_preset"`
	Security                *TierSecurity   `yaml:"security,omitempty"`
	Tools                   *TierTools      `yaml:"tools,omitempty"`
	ClaudeCode              *TierClaudeCode `yaml:"claude_code,omitempty"`
}

// TierSecurity holds security settings within a tier definition.
type TierSecurity struct {
	Level           string `yaml:"level,omitempty"`
	AgeGating       *bool  `yaml:"age_gating,omitempty"`
	ScriptBlocking  *bool  `yaml:"script_blocking,omitempty"`
	LockEnforcement *bool  `yaml:"lock_enforcement,omitempty"`
	VulnScanning    *bool  `yaml:"vuln_scanning,omitempty"`
}

// TierTools holds tool configuration within a tier definition.
type TierTools struct {
	Enabled []string `yaml:"enabled"`
}

// TierClaudeCode holds Claude Code settings within a tier definition.
type TierClaudeCode struct {
	Enabled         *bool    `yaml:"enabled,omitempty"`
	PermissionLevel string   `yaml:"permission_level,omitempty"`
	MCPServers      []string `yaml:"mcp_servers,omitempty"`
}

// ComplianceFile represents the compliance.yaml schema.
type ComplianceFile struct {
	Levels map[string]ComplianceLevelDef `yaml:"levels"`
}

// ComplianceLevelDef defines a single compliance level.
type ComplianceLevelDef struct {
	Order                   int      `yaml:"order"`
	AgeGatingThresholdHours int      `yaml:"age_gating_threshold_hours"`
	ScriptBlocking          bool     `yaml:"script_blocking"`
	RequiredPreCommitHooks  []string `yaml:"required_pre_commit_hooks"`
	MCPServerPolicy         string   `yaml:"mcp_server_policy"`
	ClaudePermissionLevel   string   `yaml:"claude_permission_level"`
	ClaudeAuditLog          bool     `yaml:"claude_audit_log"`
	SBOMPolicy              string   `yaml:"sbom_policy"`
	LicenseScanning         bool     `yaml:"license_scanning"`
}

// ProfilesFile represents the profiles.yaml schema.
type ProfilesFile struct {
	Profiles map[string]ProfileDef `yaml:"profiles"`
	Aliases  map[string]string     `yaml:"aliases,omitempty"`
}

// ProfileDef defines a tier-based infrastructure profile.
type ProfileDef struct {
	Tier        string             `yaml:"tier"`
	Description string             `yaml:"description"`
	Security    *ProfileSecurity   `yaml:"security,omitempty"`
	Tools       *ProfileTools      `yaml:"tools,omitempty"`
	ClaudeCode  *ProfileClaudeCode `yaml:"claude_code,omitempty"`
}

// ProfileSecurity holds security settings within a profile.
type ProfileSecurity struct {
	Level           string `yaml:"level,omitempty"`
	AgeGating       *bool  `yaml:"age_gating,omitempty"`
	ScriptBlocking  *bool  `yaml:"script_blocking,omitempty"`
	LockEnforcement *bool  `yaml:"lock_enforcement,omitempty"`
	VulnScanning    *bool  `yaml:"vuln_scanning,omitempty"`
}

// ProfileTools holds tool settings within a profile.
type ProfileTools struct {
	Enabled []string `yaml:"enabled"`
}

// ProfileClaudeCode holds Claude Code settings within a profile.
type ProfileClaudeCode struct {
	Enabled         *bool    `yaml:"enabled,omitempty"`
	PermissionLevel string   `yaml:"permission_level,omitempty"`
	MCPServers      []string `yaml:"mcp_servers,omitempty"`
}

// ProjectProfilesFile represents the project_profiles.yaml schema.
type ProjectProfilesFile struct {
	Profiles map[string]ProjectProfileDef `yaml:"profiles"`
}

// ProjectProfileDef defines a project-type profile (e.g. go-web, ts-fullstack).
type ProjectProfileDef struct {
	Description     string               `yaml:"description"`
	Tier            string               `yaml:"tier"`
	Languages       []ProjectProfileLang `yaml:"languages,omitempty"`
	Services        []string             `yaml:"services"`
	Direnv          bool                 `yaml:"direnv"`
	ClaudeCode      bool                 `yaml:"claude_code"`
	PermissionLevel string               `yaml:"permission_level"`
	Skills          []string             `yaml:"skills,omitempty"`
	Hooks           []string             `yaml:"hooks,omitempty"`
}

// ProjectProfileLang defines a language entry within a project profile.
type ProjectProfileLang struct {
	Name           string `yaml:"name"`
	Version        string `yaml:"version,omitempty"`
	PackageManager string `yaml:"package_manager,omitempty"`
}

// ToolsFile represents the tools.yaml schema.
type ToolsFile struct {
	Tools map[string]ToolDef `yaml:"tools"`
}

// ToolDef defines the declarative metadata for a tool.
type ToolDef struct {
	DisplayName   string             `yaml:"display_name"`
	Category      string             `yaml:"category"`
	Description   string             `yaml:"description"`
	DefaultPolicy string             `yaml:"default_policy"`
	Prerequisites []string           `yaml:"prerequisites,omitempty"`
	Conflicts     []string           `yaml:"conflicts,omitempty"`
	OwnedFiles    []ToolOwnedFileDef `yaml:"owned_files,omitempty"`
}

// ToolOwnedFileDef describes a file owned or contributed to by a tool.
type ToolOwnedFileDef struct {
	Path           string `yaml:"path"`
	Ownership      string `yaml:"ownership"`
	SectionID      string `yaml:"section_id,omitempty"`
	SectionContent string `yaml:"section_content,omitempty"`
}

// SecurityFile represents the security.yaml schema.
type SecurityFile struct {
	Hooks            SecurityHooks       `yaml:"hooks"`
	BasePackages     []string            `yaml:"base_packages"`
	CleanEnvironment CleanEnvironmentDef `yaml:"clean_environment"`
	CustomHooks      []CustomHookDef     `yaml:"custom_hooks,omitempty"`
}

// SecurityHooks holds hook configuration.
type SecurityHooks struct {
	Default []string `yaml:"default"`
}

// CleanEnvironmentDef holds environment variable sanitization config.
type CleanEnvironmentDef struct {
	UnsetVars []string `yaml:"unset_vars"`
	KeepVars  []string `yaml:"keep_vars"`
}

// CustomHookDef defines a custom pre-commit hook.
type CustomHookDef struct {
	ID                 string   `yaml:"id"`
	Name               string   `yaml:"name"`
	Description        string   `yaml:"description"`
	Entry              string   `yaml:"entry,omitempty"`
	EnvPattern         string   `yaml:"env_pattern,omitempty"`
	CredentialPatterns []string `yaml:"credential_patterns,omitempty"`
	Language           string   `yaml:"language"`
	Files              string   `yaml:"files,omitempty"`
	PassFilenames      bool     `yaml:"pass_filenames"`
	Stages             []string `yaml:"stages"`
}

// HookTiersFile represents the hook_tiers.yaml schema.
type HookTiersFile struct {
	TierOrder []string            `yaml:"tier_order"`
	Tiers     map[string][]string `yaml:"tiers"`
}

// DerivationsFile represents the derivations.yaml schema.
type DerivationsFile struct {
	TierToCompliance   map[string]string   `yaml:"tier_to_compliance"`
	TierToEnabledTools map[string][]string `yaml:"tier_to_enabled_tools"`
	DefaultMCPServers  []string            `yaml:"default_mcp_servers"`
	DefaultAgentTools  DefaultAgentTools   `yaml:"default_agent_tools"`
}

// DefaultAgentTools holds default agent tool configuration.
type DefaultAgentTools struct {
	PostmortemEnabled    bool   `yaml:"postmortem_enabled"`
	VersionSentinel      bool   `yaml:"version_sentinel"`
	VersionSentinelHours int    `yaml:"version_sentinel_hours"`
	SembleEnabled        bool   `yaml:"semble_enabled"`
	SembleMode           string `yaml:"semble_mode"`
}

// ValidationFile represents the validation.yaml schema.
type ValidationFile struct {
	Languages           ValidationLanguages `yaml:"languages"`
	Services            []string            `yaml:"services"`
	PermissionPresets   []string            `yaml:"permission_presets"`
	HookPresets         []string            `yaml:"hook_presets"`
	SecurityLevels      []string            `yaml:"security_levels"`
	DataClassifications []string            `yaml:"data_classifications"`
	PackageManagers     map[string][]string `yaml:"package_managers"`
	ToolCategories      []ToolCategoryDef   `yaml:"tool_categories"`
}

// ValidationLanguages holds language lists.
type ValidationLanguages struct {
	All  []string `yaml:"all"`
	Core []string `yaml:"core"`
}

// ToolCategoryDef defines a tool category for display.
type ToolCategoryDef struct {
	ID          string `yaml:"id"`
	DisplayName string `yaml:"display_name"`
}
