// Package projectctx implements the project context engine for the universal
// qsdev MCP server (Phase 32, Unit 32.2). It aggregates qsdev's existing project
// introspection — ecosystem detection (internal/detect), generated-file state
// (internal/state), the tool lifecycle registry (internal/toolreg), and the MCP
// server registry (internal/mcpregistry) — and renders it as the generic
// (framework-agnostic) MCP tools, resources, and prompts that any connected AI
// framework can consume.
//
// The engine produces neutral spi registrations (internal/mcpserve/spi); the
// mcpserve.Server mounts them through its public MountProjectContext path. The
// engine never imports mcpserve, so it stays free of the mcp-go dependency and
// cannot create an import cycle.
//
// Graceful degradation: handlers that depend on configuration a project may not
// have yet (a missing .qsdev.yaml, an absent state file, the not-yet-built
// Unit 32.7 workspace graph) return a structured not_configured result with
// IsError set rather than failing. They never crash.
package projectctx

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/Quantum-Serendipity/qsdev/internal/detect"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpregistry"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/internal/state"
	"github.com/Quantum-Serendipity/qsdev/internal/toolreg"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
	"github.com/Quantum-Serendipity/qsdev/pkg/types"
)

// ProjectContext aggregates the project model the generic MCP surface exposes.
// It is constructed once at server startup against the resolved project root and
// is safe for concurrent reads (its handlers re-run detection on demand rather
// than mutating shared fields).
type ProjectContext struct {
	projectRoot string

	// detection is the detection result captured at construction. Handlers that
	// need a fresh view (qsdev_detect) re-run detection rather than reading this.
	detection types.DetectedProject

	// state is the primary generated-file state ledger, or an empty ledger when
	// the project has not been initialized yet.
	state     types.GeneratedState
	statePath string // absolute path to the primary state file (may not exist)

	toolReg *toolreg.Registry
	mcpReg  *mcpregistry.McpServerRegistry

	// workspace is the slot for the Unit 32.7 monorepo workspace graph. It is an
	// empty-interface placeholder that is always nil until Task T9 lands; while it
	// is nil the per-package context resource degrades to a structured
	// not_configured result.
	workspace any

	pruner *ToolPruner
}

// NewProjectContext builds the engine for projectRoot. Detection always runs;
// state and configuration are loaded best-effort so a project that has not been
// initialized still yields a usable engine (handlers then report not_configured
// where appropriate). It returns an error only for unrecoverable wiring problems
// (an empty root, or a tool catalog that fails to load).
func NewProjectContext(projectRoot string) (*ProjectContext, error) {
	if projectRoot == "" {
		return nil, fmt.Errorf("project context: project root is required")
	}

	detection := detect.Detect(projectRoot)

	statePath := filepath.Join(projectRoot, state.StateFilePaths()[0])
	st, err := state.LoadStateFromFile(statePath)
	if err != nil {
		// A malformed or unreadable state file must not crash the engine: degrade
		// to an empty ledger so introspection still works.
		slog.Warn("project context: state file unreadable; using empty state",
			"path", statePath, "error", err)
		st = types.GeneratedState{Files: map[string]types.FileState{}}
	}

	reg, err := toolreg.Default()
	if err != nil {
		return nil, fmt.Errorf("project context: loading tool registry: %w", err)
	}

	return &ProjectContext{
		projectRoot: projectRoot,
		detection:   detection,
		state:       st,
		statePath:   statePath,
		toolReg:     reg,
		mcpReg:      mcpregistry.DefaultRegistry(),
		pruner:      NewToolPruner(),
	}, nil
}

// ProjectRoot returns the resolved project root the engine operates within.
func (pc *ProjectContext) ProjectRoot() string { return pc.projectRoot }

// Detection returns the detection result captured at construction.
func (pc *ProjectContext) Detection() types.DetectedProject { return pc.detection }

// Pruner returns the engine's tool pruner (the tier-based ceiling mechanism).
func (pc *ProjectContext) Pruner() *ToolPruner { return pc.pruner }

// configFile returns the absolute path to the project's .qsdev.yaml.
func (pc *ProjectContext) configFile() string {
	return filepath.Join(pc.projectRoot, branding.Get().ConfigFile)
}

// localConfigFile returns the absolute path to the project's .qsdev.local.yaml.
func (pc *ProjectContext) localConfigFile() string {
	return filepath.Join(pc.projectRoot, branding.Get().LocalConfig)
}

// notConfiguredResult builds the canonical graceful-degradation tool result: a
// structured not_configured payload serialized into Text with IsError set.
//
// The payload is placed in Text (not Structured) deliberately: the mcpserve
// bridge marshals a result with a non-nil Structured field via mcp-go's
// structured-content path, which does not carry the IsError flag. Encoding the
// structured payload as JSON Text and leaving Structured nil makes the bridge
// emit a real protocol error result, so MCP clients see IsError=true exactly as
// the graceful-degradation contract requires.
func notConfiguredResult(reason string, extra map[string]any) *spi.ToolResult {
	payload := map[string]any{"status": "not_configured", "reason": reason}
	for k, v := range extra {
		payload[k] = v
	}
	text, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return &spi.ToolResult{Text: "not_configured: " + reason, IsError: true}
	}
	return &spi.ToolResult{Text: string(text), IsError: true}
}

// boolArg extracts an optional boolean argument, defaulting to false when absent
// or of the wrong type. JSON decoding yields a bool for JSON booleans.
func boolArg(args map[string]any, name string) bool {
	v, _ := args[name].(bool)
	return v
}
