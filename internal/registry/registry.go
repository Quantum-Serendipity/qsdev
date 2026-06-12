// Package registry provides a generic, thread-safe, keyed collection.
//
// It eliminates the duplicated sync.RWMutex+map[string]T boilerplate that
// appears across many qsdev packages. Domain-specific registries embed
// *Registry[T] and add any extra methods they need.
package registry

import (
	"fmt"
	"sort"
	"sync"
)

// DuplicatePolicy controls what happens when Register is called with a key
// that already exists in the registry.
type DuplicatePolicy int

const (
	// DenyDuplicates returns an error on duplicate registration (default).
	DenyDuplicates DuplicatePolicy = iota

	// AllowOverwrite silently replaces the existing entry.
	AllowOverwrite
)

// Option configures a Registry at construction time.
type Option func(*config)

type config struct {
	dupPolicy  DuplicatePolicy
	keepOrder  bool
	entityName string
}

// WithDuplicatePolicy sets the duplicate-key handling strategy.
func WithDuplicatePolicy(p DuplicatePolicy) Option {
	return func(c *config) { c.dupPolicy = p }
}

// WithInsertionOrder preserves the order in which keys were registered.
// Names() and Keys() will return keys in insertion order instead of sorted.
func WithInsertionOrder() Option {
	return func(c *config) { c.keepOrder = true }
}

// WithEntityName sets the name used in error messages (e.g. "tool", "provider").
// Defaults to "item".
func WithEntityName(name string) Option {
	return func(c *config) { c.entityName = name }
}

// Registry is a generic, thread-safe, keyed collection.
type Registry[T any] struct {
	mu    sync.RWMutex
	items map[string]T
	order []string // populated only when keepOrder is true
	cfg   config
}

// New creates an empty Registry with the given options.
func New[T any](opts ...Option) *Registry[T] {
	c := config{entityName: "item"}
	for _, o := range opts {
		o(&c)
	}
	return &Registry[T]{
		items: make(map[string]T),
		cfg:   c,
	}
}

// Register adds an item under the given key. Behaviour on duplicate keys is
// controlled by the DuplicatePolicy set at construction time.
func (r *Registry[T]) Register(key string, item T) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.items[key]; exists {
		if r.cfg.dupPolicy == DenyDuplicates {
			return fmt.Errorf("%s %q already registered", r.cfg.entityName, key)
		}
		// AllowOverwrite: replace value, don't touch order slice.
		r.items[key] = item
		return nil
	}
	r.items[key] = item
	if r.cfg.keepOrder {
		r.order = append(r.order, key)
	}
	return nil
}

// Get returns the item for key and whether it was found.
func (r *Registry[T]) Get(key string) (T, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	item, ok := r.items[key]
	return item, ok
}

// Names returns all registered keys. When WithInsertionOrder was used the
// keys come back in insertion order; otherwise they are sorted alphabetically.
func (r *Registry[T]) Names() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.cfg.keepOrder {
		out := make([]string, len(r.order))
		copy(out, r.order)
		return out
	}

	names := make([]string, 0, len(r.items))
	for k := range r.items {
		names = append(names, k)
	}
	sort.Strings(names)
	return names
}

// All returns a shallow copy of the internal map. Callers that need a
// particular ordering should sort the returned values themselves.
func (r *Registry[T]) All() map[string]T {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make(map[string]T, len(r.items))
	for k, v := range r.items {
		out[k] = v
	}
	return out
}

// Count returns the number of registered items.
func (r *Registry[T]) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.items)
}

// Range calls fn for each registered item while holding the read lock.
// If fn returns false the iteration stops early.
func (r *Registry[T]) Range(fn func(key string, item T) bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for k, v := range r.items {
		if !fn(k, v) {
			return
		}
	}
}

// Modify looks up the item for key and, if found, calls fn while holding
// the write lock. This is useful when T is a pointer type and the caller
// needs to mutate the pointed-to value atomically. Returns false if key
// was not found.
func (r *Registry[T]) Modify(key string, fn func(item T)) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	item, ok := r.items[key]
	if !ok {
		return false
	}
	fn(item)
	return true
}

// Delete removes the item for key. Returns true if the item existed.
func (r *Registry[T]) Delete(key string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.items[key]; !ok {
		return false
	}
	delete(r.items, key)
	if r.cfg.keepOrder {
		for i, k := range r.order {
			if k == key {
				r.order = append(r.order[:i], r.order[i+1:]...)
				break
			}
		}
	}
	return true
}
