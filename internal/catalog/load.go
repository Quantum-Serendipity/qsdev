package catalog

import (
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
)

// Load reads the embedded defaults and optionally overlays project and
// organization configuration from unified defaults files. Layers apply in
// order embedded, project, org, so the org file (the developer's own
// defaults) overrides the project file. The project file is committed to
// the repository and is restricted to adding or tightening (see
// applyProjectOverlay); a file that tries anything else fails the load with
// an error wrapping ErrProjectOverlayRejected.
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

	// The project layer goes first so the org layer above it keeps the last
	// word; the project layer itself may only add or tighten.
	if cfg.projectConfigFile != "" {
		cat, err = applyProjectConfigFile(cat, cfg.projectConfigFile)
		if err != nil {
			return nil, err
		}
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

	errs := cat.Validate()
	errs = append(errs, validateBuiltinTierOrders(embedded, cat)...)
	if len(errs) > 0 {
		return nil, fmt.Errorf("catalog validation: %s", joinCatalogErrors(errs))
	}

	return cat, nil
}

// applyProjectConfigFile applies the project defaults file at path to cat.
// A missing file leaves cat unchanged.
func applyProjectConfigFile(cat *Catalog, path string) (*Catalog, error) {
	ov, err := loadProjectOverlay(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cat, nil
		}
		return nil, fmt.Errorf("loading project config from %s: %w", path, err)
	}
	out, errs := applyProjectOverlay(cat, ov)
	if len(errs) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrProjectOverlayRejected, joinCatalogErrors(errs))
	}
	return out, nil
}

// joinCatalogErrors renders catalog errors as one "; "-separated message.
func joinCatalogErrors(errs []CatalogError) string {
	msgs := make([]string, len(errs))
	for i, e := range errs {
		msgs[i] = e.Error()
	}
	return strings.Join(msgs, "; ")
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
