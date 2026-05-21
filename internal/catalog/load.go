package catalog

import (
	"fmt"
	"io/fs"
	"os"
	"path"
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

	cat, err := loadFromFS(defaultsFS, "defaults")
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

// loadFromFS loads catalog data from an embedded or OS filesystem.
func loadFromFS(fsys fs.FS, root string) (*Catalog, error) {
	cat := &Catalog{}

	if err := loadYAMLFile(fsys, path.Join(root, "tiers.yaml"), &cat.tiers); err != nil {
		return nil, fmt.Errorf("tiers.yaml: %w", err)
	}
	if err := loadYAMLFile(fsys, path.Join(root, "compliance.yaml"), &cat.compliance); err != nil {
		return nil, fmt.Errorf("compliance.yaml: %w", err)
	}
	if err := loadYAMLFile(fsys, path.Join(root, "profiles.yaml"), &cat.profiles); err != nil {
		return nil, fmt.Errorf("profiles.yaml: %w", err)
	}
	if err := loadYAMLFile(fsys, path.Join(root, "project_profiles.yaml"), &cat.projectProfiles); err != nil {
		return nil, fmt.Errorf("project_profiles.yaml: %w", err)
	}
	if err := loadYAMLFile(fsys, path.Join(root, "tools.yaml"), &cat.tools); err != nil {
		return nil, fmt.Errorf("tools.yaml: %w", err)
	}
	if err := loadYAMLFile(fsys, path.Join(root, "security.yaml"), &cat.security); err != nil {
		return nil, fmt.Errorf("security.yaml: %w", err)
	}
	if err := loadYAMLFile(fsys, path.Join(root, "hook_tiers.yaml"), &cat.hookTiers); err != nil {
		return nil, fmt.Errorf("hook_tiers.yaml: %w", err)
	}
	if err := loadYAMLFile(fsys, path.Join(root, "derivations.yaml"), &cat.derivations); err != nil {
		return nil, fmt.Errorf("derivations.yaml: %w", err)
	}
	if err := loadYAMLFile(fsys, path.Join(root, "validation.yaml"), &cat.validation); err != nil {
		return nil, fmt.Errorf("validation.yaml: %w", err)
	}

	return cat, nil
}

func loadYAMLFile(fsys fs.FS, path string, target any) error {
	data, err := fs.ReadFile(fsys, path)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(data, target)
}
