package mcpregistry

import (
	"fmt"
	"sort"
	"sync"

	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
)

// McpServerRegistry is a thread-safe collection of MCP server definitions
// and their cached health results.
type McpServerRegistry struct {
	mu      sync.RWMutex
	servers map[string]*McpServerDefinition
	health  map[string]*HealthResult
}

// NewRegistry creates an empty MCP server registry with initialized maps.
func NewRegistry() *McpServerRegistry {
	return &McpServerRegistry{
		servers: make(map[string]*McpServerDefinition),
		health:  make(map[string]*HealthResult),
	}
}

// Register adds an MCP server definition to the registry. Returns an error
// if a server with the same name is already registered.
func (r *McpServerRegistry) Register(def McpServerDefinition) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.servers[def.Name]; exists {
		return fmt.Errorf("mcp server %q already registered", def.Name)
	}
	r.servers[def.Name] = &def
	return nil
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
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.servers[name]
	return def, ok
}

// All returns all registered server definitions sorted by name.
func (r *McpServerRegistry) All() []*McpServerDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*McpServerDefinition, 0, len(r.servers))
	for _, def := range r.servers {
		result = append(result, def)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// ByCategory returns all servers in the given category, sorted by name.
func (r *McpServerRegistry) ByCategory(cat McpCategory) []*McpServerDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*McpServerDefinition
	for _, def := range r.servers {
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
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*McpServerDefinition
	for name, def := range r.servers {
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
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.servers))
	for name := range r.servers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Count returns the number of registered servers.
func (r *McpServerRegistry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.servers)
}

// SetHealth updates the cached health result for the named server.
func (r *McpServerRegistry) SetHealth(name string, result *HealthResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.health[name] = result
}

// GetHealth retrieves the cached health result for the named server.
func (r *McpServerRegistry) GetHealth(name string) (*HealthResult, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
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
