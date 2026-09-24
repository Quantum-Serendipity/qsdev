// Package claudecode is the P19 reference adapter that wires the Claude Code
// framework into the framework-agnostic aiframework interface layer.
//
// It is the single aiframework implementation for Claude Code: detection,
// config rendering and policy translation live here and build on the addon's
// exported generators (GenerateSettings, GenerateMcpJson,
// CalculateContextBudget), so hook invariants, permission rules, sandbox and
// MCP servers have exactly one definition.
//
// The adapter implements only the interfaces Claude Code genuinely supports:
// DetectionAdapter, ConfigRenderer and ToolAdapter. It deliberately does not
// implement HookDeployer, RegistryClient, MetricsProvider or StateBackend:
// a stub that reported success (a permanently healthy metrics report, a no-op
// undeploy) would be trusted by any caller that wired it in.
//
// The pkg/aiframework -> addons/claudecode import direction is deliberate and
// lint-legal: depguard's isolation rules are scoped to addon<->addon imports
// only, so a reference adapter living under pkg may import the addon it wraps.
package claudecode

import (
	ccaddon "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
)

// Adapter implements the three P19 interfaces that a config-generating
// framework must satisfy: DetectionAdapter, ToolAdapter, and ConfigRenderer.
// A single value satisfies all three because they share FrameworkID() and the
// Claude Code generators are stateless beyond the addon config + registry.
type Adapter struct {
	cfg      ccaddon.Config
	registry *ecosystem.Registry
}

// New returns a reference Adapter wired to the given addon configuration and
// ecosystem registry. The registry may be nil; the underlying generators
// tolerate a nil registry (no ecosystem-specific deny rules are contributed).
func New(cfg ccaddon.Config, registry *ecosystem.Registry) *Adapter {
	return &Adapter{cfg: cfg, registry: registry}
}

var (
	_ aiframework.DetectionAdapter = (*Adapter)(nil)
	_ aiframework.ToolAdapter      = (*Adapter)(nil)
	_ aiframework.ConfigRenderer   = (*Adapter)(nil)
)

// FrameworkID identifies this adapter as the Claude Code framework. It is the
// single method shared by the DetectionAdapter, ToolAdapter, and
// ConfigRenderer interfaces.
func (a *Adapter) FrameworkID() aiframework.FrameworkID { return aiframework.ClaudeCode }
