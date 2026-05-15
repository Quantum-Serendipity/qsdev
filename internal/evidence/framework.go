package evidence

import (
	"fmt"
	"sort"
	"sync"
)

// Framework defines a compliance framework with its controls.
type Framework struct {
	ID          string
	Name        string
	Version     string
	Description string
	Controls    func() []ControlDefinition
}

// ControlDefinition defines a single control within a framework,
// including how it maps to qsdev defense layers.
type ControlDefinition struct {
	ID                  string
	Name                string
	Desc                string
	Category            string
	Layers              []LayerMapping
	NotApplicableReason string
}

// LayerMapping describes the relationship between a qsdev defense layer
// and a compliance control.
type LayerMapping struct {
	LayerName   string
	Relevance   string // "primary"|"supporting"
	Description string
}

// FrameworkInfo is a read-only summary of a registered framework.
type FrameworkInfo struct {
	ID          string
	Name        string
	Version     string
	Description string
}

// FrameworkRegistry is a thread-safe registry of compliance frameworks.
type FrameworkRegistry struct {
	mu         sync.RWMutex
	frameworks map[string]Framework
}

// NewFrameworkRegistry creates an empty FrameworkRegistry.
func NewFrameworkRegistry() *FrameworkRegistry {
	return &FrameworkRegistry{
		frameworks: make(map[string]Framework),
	}
}

// Register adds a framework to the registry. Returns an error if a
// framework with the same ID is already registered.
func (r *FrameworkRegistry) Register(f Framework) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.frameworks[f.ID]; exists {
		return fmt.Errorf("framework %q is already registered", f.ID)
	}
	r.frameworks[f.ID] = f
	return nil
}

// Get retrieves a framework by ID. Returns the framework and true if
// found, or a zero Framework and false otherwise.
func (r *FrameworkRegistry) Get(id string) (*Framework, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f, ok := r.frameworks[id]
	if !ok {
		return nil, false
	}
	return &f, true
}

// List returns information about all registered frameworks, sorted by ID.
func (r *FrameworkRegistry) List() []FrameworkInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	infos := make([]FrameworkInfo, 0, len(r.frameworks))
	for _, f := range r.frameworks {
		infos = append(infos, FrameworkInfo{
			ID:          f.ID,
			Name:        f.Name,
			Version:     f.Version,
			Description: f.Description,
		})
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].ID < infos[j].ID
	})
	return infos
}

// DefaultRegistry returns a FrameworkRegistry pre-loaded with all
// built-in compliance frameworks (SOC2, HIPAA, ASVS).
func DefaultRegistry() *FrameworkRegistry {
	r := NewFrameworkRegistry()
	_ = r.Register(SOC2Framework())
	_ = r.Register(HIPAAFramework())
	_ = r.Register(ASVSFramework())
	return r
}
