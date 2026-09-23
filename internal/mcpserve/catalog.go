package mcpserve

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/registry"
)

// genericOwner is the sentinel owner recorded for tools and resources that are
// not contributed by a framework adapter (the generic project-context surface).
// Tools owned by it are ALWAYS visible regardless of the connected client.
const genericOwner = "generic"

// catalog records, for the composite tool/resource/prompt surface mounted on
// the server, which owner each tool name, resource URI, and prompt name belongs
// to — a framework id string for adapter contributions, or genericOwner for the
// generic project-context surface. It is populated at mount time and read by the
// per-request tool filter to decide tool visibility.
//
// Storage and concurrency-safety come from registry.Registry[string], one
// instance per surface: mounting happens at construction (single goroutine)
// but filtering happens at request time on possibly concurrent goroutines,
// and the registry guards both with its own RWMutex.
//
// Collision policy (see claim): among adapters the first registration wins and
// any later collision is reported as an error. Generic names are reserved,
// however: a generic registration reclaims a key an adapter already holds, so an
// adapter mounted earlier (adapters mount at construction, before the generic
// surfaces) can never shadow a generic tool and inherit the policy keyed on its
// name — e.g. the credential-vend redaction exemption.
type catalog struct {
	tools     *registry.Registry[string]
	resources *registry.Registry[string]
	prompts   *registry.Registry[string]
}

// newCatalog returns an empty catalog.
func newCatalog() *catalog {
	return &catalog{
		tools:     registry.New[string](registry.WithEntityName("tool")),
		resources: registry.New[string](registry.WithEntityName("resource")),
		prompts:   registry.New[string](registry.WithEntityName("prompt")),
	}
}

// addTool records that the tool named name is owned by owner. It returns the
// adapter owner a generic registration displaced (empty when none), or an error
// when name is already recorded and cannot be reclaimed — leaving the first
// registration in place so callers can skip the colliding tool.
func (c *catalog) addTool(name, owner string) (string, error) {
	return claim(c.tools, "tool name", name, owner)
}

// addResource records that the resource URI is owned by owner, with the same
// collision policy as addTool.
func (c *catalog) addResource(uri, owner string) (string, error) {
	return claim(c.resources, "resource URI", uri, owner)
}

// addPrompt records that the prompt named name is owned by owner, with the same
// collision policy as addTool.
func (c *catalog) addPrompt(name, owner string) (string, error) {
	return claim(c.prompts, "prompt name", name, owner)
}

// claim registers key under owner in r. On a collision a generic owner
// reclaims the key from an adapter (returning the displaced adapter owner);
// every other collision is an error that leaves the existing owner in place.
func claim(r *registry.Registry[string], kind, key, owner string) (string, error) {
	err := r.Register(key, owner)
	if err == nil {
		return "", nil
	}
	existing, _ := r.Get(key)
	if owner == genericOwner && existing != genericOwner {
		r.Delete(key)
		if rerr := r.Register(key, owner); rerr != nil {
			return "", fmt.Errorf("reclaiming %s %q from %q: %w", kind, key, existing, rerr)
		}
		return existing, nil
	}
	return "", fmt.Errorf("duplicate %s %q: already owned by %q, cannot also assign to %q: %w", kind, key, existing, owner, err)
}

// toolOwnerOf returns the recorded owner of the named tool. The boolean reports
// whether the tool was tracked at mount time.
func (c *catalog) toolOwnerOf(name string) (string, bool) {
	return c.tools.Get(name)
}
