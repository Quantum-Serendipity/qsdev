package mcpserve

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/registry"
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
// Storage and concurrency-safety come from registry.Registry[string], one
// instance per surface: mounting happens at construction (single goroutine)
// but filtering happens at request time on possibly concurrent goroutines,
// and the registry guards both with its own RWMutex. Each registry uses the
// default DenyDuplicates policy so the first registration of a name/URI wins
// and any later collision is reported as an error.
type catalog struct {
	tools     *registry.Registry[string]
	resources *registry.Registry[string]
}

// newCatalog returns an empty catalog.
func newCatalog() *catalog {
	return &catalog{
		tools:     registry.New[string](registry.WithEntityName("tool")),
		resources: registry.New[string](registry.WithEntityName("resource")),
	}
}

// addTool records that the tool named name is owned by owner. It returns an
// error when name is already recorded — a duplicate tool name across the
// composite surface — leaving the first registration in place so callers can
// skip the colliding tool rather than overwrite the original.
func (c *catalog) addTool(name, owner string) error {
	if err := c.tools.Register(name, owner); err != nil {
		existing, _ := c.tools.Get(name)
		return fmt.Errorf("duplicate tool name %q: already owned by %q, cannot also assign to %q: %w", name, existing, owner, err)
	}
	return nil
}

// addResource records that the resource URI is owned by owner. It returns an
// error when uri is already recorded, leaving the first registration in place.
func (c *catalog) addResource(uri, owner string) error {
	if err := c.resources.Register(uri, owner); err != nil {
		existing, _ := c.resources.Get(uri)
		return fmt.Errorf("duplicate resource URI %q: already owned by %q, cannot also assign to %q: %w", uri, existing, owner, err)
	}
	return nil
}

// toolOwnerOf returns the recorded owner of the named tool. The boolean reports
// whether the tool was tracked at mount time.
func (c *catalog) toolOwnerOf(name string) (string, bool) {
	return c.tools.Get(name)
}
