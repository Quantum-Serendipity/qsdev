package ecosystem

import (
	"sort"
	"sync"

	"github.com/Quantum-Serendipity/qsdev/internal/registry"
)

// Registry is a thread-safe collection of EcosystemModule implementations.
// Modules are registered by name and can be queried individually or in bulk.
type Registry struct {
	*registry.Registry[EcosystemModule]
}

// NewRegistry creates an empty Registry.
func NewRegistry() *Registry {
	return &Registry{
		Registry: registry.New[EcosystemModule](
			registry.WithEntityName("ecosystem module"),
		),
	}
}

// Register adds a module to the registry. It returns an error if a module
// with the same Name() is already registered.
func (r *Registry) Register(m EcosystemModule) error {
	return r.Registry.Register(m.Name(), m)
}

// All returns every registered module, sorted by tier (ascending) then
// name (alphabetical) within each tier.
func (r *Registry) All() []EcosystemModule {
	items := r.Registry.All()
	mods := make([]EcosystemModule, 0, len(items))
	for _, m := range items {
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
	items := r.Registry.All()
	var mods []EcosystemModule
	for _, m := range items {
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
	return r.Get(name)
}

// DetectAll runs every registered module's Detect method against root and
// returns a DetectionSummary containing the raw results plus an aggregated
// DetectedProject.
func (r *Registry) DetectAll(root string) *DetectionSummary {
	items := r.Registry.All()
	mods := make([]EcosystemModule, 0, len(items))
	for _, m := range items {
		mods = append(mods, m)
	}

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
	return r.Registry.Names()
}

// Count returns the number of registered modules.
func (r *Registry) Count() int {
	return r.Registry.Count()
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
