package catalog

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads the embedded defaults and optionally overlays organization
// and project configuration from unified defaults files.
func Load(opts ...LoadOption) (*Catalog, error) {
	cfg := &loadConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	cat, err := loadEmbeddedDefaults()
	if err != nil {
		return nil, fmt.Errorf("loading embedded defaults: %w", err)
	}

	if cfg.orgConfigFile != "" {
		orgCat, err := loadUnifiedFile(cfg.orgConfigFile)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("loading org config from %s: %w", cfg.orgConfigFile, err)
			}
		} else {
			cat = MergeCatalogs(cat, orgCat)
		}
	}

	if cfg.projectConfigFile != "" {
		projCat, err := loadUnifiedFile(cfg.projectConfigFile)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("loading project config from %s: %w", cfg.projectConfigFile, err)
			}
		} else {
			cat = MergeCatalogs(cat, projCat)
		}
	}

	if errs := cat.Validate(); len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return nil, fmt.Errorf("catalog validation: %s", strings.Join(msgs, "; "))
	}

	return cat, nil
}

// LoadOption configures the Load function.
type LoadOption func(*loadConfig)

type loadConfig struct {
	orgConfigFile     string
	projectConfigFile string
}

// WithOrgConfigFile sets the organization-level unified defaults file path.
func WithOrgConfigFile(path string) LoadOption {
	return func(c *loadConfig) { c.orgConfigFile = path }
}

// WithProjectConfigFile sets the project-level unified defaults file path.
func WithProjectConfigFile(path string) LoadOption {
	return func(c *loadConfig) { c.projectConfigFile = path }
}

// loadEmbeddedDefaults parses the embedded defaults.yaml into a Catalog.
func loadEmbeddedDefaults() (*Catalog, error) {
	var u UnifiedDefaults
	if err := yaml.Unmarshal(defaultsData, &u); err != nil {
		return nil, fmt.Errorf("parsing embedded defaults: %w", err)
	}
	return u.ToCatalog(), nil
}
