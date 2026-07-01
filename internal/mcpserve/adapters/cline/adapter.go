// Package cline registers the Cline (Continue.dev family) framework stub adapter
// for the universal qsdev MCP server (Phase 32, Unit 32.5). All behavior lives in
// the shared frameworkstub adapter; this package contributes only the Cline
// descriptor. It is registered into the adapter registry explicitly from
// cmd/qsdev/main.go.
//
// Identity note: Cline belongs to the Continue.dev family, so its FrameworkID is
// aiframework.ContinueDev ("continue"). Because the default client-match token
// derived from that id ("continue") would not match a client reporting itself as
// "cline", the descriptor's client hints match both "cline" and "continue".
//
// RESEARCH-GATED: full Cline config rendering is pending the
// gdev-universal-mcp-server-design spike and a Continue.dev P19 reference
// adapter; the config tool returns an honest structured research_gated result.
package cline

import (
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/adapters/frameworkstub"
	"github.com/Quantum-Serendipity/qsdev/pkg/aiframework"
)

// descriptor is the Cline manifest consumed by the shared framework stub. Cline
// has no documented tool ceiling, stores configuration as YAML under .continue/,
// and has no native pre-tool hook, so the strongest enforcement qsdev can target
// is advisory.
var descriptor = frameworkstub.Descriptor{
	ID:               aiframework.ContinueDev,
	Label:            "cline",
	Marker:           ".continue",
	MarkerIsDir:      true,
	ClientHints:      []string{"cline", "continue"},
	ConfigConvention: ".continue/ (YAML config, Continue.dev family)",

	InfoDescription:         "Summarize the detected project (languages, ecosystems, container runtime, git state) framed for Cline, including whether Cline project markers are present and Cline's config convention (.continue/ (YAML config, Continue.dev family)).",
	ConfigDescription:       "Render the Cline configuration (.continue/ YAML) from the project's qsdev policy. RESEARCH-GATED: full Cline config rendering is not yet implemented (no Continue.dev P19 reference adapter exists); this returns an honest structured research_gated result describing what is pending.",
	CapabilitiesDescription: "Report Cline's validated capability profile: its (undocumented) tool ceiling, configuration format (YAML), Continue.dev family lineage, native-hook support, and the strongest enforcement tier qsdev can target on Cline.",

	ConfigFormat: "YAML",
	ConfigPaths:  []string{".continue/"},
	GatedReason:  "Full Cline configuration rendering is not yet implemented: it is gated on the gdev-universal-mcp-server-design research spike and a Continue.dev P19 reference adapter (pkg/aiframework/adapters/continue), neither of which exists yet.",
	PendingP19:   "pkg/aiframework/adapters/continue",

	Capabilities: map[string]any{
		"framework":               string(aiframework.ContinueDev),
		"family":                  "Continue.dev",
		"tool_ceiling":            0,
		"tool_ceiling_documented": false,
		"config_format":           "YAML",
		"config_paths":            []string{".continue/"},
		"native_hooks":            false,
		"enforcement_tier":        aiframework.TierAdvisory.String(),
		"enforcement_note":        "Cline has no native pre-tool hook; policy is advisory (instructions only).",
	},
	CapabilitiesText: "cline capabilities: tool ceiling undocumented, YAML config, advisory enforcement (Continue.dev family)",
}

// New constructs the Cline stub adapter from its descriptor.
func New() *frameworkstub.Adapter { return frameworkstub.New(descriptor) }
