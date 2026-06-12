package extlog

import (
	"sync"

	"github.com/Quantum-Serendipity/qsdev/internal/registry"
)

// Registry is a thread-safe collection of LogProvider implementations.
type Registry struct {
	*registry.Registry[LogProvider]
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		Registry: registry.New[LogProvider](
			registry.WithDuplicatePolicy(registry.AllowOverwrite),
			registry.WithEntityName("log provider"),
		),
	}
}

// Register adds a provider. Duplicate names are silently overwritten.
func (r *Registry) Register(p LogProvider) {
	_ = r.Registry.Register(p.Name(), p)
}

// All returns all registered providers.
func (r *Registry) All() []LogProvider {
	items := r.Registry.All()
	result := make([]LogProvider, 0, len(items))
	for _, p := range items {
		result = append(result, p)
	}
	return result
}

// ByName returns a provider by name.
func (r *Registry) ByName(name string) (LogProvider, bool) {
	return r.Get(name)
}

// DetectAll returns providers that found available logs.
func (r *Registry) DetectAll(projectRoot, homeDir string) []LogProvider {
	items := r.Registry.All()
	var available []LogProvider
	for _, p := range items {
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
