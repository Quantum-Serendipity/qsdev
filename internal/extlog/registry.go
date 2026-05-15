package extlog

import (
	"sync"
)

// Registry is a thread-safe collection of LogProvider implementations.
type Registry struct {
	mu        sync.RWMutex
	providers map[string]LogProvider
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]LogProvider),
	}
}

// Register adds a provider. Duplicate names are silently overwritten.
func (r *Registry) Register(p LogProvider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

// All returns all registered providers.
func (r *Registry) All() []LogProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]LogProvider, 0, len(r.providers))
	for _, p := range r.providers {
		result = append(result, p)
	}
	return result
}

// ByName returns a provider by name.
func (r *Registry) ByName(name string) (LogProvider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

// DetectAll returns providers that found available logs.
func (r *Registry) DetectAll(projectRoot, homeDir string) []LogProvider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var available []LogProvider
	for _, p := range r.providers {
		if p.Detect(projectRoot, homeDir) {
			available = append(available, p)
		}
	}
	return available
}

var (
	defaultRegistryOnce sync.Once
	defaultRegistry     *Registry
)

// DefaultRegistry returns the global provider registry.
func DefaultRegistry() *Registry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewRegistry()
	})
	return defaultRegistry
}

// RegisterProvider adds a provider to the default registry.
func RegisterProvider(p LogProvider) {
	DefaultRegistry().Register(p)
}
