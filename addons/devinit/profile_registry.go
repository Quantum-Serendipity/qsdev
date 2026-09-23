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
// It preserves insertion order for deterministic List() output. Register, Get
// and Names are promoted unchanged from the embedded generic registry; only
// List adds behavior.
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

// List returns summaries of all registered profiles in insertion order.
func (r *ProjectProfileRegistry) List() []ProfileSummary {
	names := r.Names() // insertion order due to WithInsertionOrder
	items := r.All()
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
