// Package claudecode is the P19 reference adapter that wires the Claude Code
// framework into the framework-agnostic aiframework interface layer.
//
// It exposes the DetectionAdapter, ToolAdapter, and ConfigRenderer interfaces
// of the Claude Code addon (addons/claudecode.Adapter), which is the single
// canonical implementation: every operation delegates to it, so policy
// translation (hook invariants, permission rules, sandbox, MCP servers) has
// exactly one definition and cannot drift between two copies.
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
	addon *ccaddon.Adapter
}

// New returns a reference Adapter wired to the given addon configuration and
// ecosystem registry. The registry may be nil; the underlying generators
// tolerate a nil registry (no ecosystem-specific deny rules are contributed).
func New(cfg ccaddon.Config, registry *ecosystem.Registry) *Adapter {
	return &Adapter{addon: ccaddon.New(cfg, registry)}
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
