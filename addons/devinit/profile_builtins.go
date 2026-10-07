package devinit

import (
	"errors"
	"fmt"
	"log/slog"
	"slices"

	"github.com/Quantum-Serendipity/qsdev/internal/catalog"
)

// projectProfileOrder defines the canonical registration order for project profiles.
var projectProfileOrder = []string{
	"go-web", "ts-fullstack", "ts-backend", "python-data", "python-web",
	"rust-cli", "rust-web", "java-web", "elixir-web", "dotnet-web",
}

// errCatalogLoad marks a loadDefaultProjectProfiles failure caused by the
// catalog itself rather than by one profile.
var errCatalogLoad = errors.New("loading catalog")

// DefaultProjectProfileRegistry returns a ProjectProfileRegistry pre-loaded
// with the built-in project-type profiles. A profile that cannot be loaded
// from the catalog is logged and left unregistered, so selecting it fails with
// "unknown profile" instead of silently producing an empty configuration.
//
// A catalog that does not load at all registers nothing and is logged at
// debug only: the root catalog gate or check reports that failure to the
// user for the commands that need the catalog.
func DefaultProjectProfileRegistry() *ProjectProfileRegistry {
	r, err := loadDefaultProjectProfiles()
	switch {
	case errors.Is(err, errCatalogLoad):
		slog.Debug("loading built-in project profiles", "error", err)
	case err != nil:
		slog.Error("loading built-in project profiles", "error", err)
	}
	return r
}

// loadDefaultProjectProfiles registers every built-in profile that loads and
// returns the joined errors for those that do not, or an error wrapping
// errCatalogLoad when the catalog does not load.
func loadDefaultProjectProfiles() (*ProjectProfileRegistry, error) {
	r := NewProjectProfileRegistry()
	cat, err := catalog.Default()
	if err != nil {
		return r, fmt.Errorf("%w: %w", errCatalogLoad, err)
	}
	var errs []error
	for _, name := range projectProfileOrder {
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
