// Package cursor registers the Cursor framework stub adapter for the universal
// qsdev MCP server (Phase 32, Unit 32.5). All behavior lives in the shared
// frameworkstub adapter; this package contributes only the Cursor descriptor. It
// is registered into the adapter registry explicitly from cmd/qsdev/main.go.
//
// RESEARCH-GATED: full Cursor config rendering is pending the
// gdev-universal-mcp-server-design spike and a Cursor P19 reference adapter; the
// config tool returns an honest structured research_gated result. Project-marker
// detection, client-identity matching, the project summary, and the validated
// capabilities report are fully real.
package cursor

import (
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/frameworkstub"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

// descriptor is the Cursor manifest consumed by the shared framework stub. The
// Cursor tool ceiling (~40) is a client-side limit enforced by the Cursor
// client, not an MCP-protocol guarantee; Cursor has no native pre-tool hook, so
// the strongest enforcement qsdev can target is advisory, with external
// isolation as the fallback.
var descriptor = frameworkstub.Descriptor{
	ID:               aiframework.Cursor,
	Label:            "cursor",
	Marker:           ".cursor/rules",
	MarkerIsDir:      true,
	ClientHints:      []string{"cursor"},
	ConfigConvention: ".cursor/rules/*.mdc + .cursor/mcp.json",

	InfoDescription:         "Summarize the detected project (languages, ecosystems, container runtime, git state) framed for Cursor, including whether Cursor project markers are present and Cursor's config convention (.cursor/rules/*.mdc + .cursor/mcp.json).",
	ConfigDescription:       "Render the Cursor configuration (.cursor/rules/*.mdc and .cursor/mcp.json) from the project's qsdev policy. RESEARCH-GATED: full Cursor config rendering is not yet implemented (no Cursor P19 reference adapter exists); this returns an honest structured research_gated result describing what is pending.",
	CapabilitiesDescription: "Report Cursor's validated capability profile: its client-side tool ceiling, configuration format (MDC), native-hook support, and the strongest enforcement tier qsdev can target on Cursor.",

	ConfigFormat: "MDC",
	ConfigPaths:  []string{".cursor/rules/*.mdc", ".cursor/mcp.json"},
	GatedReason:  "Full Cursor configuration rendering is not yet implemented: it is gated on the gdev-universal-mcp-server-design research spike and a Cursor P19 reference adapter (pkg/aiframework/adapters/cursor), neither of which exists yet.",
	PendingP19:   "pkg/aiframework/adapters/cursor",

	Capabilities: map[string]any{
		"framework":               string(aiframework.Cursor),
		"tool_ceiling":            40,
		"tool_ceiling_documented": true,
		"tool_ceiling_kind":       "client-side", // enforced by the Cursor client, not the MCP protocol
		"config_format":           "MDC",
		"config_paths":            []string{".cursor/rules/*.mdc", ".cursor/mcp.json"},
		"native_hooks":            false,
		"enforcement_tier":        aiframework.TierAdvisory.String(),
		"enforcement_fallback":    aiframework.TierExternal.String(),
		"enforcement_note":        "Cursor has no native pre-tool hook; policy is advisory (instructions only), with qsdev external isolation as the fallback.",
	},
	CapabilitiesText: "cursor capabilities: tool ceiling 40 (client-side), MDC config, advisory enforcement",
}

// New constructs the Cursor stub adapter from its descriptor.
func New() *frameworkstub.Adapter { return frameworkstub.New(descriptor) }
