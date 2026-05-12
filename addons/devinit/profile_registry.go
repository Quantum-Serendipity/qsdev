package devinit

import (
	"fmt"
	"sync"
)

// ProfileSummary provides a lightweight view of a registered project-type profile.
type ProfileSummary struct {
	Name        string
	Description string
}

// ProjectProfileRegistry is a thread-safe store of named project-type profiles.
// It preserves insertion order for deterministic List() output.
type ProjectProfileRegistry struct {
	mu       sync.RWMutex
	profiles map[string]Profile
	order    []string
}

// NewProjectProfileRegistry creates an empty ProjectProfileRegistry.
func NewProjectProfileRegistry() *ProjectProfileRegistry {
	return &ProjectProfileRegistry{
		profiles: make(map[string]Profile),
	}
}

// Register adds a named profile to the registry. It returns an error if a
// profile with the same name is already registered.
func (r *ProjectProfileRegistry) Register(name string, p Profile) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.profiles[name]; exists {
		return fmt.Errorf("project-type profile %q already registered", name)
	}
	r.profiles[name] = p
	r.order = append(r.order, name)
	return nil
}

// Get retrieves a profile by name. The boolean reports whether it was found.
func (r *ProjectProfileRegistry) Get(name string) (Profile, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	p, ok := r.profiles[name]
	return p, ok
}

// List returns summaries of all registered profiles in insertion order.
func (r *ProjectProfileRegistry) List() []ProfileSummary {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]ProfileSummary, 0, len(r.order))
	for _, name := range r.order {
		p := r.profiles[name]
		list = append(list, ProfileSummary{
			Name:        name,
			Description: p.Description,
		})
	}
	return list
}

// Names returns the names of all registered profiles in insertion order.
func (r *ProjectProfileRegistry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, len(r.order))
	copy(names, r.order)
	return names
}
