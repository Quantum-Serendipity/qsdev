package catalog

import (
	"fmt"
	"log/slog"
	"maps"
	"os"
	"slices"
	"strings"
)

// Load reads the embedded defaults and optionally overlays organization and
// project configuration from unified defaults files. Layers apply in order
// embedded, org, project:
//
//   - the org file (the developer's own defaults) may change most of the
//     catalog, but not below the built-in security floor (see
//     securityFloorViolations); a file that tries fails the load with an
//     error wrapping ErrOverlayLoosens;
//   - the committed project policy applies last and may only add or tighten
//     (see applyProjectOverlay), so the org file cannot erase its additions;
//     it is judged against the embedded defaults alone, so the org file never
//     makes it fail, and a file that tries anything else fails the load with
//     an error wrapping ErrProjectOverlayRejected.
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
		cat, err = applyOrgConfigFile(embedded, cfg.orgConfigFile)
		if err != nil {
			return nil, err
		}
	}

	if cfg.projectConfigFile != "" {
		cat, err = applyProjectConfigFile(cat, embedded, cfg.projectConfigFile)
		if err != nil {
			return nil, err
		}
	}

	errs := cat.Validate()
	errs = append(errs, validateBuiltinTierOrders(embedded, cat)...)
	if len(errs) > 0 {
		return nil, fmt.Errorf("catalog validation: %s", joinCatalogErrors(errs))
	}

	return cat, nil
}

// LoadUserScope loads the catalog qsdev and the user vouch for: the embedded
// defaults plus the user's org overlay (PolicyOrgConfigFile), never the
// project defaults file, which is repository content. It is the source of
// launch trust (mcpregistry.TrustedDefinitions) and of the MCP server's
// operator opt-ins (MCPServeOptIns). An overlay that fails to load is
// skipped with a warning, so everything it would grant fails closed; only a
// broken embedded catalog is an error.
func LoadUserScope() (*Catalog, error) {
	org := PolicyOrgConfigFile()
	if org == "" {
		return Load()
	}
	cat, err := Load(WithOrgConfigFile(org))
	if err == nil {
		return cat, nil
	}
	slog.Warn("ignoring invalid user defaults file; using built-in defaults",
		"path", org, "error", err)
	return Load()
}

// applyOrgConfigFile merges the org defaults file at path onto embedded and
// rejects a result below embedded's security floor. A missing file leaves
// embedded unchanged.
func applyOrgConfigFile(embedded *Catalog, path string) (*Catalog, error) {
	orgCat, err := loadUnifiedFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return embedded, nil
		}
		return nil, fmt.Errorf("loading org config from %s: %w", path, err)
	}
	merged := MergeCatalogs(embedded, orgCat)
	if errs := securityFloorViolations(embedded, merged, path); len(errs) > 0 {
		return nil, fmt.Errorf("%w: %s", ErrOverlayLoosens, joinCatalogErrors(errs))
	}
	return merged, nil
}

// applyProjectConfigFile applies the project defaults file at path to cat,
// judging it against the embedded defaults (see applyProjectOverlay). A
// missing file leaves cat unchanged.
func applyProjectConfigFile(cat, embedded *Catalog, path string) (*Catalog, error) {
	ov, err := loadProjectOverlay(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cat, nil
		}
		return nil, fmt.Errorf("loading project config from %s: %w", path, err)
	}
	out, errs := applyProjectOverlay(cat, embedded, ov)
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
