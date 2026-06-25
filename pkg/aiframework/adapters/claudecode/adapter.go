// Package claudecode is the P19 reference adapter that wires the Claude Code
// framework into the framework-agnostic aiframework interface layer.
//
// It proves the interface contracts are implementable for a real framework by
// delegating every operation to the concrete generators in
// addons/claudecode (GenerateSettings, GenerateMcpJson, CalculateContextBudget,
// and the detection/permission logic). It performs no generation of its own:
// it only translates framework-agnostic policy input into the inputs those
// real generators expect, and adapts their output back into the aiframework
// artifact types.
//
// The pkg/aiframework -> addons/claudecode import direction is deliberate and
// lint-legal: depguard's isolation rules are scoped to addon<->addon imports
// only, so a reference adapter living under pkg may import the addon it wraps.
package claudecode

import (
	ccaddon "github.com/Quantum-Serendipity/qsdev/addons/claudecode"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
	"github.com/Quantum-Serendipity/qsdev/pkg/ecosystem"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// Adapter implements the three P19 interfaces that a config-generating
// framework must satisfy: DetectionAdapter, ToolAdapter, and ConfigRenderer.
// A single value satisfies all three because they share FrameworkID() and the
// Claude Code generators are stateless beyond the addon config + registry.
type Adapter struct {
	cfg      ccaddon.Config
	registry *ecosystem.Registry
	addon    *ccaddon.Adapter
}

// New returns a reference Adapter wired to the given addon configuration and
// ecosystem registry. The registry may be nil; the underlying generators
// tolerate a nil registry (no ecosystem-specific deny rules are contributed).
func New(cfg ccaddon.Config, registry *ecosystem.Registry) *Adapter {
	return &Adapter{
		cfg:      cfg,
		registry: registry,
		addon:    ccaddon.New(cfg, registry),
	}
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

// rulePatterns extracts the non-empty Pattern strings from permission rules.
func rulePatterns(rules []aiframework.PermissionRule) []string {
	out := make([]string, 0, len(rules))
	for _, r := range rules {
		if r.Pattern != "" {
			out = append(out, r.Pattern)
		}
	}
	return out
}

// hookSpecsToChoices maps framework-agnostic hook specs onto the addon's
// HookChoices by matching each spec's Command against the known hook logic IDs.
func hookSpecsToChoices(specs []aiframework.HookSpec) types.HookChoices {
	var c types.HookChoices
	for _, s := range specs {
		switch aiframework.HookLogicID(s.Command) {
		case aiframework.LogicPackageGuard:
			c.SafetyBlock = true
		case aiframework.LogicCredentialScan:
			c.CredentialScan = true
		case aiframework.LogicDestructiveBlock:
			c.DestructivePrevention = true
		case aiframework.LogicFileBoundary:
			c.FileBoundary = true
		case aiframework.LogicToolGates:
			c.ToolGates = true
		}
	}
	return c
}
