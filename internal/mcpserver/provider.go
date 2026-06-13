package mcpserver

import (
	"context"
	"sort"

	"github.com/Quantum-Serendipity/qsdev/internal/registry"
)

type HandlerFunc func(ctx context.Context, args map[string]any) (string, error)

type ParamDef struct {
	Name        string
	Description string
	Required    bool
}

type ToolDef struct {
	Name        string
	Description string
	Params      []ParamDef
	Handler     HandlerFunc
}

type Provider interface {
	Name() string
	Description() string
	Tools() []ToolDef
}

// Registry is a thread-safe collection of MCP server providers.
type Registry struct {
	*registry.Registry[Provider]
}

// NewRegistry creates an empty provider registry.
func NewRegistry() *Registry {
	return &Registry{
		Registry: registry.New[Provider](
			registry.WithDuplicatePolicy(registry.AllowOverwrite),
			registry.WithEntityName("provider"),
		),
	}
}

// Register adds a provider, keyed by its Name(). Overwrites are allowed.
func (r *Registry) Register(p Provider) {
	_ = r.Registry.Register(p.Name(), p)
}

// Get returns the provider for the given name and whether it was found.
func (r *Registry) Get(name string) (Provider, bool) {
	return r.Registry.Get(name)
}

// All returns all registered providers sorted by name.
func (r *Registry) All() []Provider {
	result := r.Values()
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name() < result[j].Name()
	})
	return result
}

var defaultRegistry = NewRegistry()

// DefaultRegistry returns the package-level singleton provider registry.
func DefaultRegistry() *Registry {
	return defaultRegistry
}
