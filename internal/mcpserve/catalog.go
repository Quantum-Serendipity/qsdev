package mcpserve

import (
	"fmt"
	"sync"
)

// genericOwner is the sentinel owner recorded for tools and resources that are
// not contributed by a framework adapter (the generic project-context surface).
// Tools owned by it are ALWAYS visible regardless of the connected client.
const genericOwner = "generic"

// catalog records, for the composite tool/resource surface mounted on the
// server, which owner each tool name and resource URI belongs to — a framework
// id string for adapter contributions, or genericOwner for the generic
// project-context surface. It is populated at mount time and read by the
// per-request tool filter to decide tool visibility.
//
// All access is mutex-guarded: mounting happens at construction (single
// goroutine) but filtering happens at request time on possibly concurrent
// goroutines, both touching the same maps.
type catalog struct {
	mu        sync.RWMutex
	toolOwner map[string]string
	resOwner  map[string]string
}

// newCatalog returns an empty catalog.
func newCatalog() *catalog {
	return &catalog{
		toolOwner: make(map[string]string),
		resOwner:  make(map[string]string),
	}
}

// addTool records that the tool named name is owned by owner. It returns an
// error when name is already recorded — a duplicate tool name across the
// composite surface — leaving the first registration in place so callers can
// skip the colliding tool rather than overwrite the original.
func (c *catalog) addTool(name, owner string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if existing, ok := c.toolOwner[name]; ok {
		return fmt.Errorf("duplicate tool name %q: already owned by %q, cannot also assign to %q", name, existing, owner)
	}
	c.toolOwner[name] = owner
	return nil
}

// addResource records that the resource URI is owned by owner. It returns an
// error when uri is already recorded, leaving the first registration in place.
func (c *catalog) addResource(uri, owner string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if existing, ok := c.resOwner[uri]; ok {
		return fmt.Errorf("duplicate resource URI %q: already owned by %q, cannot also assign to %q", uri, existing, owner)
	}
	c.resOwner[uri] = owner
	return nil
}

// toolOwnerOf returns the recorded owner of the named tool. The boolean reports
// whether the tool was tracked at mount time.
func (c *catalog) toolOwnerOf(name string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	owner, ok := c.toolOwner[name]
	return owner, ok
}
