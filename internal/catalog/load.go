package catalog

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
)

// Load reads the embedded defaults and optionally overlays organization
// and project configuration from unified defaults files.
func Load(opts ...LoadOption) (*Catalog, error) {
	cfg := &loadConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	embedded, err := loadEmbeddedDefaults()
	if err != nil {
		return nil, fmt.Errorf("loading embedded defaults: %w", err)
	}
	cat := embedded

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

	errs := cat.Validate()
	errs = append(errs, validateBuiltinTierOrders(embedded, cat)...)
	if len(errs) > 0 {
		msgs := make([]string, len(errs))
		for i, e := range errs {
			msgs[i] = e.Error()
		}
		return nil, fmt.Errorf("catalog validation: %s", strings.Join(msgs, "; "))
	}

	return cat, nil
}

// validateBuiltinTierOrders rejects overlays that renumber a built-in tier.
// A tier's order is its numeric level, which Go code compares against fixed
// levels (see internal/tier), so built-in orders cannot move; overlays may
// only add tiers with new, unique orders.
func validateBuiltinTierOrders(embedded, merged *Catalog) []CatalogError {
	var errs []CatalogError
	for _, name := range slices.Sorted(maps.Keys(embedded.tiers.Tiers)) {
		want := embedded.tiers.Tiers[name].Order
		if got := merged.tiers.Tiers[name].Order; got != want {
			errs = append(errs, CatalogError{
				"tiers.yaml", name,
				fmt.Sprintf("order of built-in tier cannot change (got %d, built-in %d)", got, want),
			})
		}
	}
	return errs
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
	cat, err := parseUnifiedBytes(defaultsData)
	if err != nil {
		return nil, fmt.Errorf("parsing embedded defaults: %w", err)
	}
	return cat, nil
}
