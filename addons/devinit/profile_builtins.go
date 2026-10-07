package devinit

import (
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
)

// catalogProjectProfiles returns a registry of every project profile the
// catalog defines, registered in name order. The catalog is the single source
// of the built-in profile set, so an org or project overlay that adds a
// profile makes it selectable. Profiles that fail to convert or register are
// returned as joined errors next to the ones that succeeded.
func catalogProjectProfiles(cat *catalog.Catalog) (*ProjectProfileRegistry, error) {
	r := NewProjectProfileRegistry()
	var errs []error
	for _, name := range slices.Sorted(maps.Keys(cat.ProjectProfiles())) {
		p, err := catalogProfile(cat, name)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		if err := r.Register(name, p); err != nil {
			errs = append(errs, fmt.Errorf("registering profile %q: %w", name, err))
		}
	}
	return r, errors.Join(errs...)
}

// catalogProfile converts the catalog definition of a project profile into a
// Profile. Slices are copied so callers cannot mutate the shared catalog.
func catalogProfile(cat *catalog.Catalog, name string) (Profile, error) {
	def, ok := cat.ProjectProfile(name)
	if !ok {
		return Profile{}, fmt.Errorf("project profile %q not found in catalog", name)
	}

	langs := make([]LanguageSpec, len(def.Languages))
	for i, l := range def.Languages {
		langs[i] = LanguageSpec{
			Name:           l.Name,
			Version:        l.Version,
			PackageManager: l.PackageManager,
		}
	}

	return Profile{
		Description:     def.Description,
		Languages:       langs,
		Services:        slices.Clone(def.Services),
		Direnv:          def.Direnv,
		ClaudeCode:      def.ClaudeCode,
		PermissionLevel: def.PermissionLevel,
		Tier:            def.Tier,
		Skills:          slices.Clone(def.Skills),
		Hooks:           slices.Clone(def.Hooks),
	}, nil
}
