package types

// Schema version constants for .gdev.yaml configuration files.
const (
	ConfigVersionMin     = 1
	ConfigVersionMax     = 1
	ConfigVersionCurrent = 1
)

// GdevConfig represents the parsed contents of a .gdev.yaml file.
// It is the declarative project configuration that drives devinit behavior.
type GdevConfig struct {
	Version        int              `yaml:"version"`
	GdevVersion    string           `yaml:"gdev_version,omitempty"`
	Profile        string           `yaml:"profile,omitempty"`
	Languages      []LanguageConfig `yaml:"languages,omitempty"`
	Services       []ServiceConfig  `yaml:"services,omitempty"`
	Security       SecurityConfig   `yaml:"security,omitempty"`
	Tools          ToolsConfig      `yaml:"tools,omitempty"`
	ClaudeCode     ClaudeCodeConfig `yaml:"claude_code,omitempty"`
	Infrastructure InfraConfig      `yaml:"infrastructure,omitempty"`
	Client         *ClientConfig    `yaml:"client,omitempty"`
	Git            GitConfig        `yaml:"git,omitempty"`
}

// LanguageConfig specifies a language/platform ecosystem in .gdev.yaml.
type LanguageConfig struct {
	Name           string `yaml:"name"`
	Version        string `yaml:"version,omitempty"`
	PackageManager string `yaml:"package_manager,omitempty"`
}

// ServiceConfig specifies a development service (database, cache, etc.) in .gdev.yaml.
type ServiceConfig struct {
	Name    string            `yaml:"name"`
	Version string            `yaml:"version,omitempty"`
	Options map[string]string `yaml:"options,omitempty"`
}

// SecurityConfig holds security posture settings in .gdev.yaml.
type SecurityConfig struct {
	Level          string `yaml:"level,omitempty"`
	AgeGating      *bool  `yaml:"age_gating,omitempty"`
	ScriptBlocking *bool  `yaml:"script_blocking,omitempty"`
	LockEnforce    *bool  `yaml:"lock_enforcement,omitempty"`
	VulnScanning   *bool  `yaml:"vuln_scanning,omitempty"`
}

// ToolsConfig controls which optional tools are enabled/disabled in .gdev.yaml.
type ToolsConfig struct {
	Enabled  []string                  `yaml:"enabled,omitempty"`
	Disabled []string                  `yaml:"disabled,omitempty"`
	Config   map[string]map[string]any `yaml:"config,omitempty"`
}

// ClaudeCodeConfig holds Claude Code agent settings in .gdev.yaml.
type ClaudeCodeConfig struct {
	Enabled         *bool    `yaml:"enabled,omitempty"`
	PermissionLevel string   `yaml:"permission_level,omitempty"`
	Skills          []string `yaml:"skills,omitempty"`
	MCPServers      []string `yaml:"mcp_servers,omitempty"`
}

// InfraConfig holds infrastructure settings (registry proxy, caches) in .gdev.yaml.
type InfraConfig struct {
	RegistryProxy          string            `yaml:"registry_proxy,omitempty"`
	RegistryProxyOverrides map[string]string `yaml:"registry_proxy_overrides,omitempty"`
	NixCache               string            `yaml:"nix_cache,omitempty"`
	BuildCache             string            `yaml:"build_cache,omitempty"`
}

// GitConfig holds git workflow settings in .gdev.yaml.
type GitConfig struct {
	BranchPattern string `yaml:"branch_pattern,omitempty"`
}

// ClientConfig holds client-specific constraints in .gdev.yaml.
type ClientConfig struct {
	Name               string   `yaml:"name"`
	Compliance         []string `yaml:"compliance,omitempty"`
	SecurityLevel      string   `yaml:"security_level,omitempty"`
	RegistryProxy      string   `yaml:"registry_proxy,omitempty"`
	NixCache           string   `yaml:"nix_cache,omitempty"`
	AllowedMCP         []string `yaml:"allowed_mcp_servers,omitempty"`
	BlockedMCP         []string `yaml:"blocked_mcp_servers,omitempty"`
	DataClassification string   `yaml:"data_classification,omitempty"`
}
