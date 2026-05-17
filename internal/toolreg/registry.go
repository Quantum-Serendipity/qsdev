package toolreg

import (
	"fmt"
	"sort"
	"sync"
)

// Registry is a thread-safe collection of Tool definitions.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]*Tool
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]*Tool),
	}
}

// Register adds a tool to the registry. Returns an error if a tool with
// the same name is already registered.
func (r *Registry) Register(t Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[t.Name]; exists {
		return fmt.Errorf("tool %q already registered", t.Name)
	}
	r.tools[t.Name] = &t
	return nil
}

// MustRegister adds a tool to the registry and panics if registration fails.
// Intended for use in init() where a registration failure is a programmer error.
func (r *Registry) MustRegister(t Tool) {
	if err := r.Register(t); err != nil {
		panic(fmt.Sprintf("toolreg: %v", err))
	}
}

// ByName returns the tool with the given name.
func (r *Registry) ByName(name string) (*Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	t, ok := r.tools[name]
	return t, ok
}

// All returns all registered tools sorted by category then name.
func (r *Registry) All() []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Tool, 0, len(r.tools))
	for _, t := range r.tools {
		result = append(result, t)
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].Category != result[j].Category {
			return categoryOrder(result[i].Category) < categoryOrder(result[j].Category)
		}
		return result[i].Name < result[j].Name
	})
	return result
}

// ByCategory returns all tools in the given category, sorted by name.
func (r *Registry) ByCategory(cat ToolCategory) []*Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*Tool
	for _, t := range r.tools {
		if t.Category == cat {
			result = append(result, t)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

// Names returns all registered tool names, sorted alphabetically.
func (r *Registry) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Count returns the number of registered tools.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.tools)
}

func categoryOrder(c ToolCategory) int {
	switch c {
	case CategorySecurity:
		return 0
	case CategoryAIAgent:
		return 1
	case CategoryDevEx:
		return 2
	case CategoryInfrastructure:
		return 3
	default:
		return 99
	}
}

var (
	defaultRegistryOnce sync.Once
	defaultRegistry     *Registry
)

// DefaultRegistry returns the lazily-initialized singleton tool registry.
func DefaultRegistry() *Registry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewRegistry()
	})
	return defaultRegistry
}
