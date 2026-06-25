package spi

import (
	"context"
	"errors"
	"sort"
	"sync"

	"github.com/Quantum-Serendipity/qsdev/internal/registry"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

// errNilHandler is returned when a Chain is executed without a final handler.
var errNilHandler = errors.New("spi: nil final tool handler")

// FrameworkAdapter is the contract concrete framework adapters implement to
// contribute framework-specific tools, resources, and prompts to the universal
// server. Adapters live under internal/mcpserve/adapters/* and delegate to
// addons; they self-register into the DefaultRegistry from cmd/qsdev/main.go.
//
// The interface is intentionally minimal: identity, an applicability check, and
// three contribution methods. Later tasks implement concrete adapters against
// it.
type FrameworkAdapter interface {
	// ID returns the framework this adapter serves.
	ID() aiframework.FrameworkID
	// Applies reports whether this adapter should contribute for the given
	// resolved project root. Adapters that are always applicable return true.
	Applies(ctx context.Context, projectRoot string) bool
	// Tools returns the tool registrations this adapter contributes.
	Tools() []ToolRegistration
	// Resources returns the resource registrations this adapter contributes.
	Resources() []ResourceRegistration
	// Prompts returns the prompt registrations this adapter contributes.
	Prompts() []PromptRegistration
}

// AdapterRegistry is a thread-safe collection of FrameworkAdapter
// implementations, keyed by FrameworkID. It mirrors the pattern used by
// pkg/ecosystem and internal/mcpserver: embed the generic registry and add a
// domain-typed Register/All.
type AdapterRegistry struct {
	*registry.Registry[FrameworkAdapter]
}

// NewAdapterRegistry creates an empty AdapterRegistry.
func NewAdapterRegistry() *AdapterRegistry {
	return &AdapterRegistry{
		Registry: registry.New[FrameworkAdapter](
			registry.WithEntityName("framework adapter"),
		),
	}
}

// Register adds an adapter keyed by its ID(). It returns an error if an adapter
// with the same ID is already registered.
func (r *AdapterRegistry) Register(a FrameworkAdapter) error {
	return r.Registry.Register(string(a.ID()), a)
}

// All returns every registered adapter, sorted by FrameworkID for determinism.
func (r *AdapterRegistry) All() []FrameworkAdapter {
	adapters := r.Values()
	sort.Slice(adapters, func(i, j int) bool {
		return adapters[i].ID() < adapters[j].ID()
	})
	return adapters
}

var (
	defaultRegistryOnce sync.Once
	defaultRegistry     *AdapterRegistry
)

// DefaultRegistry returns the package-level singleton AdapterRegistry, lazily
// initialized on first use. Concrete adapters register into this from
// cmd/qsdev/main.go; the server consumes it. Explicit registration is preferred
// over init() so wiring order stays visible at the program entry point.
func DefaultRegistry() *AdapterRegistry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewAdapterRegistry()
	})
	return defaultRegistry
}
