package toolreg

import (
	"fmt"
	"log/slog"
	"sort"
	"sync"

	"github.com/Quantum-Serendipity/qsdev/internal/registry"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Registry is a thread-safe collection of Tool definitions.
type Registry struct {
	*registry.Registry[*Tool]
}

// NewRegistry creates an empty tool registry.
func NewRegistry() *Registry {
	return &Registry{
		Registry: registry.New[*Tool](
			registry.WithEntityName("tool"),
		),
	}
}

// Register adds a tool to the registry. Returns an error if a tool with
// the same name is already registered.
func (r *Registry) Register(t Tool) error {
	return r.Registry.Register(t.Name, &t)
}

// ByName returns the tool with the given name.
func (r *Registry) ByName(name string) (*Tool, bool) {
	return r.Get(name)
}

// All returns all registered tools sorted by category then name.
func (r *Registry) All() []*Tool {
	result := r.Values()
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
	var result []*Tool
	for _, t := range r.Values() {
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
	return r.Registry.Names()
}

// Count returns the number of registered tools.
func (r *Registry) Count() int {
	return r.Registry.Count()
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

// AttachBehavior attaches behavioral functions to a tool that was loaded
// from the catalog YAML. This is the second phase of two-phase registration:
// YAML provides declarative metadata, Go code provides function hooks.
// If the tool name is not in the registry, this is a no-op.
func (r *Registry) AttachBehavior(name string, b ToolBehavior) {
	found := r.Modify(name, func(t *Tool) *Tool {
		if b.EnableFunc != nil {
			t.EnableFunc = b.EnableFunc
		}
		if b.DisableFunc != nil {
			t.DisableFunc = b.DisableFunc
		}
		if b.DetectFunc != nil {
			t.DetectFunc = b.DetectFunc
		}
		if b.GenerateFunc != nil {
			t.GenerateFunc = b.GenerateFunc
		}
		if b.SharedContent != nil {
			if t.SharedContent == nil {
				t.SharedContent = make(map[SharedSection]SharedContentFunc)
			}
			for k, v := range b.SharedContent {
				t.SharedContent[k] = v
			}
		}
		if b.SectionDataFunc != nil {
			t.SectionDataFunc = b.SectionDataFunc
		}
		return t
	})
	if !found {
		slog.Warn("AttachBehavior called for unknown tool", "tool", name)
	}
}

// ToolBehavior holds the Go function fields for a tool. Used with
// AttachBehavior to separate declarative metadata (YAML) from
// behavioral hooks (Go code).
type ToolBehavior struct {
	EnableFunc      EnableFunc
	DisableFunc     DisableFunc
	DetectFunc      DetectFunc
	GenerateFunc    GenerateFunc
	SharedContent   map[SharedSection]SharedContentFunc
	SectionDataFunc SectionDataFunc
}

// BehaviorProvider attaches behavior to, or registers additional tools in, a
// freshly built default registry. Providers run while the default registry
// is being constructed, so they must not call Default or DefaultRegistry.
type BehaviorProvider func(r *Registry)

var (
	defaultMu          sync.Mutex
	defaultBuilt       bool
	defaultRegistryVal *Registry
	defaultRegistryErr error
	behaviorProviders  []BehaviorProvider
)

// RegisterBehaviors adds a provider that is applied to the default registry
// when it is built. Registering is side-effect free — the catalog is not
// loaded — so a package may call it while initializing without freezing the
// catalog before main has configured branding, and without turning a bad
// user config into a start-up panic. If the default registry has already
// been built, the provider is applied to it immediately.
func RegisterBehaviors(p BehaviorProvider) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	behaviorProviders = append(behaviorProviders, p)
	if defaultBuilt && defaultRegistryVal != nil {
		p(defaultRegistryVal)
	}
}

// Default returns the lazily-initialized singleton tool registry and any
// error that occurred during initialization. On first use the registry is
// built from the catalog, then the package's built-in behaviors and every
// provider passed to RegisterBehaviors are attached. Callers that can
// propagate errors should prefer this over DefaultRegistry.
func Default() (*Registry, error) {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	if !defaultBuilt {
		defaultRegistryVal, defaultRegistryErr = buildDefault()
		defaultBuilt = true
	}
	return defaultRegistryVal, defaultRegistryErr
}

// buildDefault constructs the default registry. The caller holds defaultMu.
func buildDefault() (*Registry, error) {
	r, err := BuildFromCatalogE()
	if err != nil {
		return nil, err
	}
	for name, b := range builtinBehaviors() {
		r.AttachBehavior(name, b)
	}
	for _, p := range behaviorProviders {
		p(r)
	}
	return r, nil
}

// DefaultRegistry returns the lazily-initialized singleton tool registry.
// It panics if the catalog fails to load. Callers that can propagate
// errors should prefer Default() instead.
func DefaultRegistry() *Registry {
	r, err := Default()
	if err != nil {
		panic(fmt.Sprintf("toolreg: failed to build registry: %v", err))
	}
	return r
}

// ResetDefaultRegistry clears the cached registry so the next Default call
// rebuilds it, re-applying every registered behavior provider. For testing
// only.
func ResetDefaultRegistry() {
	defaultMu.Lock()
	defer defaultMu.Unlock()
	defaultBuilt = false
	defaultRegistryVal = nil
	defaultRegistryErr = nil
}

// SharedSectionContent is one enabled tool's rendered section of a shared file.
type SharedSectionContent struct {
	Tool      *Tool
	SectionID string
	Content   []byte
}

// SharedSectionsFor renders the section every enabled tool contributes to the
// shared file at path, in registry order. Tools without content for that
// file (e.g. whose contribution a generator renders directly) are skipped.
func (r *Registry) SharedSectionsFor(path string, answers types.WizardAnswers) ([]SharedSectionContent, error) {
	var out []SharedSectionContent
	for _, t := range r.All() {
		if !answers.EnabledTools[t.Name] {
			continue
		}
		for _, f := range t.SharedFiles() {
			if f.Path != path {
				continue
			}
			fn, ok := t.SharedContent[SectionOf(f)]
			if !ok {
				continue
			}
			content, err := fn(answers)
			if err != nil {
				return nil, fmt.Errorf("rendering %s section %q of tool %q: %w", path, f.SectionID, t.Name, err)
			}
			out = append(out, SharedSectionContent{Tool: t, SectionID: f.SectionID, Content: content})
		}
	}
	return out, nil
}
