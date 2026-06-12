package mcpregistry

import (
	"fmt"
	"sort"
	"sync"

	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/internal/registry"
)

// McpServerRegistry is a thread-safe collection of MCP server definitions
// and their cached health results.
type McpServerRegistry struct {
	*registry.Registry[*McpServerDefinition]

	healthMu sync.RWMutex
	health   map[string]*HealthResult
}

// NewRegistry creates an empty MCP server registry with initialized maps.
func NewRegistry() *McpServerRegistry {
	return &McpServerRegistry{
		Registry: registry.New[*McpServerDefinition](
			registry.WithEntityName("mcp server"),
		),
		health: make(map[string]*HealthResult),
	}
}

// Register adds an MCP server definition to the registry. Returns an error
// if a server with the same name is already registered.
func (r *McpServerRegistry) Register(def McpServerDefinition) error {
	return r.Registry.Register(def.Name, &def)
}

// MustRegister adds an MCP server definition to the registry and panics
// if registration fails. Intended for use during initialization where a
// duplicate is a programmer error.
func (r *McpServerRegistry) MustRegister(def McpServerDefinition) {
	if err := r.Register(def); err != nil {
		panic(fmt.Sprintf("mcpregistry: %v", err))
	}
}

// ByName returns the server definition for the given name and whether it was found.
func (r *McpServerRegistry) ByName(name string) (*McpServerDefinition, bool) {
	return r.Registry.Get(name)
}

// All returns all registered server definitions sorted by name.
func (r *McpServerRegistry) All() []*McpServerDefinition {
	items := r.Registry.All()
	result := make([]*McpServerDefinition, 0, len(items))
	for _, def := range items {
		result = append(result, def)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// ByCategory returns all servers in the given category, sorted by name.
func (r *McpServerRegistry) ByCategory(cat McpCategory) []*McpServerDefinition {
	items := r.Registry.All()
	var result []*McpServerDefinition
	for _, def := range items {
		if def.Category == cat {
			result = append(result, def)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// AllHealthy returns servers whose cached health status is healthy, sorted
// by name. Servers without a cached health result are excluded.
func (r *McpServerRegistry) AllHealthy() []*McpServerDefinition {
	items := r.Registry.All()

	r.healthMu.RLock()
	defer r.healthMu.RUnlock()

	var result []*McpServerDefinition
	for name, def := range items {
		hr, ok := r.health[name]
		if !ok || hr.ServerHealth == nil {
			continue
		}
		if hr.Status == mcphealth.StatusHealthy {
			result = append(result, def)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// Names returns all registered server names sorted alphabetically.
func (r *McpServerRegistry) Names() []string {
	return r.Registry.Names()
}

// Count returns the number of registered servers.
func (r *McpServerRegistry) Count() int {
	return r.Registry.Count()
}

// SetHealth updates the cached health result for the named server.
func (r *McpServerRegistry) SetHealth(name string, result *HealthResult) {
	r.healthMu.Lock()
	defer r.healthMu.Unlock()
	r.health[name] = result
}

// GetHealth retrieves the cached health result for the named server.
func (r *McpServerRegistry) GetHealth(name string) (*HealthResult, bool) {
	r.healthMu.RLock()
	defer r.healthMu.RUnlock()
	hr, ok := r.health[name]
	return hr, ok
}

var (
	defaultRegistryOnce sync.Once
	defaultRegistryVal  *McpServerRegistry
)

// DefaultRegistry returns the lazily-initialized singleton MCP server registry.
func DefaultRegistry() *McpServerRegistry {
	defaultRegistryOnce.Do(func() {
		defaultRegistryVal = buildDefault()
	})
	return defaultRegistryVal
}

// ResetDefaultRegistry clears the cached registry. For testing only.
func ResetDefaultRegistry() {
	defaultRegistryOnce = sync.Once{}
	defaultRegistryVal = nil
}
