package ecosystem

import (
	"fmt"
	"sort"
	"sync"
)

// Registry is a thread-safe collection of EcosystemModule implementations.
// Modules are registered by name and can be queried individually or in bulk.
type Registry struct {
	mu      sync.RWMutex
	modules map[string]EcosystemModule
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		modules: make(map[string]EcosystemModule),
	}
}

// Register adds a module to the registry. It returns an error if a module
// with the same Name() is already registered.
func (r *Registry) Register(m EcosystemModule) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := m.Name()
	if _, exists := r.modules[name]; exists {
		return fmt.Errorf("ecosystem module %q is already registered", name)
	}
	r.modules[name] = m
	return nil
}

// All returns every registered module, sorted by tier (ascending) then
// name (alphabetical) within each tier.
func (r *Registry) All() []EcosystemModule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	mods := make([]EcosystemModule, 0, len(r.modules))
	for _, m := range r.modules {
		mods = append(mods, m)
	}
	sort.Slice(mods, func(i, j int) bool {
		if mods[i].Tier() != mods[j].Tier() {
			return mods[i].Tier() < mods[j].Tier()
		}
		return mods[i].Name() < mods[j].Name()
	})
	return mods
}

// ByTier returns all modules with the given tier, sorted by name.
func (r *Registry) ByTier(tier int) []EcosystemModule {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var mods []EcosystemModule
	for _, m := range r.modules {
		if m.Tier() == tier {
			mods = append(mods, m)
		}
	}
	sort.Slice(mods, func(i, j int) bool {
		return mods[i].Name() < mods[j].Name()
	})
	return mods
}

// ByName looks up a single module by its canonical name.
func (r *Registry) ByName(name string) (EcosystemModule, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	m, ok := r.modules[name]
	return m, ok
}

// DetectAll runs every registered module's Detect method against root and
// returns a DetectionSummary containing the raw results plus an aggregated
// DetectedProject.
func (r *Registry) DetectAll(root string) *DetectionSummary {
	r.mu.RLock()
	mods := make([]EcosystemModule, 0, len(r.modules))
	for _, m := range r.modules {
		mods = append(mods, m)
	}
	r.mu.RUnlock()

	results := make(map[string]DetectionResult, len(mods))
	for _, m := range mods {
		results[m.Name()] = m.Detect(root)
	}

	return &DetectionSummary{
		Project: aggregateDetections(results),
		Results: results,
	}
}

// DetectWithEnvironment runs every registered module's Detect method against
// root, detects environment state (devenv files, claude configs, git), and
// returns a DetectionSummary with all fields populated.
func (r *Registry) DetectWithEnvironment(root string) *DetectionSummary {
	summary := r.DetectAll(root)
	env := DetectEnvironment(root)
	applyEnvironment(&summary.Project, env)
	return summary
}

// Names returns the sorted list of all registered module names.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.modules))
	for name := range r.modules {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Count returns the number of registered modules.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return len(r.modules)
}

var (
	defaultRegistryOnce sync.Once
	defaultRegistry     *Registry
)

// DefaultRegistry returns the package-level singleton Registry.
// It is lazily initialized on first call.
func DefaultRegistry() *Registry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewRegistry()
	})
	return defaultRegistry
}
