package sandbox

import (
	"fmt"
	"sort"
	"sync"
)

// BackendRegistry manages available sandbox backends and selects the best one
// based on detected system capabilities.
type BackendRegistry struct {
	mu       sync.RWMutex
	backends []SandboxBackend
}

// NewRegistry returns an empty BackendRegistry.
func NewRegistry() *BackendRegistry {
	return &BackendRegistry{}
}

// Register adds a backend to the registry.
func (r *BackendRegistry) Register(b SandboxBackend) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.backends = append(r.backends, b)
}

// Select returns the strongest available sandbox backend. Backends are tried
// in tier order (lowest tier value = strongest isolation). Returns the
// UnsandboxedBackend if nothing better is available.
func (r *BackendRegistry) Select() (SandboxBackend, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Sort by tier (strongest first).
	sorted := make([]SandboxBackend, len(r.backends))
	copy(sorted, r.backends)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].Tier() < sorted[j].Tier()
	})

	for _, b := range sorted {
		if err := b.Available(); err == nil {
			return b, nil
		}
	}

	return &UnsandboxedBackend{}, nil
}

// SelectByName returns a specific backend by name, or an error if not found
// or unavailable.
func (r *BackendRegistry) SelectByName(name string) (SandboxBackend, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, b := range r.backends {
		if b.Name() == name {
			if err := b.Available(); err != nil {
				return nil, fmt.Errorf("backend %q not available: %w", name, err)
			}
			return b, nil
		}
	}

	return nil, fmt.Errorf("backend %q not registered", name)
}

// List returns all registered backends with their availability status.
func (r *BackendRegistry) List() []BackendStatus {
	r.mu.RLock()
	defer r.mu.RUnlock()

	statuses := make([]BackendStatus, 0, len(r.backends))
	for _, b := range r.backends {
		err := b.Available()
		statuses = append(statuses, BackendStatus{
			Name:      b.Name(),
			Tier:      b.Tier(),
			Available: err == nil,
			Error:     err,
		})
	}
	return statuses
}

// BackendStatus summarises a registered backend's availability.
type BackendStatus struct {
	Name      string
	Tier      DegradationTier
	Available bool
	Error     error
}
