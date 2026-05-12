package devenv

type config struct {
	DefaultLanguages  []string          `yaml:"default_languages,omitempty"`
	DefaultServices   []string          `yaml:"default_services,omitempty"`
	DirenvEnabled     bool              `yaml:"direnv_enabled,omitempty"`
	ExtraPackages     []string          `yaml:"extra_packages,omitempty"`
	TemplateOverrides map[string]string `yaml:"template_overrides,omitempty"`
}

type option func(*config)

func WithDefaultLanguages(langs ...string) option {
	return func(c *config) {
		c.DefaultLanguages = append(c.DefaultLanguages, langs...)
	}
}

func WithDefaultServices(services ...string) option {
	return func(c *config) {
		c.DefaultServices = append(c.DefaultServices, services...)
	}
}

func WithDirenv(enabled bool) option {
	return func(c *config) {
		c.DirenvEnabled = enabled
	}
}

func WithExtraPackages(pkgs ...string) option {
	return func(c *config) {
		c.ExtraPackages = append(c.ExtraPackages, pkgs...)
	}
}

func WithTemplateOverride(name, path string) option {
	return func(c *config) {
		if c.TemplateOverrides == nil {
			c.TemplateOverrides = make(map[string]string)
		}
		c.TemplateOverrides[name] = path
	}
}
