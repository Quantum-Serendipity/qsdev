package claudecode

import "embed"

// PermissionPreset defines the level of tool access granted to Claude Code.
type PermissionPreset string

const (
	PermissionPresetMinimal         PermissionPreset = "minimal"
	PermissionPresetStandard        PermissionPreset = "standard"
	PermissionPresetPermissive      PermissionPreset = "permissive"
	PermissionPresetCustom          PermissionPreset = "custom"
	PermissionPresetSupplyChainOnly PermissionPreset = "supply-chain-only"
)

// HookConfig defines a Claude Code event hook.
type HookConfig struct {
	Event   string `yaml:"event"   json:"event"`
	Matcher string `yaml:"matcher" json:"matcher"`
	Command string `yaml:"command" json:"command"`
	Timeout int    `yaml:"timeout" json:"timeout"`
}

// MCPServerConfig defines an MCP server integration.
type MCPServerConfig struct {
	Name    string            `yaml:"name"             json:"name"`
	Command string            `yaml:"command"          json:"command"`
	Args    []string          `yaml:"args,omitempty"   json:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"    json:"env,omitempty"`
}

// Config holds the claudecode addon configuration.
type Config struct {
	SkillLibrary       *embed.FS         `yaml:"-"`
	DefaultPermissions PermissionPreset  `yaml:"default_permissions,omitempty"`
	DefaultHooks       []HookConfig      `yaml:"default_hooks,omitempty"`
	DefaultSkills      []string          `yaml:"default_skills,omitempty"`
	ExtraAllowPatterns []string          `yaml:"extra_allow_patterns,omitempty"`
	ExtraDenyPatterns  []string          `yaml:"extra_deny_patterns,omitempty"`
	SandboxEnabled     bool              `yaml:"sandbox_enabled,omitempty"`
	AllowedDomains     []string          `yaml:"allowed_domains,omitempty"`
	MCPServers         []MCPServerConfig `yaml:"mcp_servers,omitempty"`
}

// Option is a functional option for configuring the claudecode addon.
type Option func(*Config)

// NewConfig creates a Config by applying the given options.
func NewConfig(opts ...Option) Config {
	var cfg Config
	for _, o := range opts {
		o(&cfg)
	}
	return cfg
}

func WithSkillLibrary(fs embed.FS) Option {
	return func(c *Config) {
		c.SkillLibrary = &fs
	}
}

func WithDefaultPermissions(preset PermissionPreset) Option {
	return func(c *Config) {
		c.DefaultPermissions = preset
	}
}

func WithDefaultHooks(hooks []HookConfig) Option {
	return func(c *Config) {
		c.DefaultHooks = append(c.DefaultHooks, hooks...)
	}
}

func WithDefaultSkills(skills ...string) Option {
	return func(c *Config) {
		c.DefaultSkills = append(c.DefaultSkills, skills...)
	}
}

func WithExtraAllowPatterns(patterns ...string) Option {
	return func(c *Config) {
		c.ExtraAllowPatterns = append(c.ExtraAllowPatterns, patterns...)
	}
}

func WithExtraDenyPatterns(patterns ...string) Option {
	return func(c *Config) {
		c.ExtraDenyPatterns = append(c.ExtraDenyPatterns, patterns...)
	}
}

func WithSandbox(enabled bool) Option {
	return func(c *Config) {
		c.SandboxEnabled = enabled
	}
}

func WithAllowedDomains(domains ...string) Option {
	return func(c *Config) {
		c.AllowedDomains = append(c.AllowedDomains, domains...)
	}
}

func WithMCPServer(server MCPServerConfig) Option {
	return func(c *Config) {
		c.MCPServers = append(c.MCPServers, server)
	}
}
