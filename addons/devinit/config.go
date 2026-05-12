package devinit

// LanguageSpec specifies a language with optional version and package manager.
type LanguageSpec struct {
	Name           string `yaml:"name"`
	Version        string `yaml:"version,omitempty"`
	PackageManager string `yaml:"package_manager,omitempty"`
}

// Profile defines a pre-configured set of wizard answers for a project type.
type Profile struct {
	Description     string         `yaml:"description,omitempty"`
	Languages       []LanguageSpec `yaml:"languages,omitempty"`
	Services        []string       `yaml:"services,omitempty"`
	Direnv          bool           `yaml:"direnv,omitempty"`
	ClaudeCode      bool           `yaml:"claude_code,omitempty"`
	PermissionLevel string         `yaml:"permission_level,omitempty"`
	Skills          []string       `yaml:"skills,omitempty"`
	Hooks           []string       `yaml:"hooks,omitempty"`
	GitHooks        []string       `yaml:"git_hooks,omitempty"`
	ExtraPackages   []string       `yaml:"extra_packages,omitempty"`
	MCPServers      []string       `yaml:"mcp_servers,omitempty"`
	InfraProfile    string         `yaml:"infra_profile,omitempty"`
}

type config struct {
	DetectProjectType bool               `yaml:"detect_project_type,omitempty"`
	PlanPreview       bool               `yaml:"plan_preview,omitempty"`
	Profiles          map[string]Profile `yaml:"profiles,omitempty"`
	LastProfile       string             `yaml:"last_profile,omitempty"`
}

type option func(*config)

func WithDetectProjectType(enabled bool) option {
	return func(c *config) {
		c.DetectProjectType = enabled
	}
}

func WithPlanPreview(enabled bool) option {
	return func(c *config) {
		c.PlanPreview = enabled
	}
}

func WithProfile(name string, profile Profile) option {
	return func(c *config) {
		if c.Profiles == nil {
			c.Profiles = make(map[string]Profile)
		}
		c.Profiles[name] = profile
	}
}
