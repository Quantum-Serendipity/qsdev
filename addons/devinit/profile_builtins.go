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

// DefaultProjectProfileRegistry returns a ProjectProfileRegistry pre-loaded
// with the built-in project-type profiles. A profile that cannot be loaded
// from the catalog is logged and left unregistered, so selecting it fails with
// "unknown profile" instead of silently producing an empty configuration.
func DefaultProjectProfileRegistry() *ProjectProfileRegistry {
	r, err := loadDefaultProjectProfiles()
	if err != nil {
		slog.Error("loading built-in project profiles", "error", err)
	}
	return r
}

// loadDefaultProjectProfiles registers every built-in profile that loads and
// returns the joined errors for those that do not.
func loadDefaultProjectProfiles() (*ProjectProfileRegistry, error) {
	r := NewProjectProfileRegistry()
	cat, err := catalog.Default()
	if err != nil {
		return r, fmt.Errorf("loading catalog: %w", err)
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
