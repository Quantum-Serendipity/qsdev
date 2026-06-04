package claudecode

import (
	claudecodeaddon "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// Adapter implements all seven aiframework interfaces for Claude Code.
type Adapter struct {
	cfg      claudecodeaddon.Config
	registry *ecosystem.Registry
}

// New returns an Adapter wired to the given addon config and ecosystem registry.
func New(cfg claudecodeaddon.Config, registry *ecosystem.Registry) *Adapter {
	return &Adapter{cfg: cfg, registry: registry}
}

var (
	_ aiframework.DetectionAdapter = (*Adapter)(nil)
	_ aiframework.ConfigRenderer   = (*Adapter)(nil)
	_ aiframework.HookDeployer     = (*Adapter)(nil)
	_ aiframework.RegistryClient   = (*Adapter)(nil)
	_ aiframework.MetricsProvider  = (*Adapter)(nil)
	_ aiframework.StateBackend     = (*Adapter)(nil)
	_ aiframework.ToolAdapter      = (*Adapter)(nil)
)
