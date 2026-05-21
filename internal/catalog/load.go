package catalog

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path"
	"strings"

	"gopkg.in/yaml.v3"
)

// Load reads the embedded defaults and optionally overlays organization
// and project configuration from disk.
func Load(opts ...LoadOption) (*Catalog, error) {
	cfg := &loadConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	cat, err := loadFromFS(defaultsFS, "defaults")
	if err != nil {
		return nil, fmt.Errorf("loading embedded defaults: %w", err)
	}

	if cfg.orgConfigDir != "" {
		orgCat, err := loadFromDisk(cfg.orgConfigDir)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("loading org config from %s: %w", cfg.orgConfigDir, err)
			}
		} else {
			cat = MergeCatalogs(cat, orgCat)
		}
	}

	if cfg.projectConfigDir != "" {
		projCat, err := loadFromDisk(cfg.projectConfigDir)
		if err != nil {
			if !os.IsNotExist(err) {
				return nil, fmt.Errorf("loading project config from %s: %w", cfg.projectConfigDir, err)
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
	orgConfigDir     string
	projectConfigDir string
}

// WithOrgConfig sets the organization override directory.
func WithOrgConfig(dir string) LoadOption {
	return func(c *loadConfig) { c.orgConfigDir = dir }
}

// WithProjectConfig sets the project override directory.
func WithProjectConfig(dir string) LoadOption {
	return func(c *loadConfig) { c.projectConfigDir = dir }
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

// loadFromDisk loads catalog data from a directory on disk.
// Missing files are silently skipped; malformed YAML returns an error.
func loadFromDisk(dir string) (*Catalog, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("%s is not a directory", dir)
	}

	cat := &Catalog{}
	dirFS := os.DirFS(dir)

	overlayFiles := []struct {
		name   string
		target any
	}{
		{"tiers.yaml", &cat.tiers},
		{"compliance.yaml", &cat.compliance},
		{"profiles.yaml", &cat.profiles},
		{"project_profiles.yaml", &cat.projectProfiles},
		{"tools.yaml", &cat.tools},
		{"security.yaml", &cat.security},
		{"hook_tiers.yaml", &cat.hookTiers},
		{"derivations.yaml", &cat.derivations},
		{"validation.yaml", &cat.validation},
	}

	for _, f := range overlayFiles {
		if err := loadYAMLFile(dirFS, f.name, f.target); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, fmt.Errorf("parsing %s in %s: %w", f.name, dir, err)
		}
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
