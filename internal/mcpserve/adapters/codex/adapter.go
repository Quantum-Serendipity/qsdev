// Package codex registers the Codex framework stub adapter for the universal
// qsdev MCP server (Phase 32, Unit 32.5). All behavior lives in the shared
// frameworkstub adapter; this package contributes only the Codex descriptor and
// self-registers it into spi.DefaultRegistry() from init(). It is blank-imported
// only from cmd/qsdev/main.go.
//
// Enforcement note: Codex ships a native sandbox, so its enforcement tier is
// kernel — the strongest qsdev models. That sandbox is a client-side guarantee
// enforced by the Codex runtime, NOT a property of the MCP protocol; the
// capabilities profile records this distinction explicitly.
//
// RESEARCH-GATED: full Codex config rendering is pending the
// gdev-universal-mcp-server-design spike and a Codex P19 reference adapter; the
// config tool returns an honest structured research_gated result.
package codex

import (
	"fmt"

	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/frameworkstub"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

// descriptor is the Codex manifest consumed by the shared framework stub. Codex
// has no documented tool ceiling, uses a Starlark hook/policy format under
// .codex/, and ships a native sandbox, so its enforcement tier is kernel — but
// that sandbox is the client's, not an MCP-protocol guarantee, which
// enforcement_scope records explicitly.
var descriptor = frameworkstub.Descriptor{
	ID:               aiframework.Codex,
	Label:            "codex",
	Marker:           ".codex",
	MarkerIsDir:      true,
	ClientHints:      []string{"codex"},
	ConfigConvention: ".codex/ (Starlark hook/policy format, native sandbox)",

	InfoDescription:         "Summarize the detected project (languages, ecosystems, container runtime, git state) framed for Codex, including whether Codex project markers are present and Codex's config convention (.codex/ (Starlark hook/policy format, native sandbox)).",
	ConfigDescription:       "Render the Codex configuration (.codex/ Starlark hooks/policy) from the project's qsdev policy. RESEARCH-GATED: full Codex config rendering is not yet implemented (no Codex P19 reference adapter exists); this returns an honest structured research_gated result describing what is pending.",
	CapabilitiesDescription: "Report Codex's validated capability profile: its (undocumented) tool ceiling, configuration format (Starlark), native sandbox, and the strongest enforcement tier qsdev can target on Codex (kernel — a client-side sandbox guarantee, not an MCP-protocol one).",

	ConfigFormat: "Starlark",
	ConfigPaths:  []string{".codex/"},
	GatedReason:  "Full Codex configuration rendering is not yet implemented: it is gated on the gdev-universal-mcp-server-design research spike and a Codex P19 reference adapter (pkg/aiframework/adapters/codex), neither of which exists yet.",
	PendingP19:   "pkg/aiframework/adapters/codex",

	Capabilities: map[string]any{
		"framework":               string(aiframework.Codex),
		"tool_ceiling":            0,
		"tool_ceiling_documented": false,
		"config_format":           "Starlark",
		"config_paths":            []string{".codex/"},
		"native_hooks":            true,
		"native_sandbox":          true,
		"enforcement_tier":        aiframework.TierKernel.String(),
		"enforcement_scope":       "client-side sandbox enforced by the Codex runtime, not an MCP-protocol guarantee",
		"enforcement_note":        "Codex's native sandbox physically constrains tool actions (kernel tier), but this is the client's sandbox, not an enforcement the MCP protocol itself provides.",
	},
	CapabilitiesText: "codex capabilities: Starlark config, kernel enforcement (client-side sandbox enforced by the Codex runtime, not an MCP-protocol guarantee)",
}

// New constructs the Codex stub adapter from its descriptor.
func New() *frameworkstub.Adapter { return frameworkstub.New(descriptor) }

// init self-registers the singleton into the default adapter registry. A
// duplicate-id error can only mean the package was linked twice — a build error
// worth surfacing loudly.
func init() {
	if err := spi.DefaultRegistry().Register(New()); err != nil {
		panic(fmt.Sprintf("registering codex framework adapter: %v", err))
	}
}
