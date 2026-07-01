// Package windsurf registers the Windsurf (Cascade) framework stub adapter for
// the universal qsdev MCP server (Phase 32, Unit 32.5). All behavior lives in the
// shared frameworkstub adapter; this package contributes only the Windsurf
// descriptor. It is registered into the adapter registry explicitly from
// cmd/qsdev/main.go.
//
// RESEARCH-GATED: full Windsurf config rendering is pending the
// gdev-universal-mcp-server-design spike and a Windsurf P19 reference adapter;
// the config tool returns an honest structured research_gated result.
// Project-marker detection, client-identity matching, the project summary, and
// the validated capabilities report are fully real.
package windsurf

import (
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/frameworkstub"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

// descriptor is the Windsurf manifest consumed by the shared framework stub.
// Windsurf's agent is named Cascade, so the client hints match both names. The
// tool ceiling (~100) is a client-side limit; Cascade has no native pre-tool
// hook, so the strongest enforcement qsdev can target is advisory.
var descriptor = frameworkstub.Descriptor{
	ID:               aiframework.Windsurf,
	Label:            "windsurf",
	Marker:           ".windsurfrules",
	MarkerIsDir:      false,
	ClientHints:      []string{"windsurf", "cascade"},
	ConfigConvention: ".windsurfrules (MDC) + Cascade context model",

	InfoDescription:         "Summarize the detected project (languages, ecosystems, container runtime, git state) framed for Windsurf, including whether Windsurf project markers are present and Windsurf's config convention (.windsurfrules (MDC) + Cascade context model).",
	ConfigDescription:       "Render the Windsurf configuration (.windsurfrules) from the project's qsdev policy. RESEARCH-GATED: full Windsurf config rendering is not yet implemented (no Windsurf P19 reference adapter exists); this returns an honest structured research_gated result describing what is pending.",
	CapabilitiesDescription: "Report Windsurf's validated capability profile: its client-side tool ceiling, configuration format (MDC), Cascade context model, native-hook support, and the strongest enforcement tier qsdev can target on Windsurf.",

	ConfigFormat: "MDC",
	ConfigPaths:  []string{".windsurfrules"},
	GatedReason:  "Full Windsurf configuration rendering is not yet implemented: it is gated on the gdev-universal-mcp-server-design research spike and a Windsurf P19 reference adapter (pkg/aiframework/adapters/windsurf), neither of which exists yet.",
	PendingP19:   "pkg/aiframework/adapters/windsurf",

	Capabilities: map[string]any{
		"framework":               string(aiframework.Windsurf),
		"tool_ceiling":            100,
		"tool_ceiling_documented": true,
		"tool_ceiling_kind":       "client-side", // enforced by the Windsurf client, not the MCP protocol
		"config_format":           "MDC",
		"config_paths":            []string{".windsurfrules"},
		"context_model":           "Cascade",
		"native_hooks":            false,
		"enforcement_tier":        aiframework.TierAdvisory.String(),
		"enforcement_note":        "Windsurf's Cascade agent has no native pre-tool hook; policy is advisory (instructions only).",
	},
	CapabilitiesText: "windsurf capabilities: tool ceiling 100 (client-side), MDC config, advisory enforcement",
}

// New constructs the Windsurf stub adapter from its descriptor.
func New() *frameworkstub.Adapter { return frameworkstub.New(descriptor) }
