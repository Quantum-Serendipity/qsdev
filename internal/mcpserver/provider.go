package mcpserver

import (
	"context"
	"sort"
	"sync"
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

type Registry struct {
	mu        sync.RWMutex
	providers map[string]Provider
}

func NewRegistry() *Registry {
	return &Registry{providers: make(map[string]Provider)}
}

func (r *Registry) Register(p Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.providers[p.Name()] = p
}

func (r *Registry) Get(name string) (Provider, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.providers[name]
	return p, ok
}

func (r *Registry) All() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.providers))
	for name := range r.providers {
		names = append(names, name)
	}
	sort.Strings(names)
	result := make([]Provider, 0, len(names))
	for _, name := range names {
		result = append(result, r.providers[name])
	}
	return result
}

var defaultRegistry = NewRegistry()

func DefaultRegistry() *Registry {
	return defaultRegistry
}
