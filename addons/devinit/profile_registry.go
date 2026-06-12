package devinit

import (
	"github.com/Quantum-Serendipity/qsdev/internal/registry"
)

// ProfileSummary provides a lightweight view of a registered project-type profile.
type ProfileSummary struct {
	Name        string
	Description string
}

// ProjectProfileRegistry is a thread-safe store of named project-type profiles.
// It preserves insertion order for deterministic List() output.
type ProjectProfileRegistry struct {
	*registry.Registry[Profile]
}

// NewProjectProfileRegistry creates an empty ProjectProfileRegistry.
func NewProjectProfileRegistry() *ProjectProfileRegistry {
	return &ProjectProfileRegistry{
		Registry: registry.New[Profile](
			registry.WithEntityName("project-type profile"),
			registry.WithInsertionOrder(),
		),
	}
}

// Register adds a named profile to the registry. It returns an error if a
// profile with the same name is already registered.
func (r *ProjectProfileRegistry) Register(name string, p Profile) error {
	return r.Registry.Register(name, p)
}

// Get retrieves a profile by name. The boolean reports whether it was found.
func (r *ProjectProfileRegistry) Get(name string) (Profile, bool) {
	return r.Registry.Get(name)
}

// List returns summaries of all registered profiles in insertion order.
func (r *ProjectProfileRegistry) List() []ProfileSummary {
	names := r.Registry.Names() // insertion order due to WithInsertionOrder
	items := r.Registry.All()
	list := make([]ProfileSummary, 0, len(names))
	for _, name := range names {
		p := items[name]
		list = append(list, ProfileSummary{
			Name:        name,
			Description: p.Description,
		})
	}
	return list
}

// Names returns the names of all registered profiles in insertion order.
func (r *ProjectProfileRegistry) Names() []string {
	return r.Registry.Names()
}
