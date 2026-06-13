package profile

import (
	"sort"

	"github.com/Quantum-Serendipity/qsdev/internal/registry"
)

// ProfileRegistry is a thread-safe store of named InfraProfiles.
type ProfileRegistry struct {
	*registry.Registry[*InfraProfile]
}

// NewProfileRegistry creates an empty ProfileRegistry.
func NewProfileRegistry() *ProfileRegistry {
	return &ProfileRegistry{
		Registry: registry.New[*InfraProfile](
			registry.WithEntityName("profile"),
		),
	}
}

// DefaultProfileRegistry returns a ProfileRegistry pre-loaded with the
// built-in profiles (ConsultingDefault, StartupGitHub, Enterprise).
func DefaultProfileRegistry() *ProfileRegistry {
	r := NewProfileRegistry()
	// Errors are impossible here because names are unique constants.
	_ = r.Register(ConsultingDefault)
	_ = r.Register(StartupGitHub)
	_ = r.Register(Enterprise)
	return r
}

// Register adds a profile to the registry. It returns an error if a profile
// with the same name is already registered.
func (r *ProfileRegistry) Register(p *InfraProfile) error {
	return r.Registry.Register(p.Name, p)
}

// Get retrieves a profile by name. The boolean reports whether it was found.
func (r *ProfileRegistry) Get(name string) (*InfraProfile, bool) {
	return r.Registry.Get(name)
}

// List returns all registered profiles sorted alphabetically by name.
func (r *ProfileRegistry) List() []*InfraProfile {
	list := r.Values()
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list
}
