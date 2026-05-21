package catalog

import "fmt"

// CatalogError describes a validation problem in the loaded catalog.
type CatalogError struct {
	File    string
	Field   string
	Message string
}

func (e CatalogError) Error() string {
	return fmt.Sprintf("%s: %s: %s", e.File, e.Field, e.Message)
}

// Validate checks cross-references and invariants across all loaded
// catalog data. It returns all errors found rather than stopping at
// the first.
func (c *Catalog) Validate() []CatalogError {
	var errs []CatalogError

	errs = append(errs, c.validateTiers()...)
	errs = append(errs, c.validateProfiles()...)
	errs = append(errs, c.validateDerivations()...)
	errs = append(errs, c.validateHookTiers()...)
	errs = append(errs, c.validateProjectProfiles()...)

	return errs
}

func (c *Catalog) validateTiers() []CatalogError {
	var errs []CatalogError

	for name, def := range c.tiers.Tiers {
		if def.Description == "" {
			errs = append(errs, CatalogError{"tiers.yaml", name, "missing description"})
		}
		if def.Order == 0 {
			errs = append(errs, CatalogError{"tiers.yaml", name, "order must be > 0"})
		}
		if def.Inherits != "" {
			if _, ok := c.tiers.Tiers[def.Inherits]; !ok {
				errs = append(errs, CatalogError{
					"tiers.yaml", name,
					fmt.Sprintf("inherits unknown tier %q", def.Inherits),
				})
			}
		}
	}

	// Check for inheritance cycles.
	for name := range c.tiers.Tiers {
		if err := c.checkTierCycle(name); err != nil {
			errs = append(errs, CatalogError{"tiers.yaml", name, err.Error()})
		}
	}

	return errs
}

func (c *Catalog) checkTierCycle(start string) error {
	visited := make(map[string]bool)
	current := start
	for i := 0; i < 10; i++ {
		def, ok := c.tiers.Tiers[current]
		if !ok || def.Inherits == "" {
			return nil
		}
		if visited[def.Inherits] {
			return fmt.Errorf("circular inheritance detected")
		}
		visited[current] = true
		current = def.Inherits
	}
	return fmt.Errorf("inheritance chain exceeds maximum depth of 10")
}

func (c *Catalog) validateProfiles() []CatalogError {
	var errs []CatalogError

	for name, def := range c.profiles.Profiles {
		if def.Tier != "" {
			if _, ok := c.tiers.Tiers[def.Tier]; !ok {
				errs = append(errs, CatalogError{
					"profiles.yaml", name,
					fmt.Sprintf("references unknown tier %q", def.Tier),
				})
			}
		}
	}

	for alias, target := range c.profiles.Aliases {
		if _, ok := c.profiles.Profiles[target]; !ok {
			errs = append(errs, CatalogError{
				"profiles.yaml", fmt.Sprintf("aliases.%s", alias),
				fmt.Sprintf("target profile %q does not exist", target),
			})
		}
	}

	return errs
}

func (c *Catalog) validateDerivations() []CatalogError {
	var errs []CatalogError

	for tierName := range c.derivations.TierToCompliance {
		if _, ok := c.tiers.Tiers[tierName]; !ok {
			errs = append(errs, CatalogError{
				"derivations.yaml", "tier_to_compliance",
				fmt.Sprintf("references unknown tier %q", tierName),
			})
		}
	}

	for tierName := range c.derivations.TierToEnabledTools {
		if _, ok := c.tiers.Tiers[tierName]; !ok {
			errs = append(errs, CatalogError{
				"derivations.yaml", "tier_to_enabled_tools",
				fmt.Sprintf("references unknown tier %q", tierName),
			})
		}
	}

	return errs
}

func (c *Catalog) validateHookTiers() []CatalogError {
	var errs []CatalogError

	for _, name := range c.hookTiers.TierOrder {
		if _, ok := c.hookTiers.Tiers[name]; !ok {
			errs = append(errs, CatalogError{
				"hook_tiers.yaml", "tier_order",
				fmt.Sprintf("tier %q listed in order but not defined", name),
			})
		}
	}

	return errs
}

func (c *Catalog) validateProjectProfiles() []CatalogError {
	var errs []CatalogError

	for name, def := range c.projectProfiles.Profiles {
		if def.Tier != "" {
			if _, ok := c.tiers.Tiers[def.Tier]; !ok {
				errs = append(errs, CatalogError{
					"project_profiles.yaml", name,
					fmt.Sprintf("references unknown tier %q", def.Tier),
				})
			}
		}
	}

	return errs
}
