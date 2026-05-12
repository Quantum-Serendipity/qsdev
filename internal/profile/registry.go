package profile

import (
	"fmt"
	"sort"
	"sync"
)

// ProfileRegistry is a thread-safe store of named InfraProfiles.
type ProfileRegistry struct {
	mu       sync.RWMutex
	profiles map[string]*InfraProfile
}

// NewProfileRegistry creates an empty ProfileRegistry.
func NewProfileRegistry() *ProfileRegistry {
	return &ProfileRegistry{
		profiles: make(map[string]*InfraProfile),
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
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.profiles[p.Name]; exists {
		return fmt.Errorf("profile %q already registered", p.Name)
	}
	r.profiles[p.Name] = p
	return nil
}

// Get retrieves a profile by name. The boolean reports whether it was found.
func (r *ProfileRegistry) Get(name string) (*InfraProfile, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.profiles[name]
	return p, ok
}

// List returns all registered profiles sorted alphabetically by name.
func (r *ProfileRegistry) List() []*InfraProfile {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]*InfraProfile, 0, len(r.profiles))
	for _, p := range r.profiles {
		list = append(list, p)
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].Name < list[j].Name
	})
	return list
}
