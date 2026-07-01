// Package status implements the project-state MCP tool handlers for the
// universal qsdev server (Phase 32, Unit 32.9): qsdev_status (2-tier drift
// detection over the project's generated-file state and ecosystem detection) and
// qsdev_devenv_doctor (seven integrity checks run in parallel).
//
// qsdev_devenv_doctor is the integrity doctor and is deliberately distinct from
// qsdev_doctor (the system-prerequisite doctor contributed by the project
// context engine); the two cover different concerns and must not collide.
package status

import (
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// Pruning tiers: qsdev_status is fast-path critical; the doctor is standard.
const (
	tierCritical = 0
	tierStandard = 1
)

// Tools returns the two status tool registrations bound to projectRoot.
func Tools(projectRoot string) []spi.ToolRegistration {
	st := newStatusChecker(projectRoot)
	doc := newDoctorChecker(projectRoot)

	return []spi.ToolRegistration{
		{
			Name:        "qsdev_status",
			Description: "Report project drift with 2-tier detection. Tier 1 (fast, cached) returns the last snapshot when the generated-file state is unchanged; Tier 2 re-runs ecosystem detection and diffs it to surface added/removed ecosystems, changed tool versions, and configuration changes.",
			InputSchema: statusSchema(),
			Category:    middleware.CategoryStatus,
			Tier:        tierCritical,
			Handler:     st.handle,
		},
		{
			Name:        "qsdev_devenv_doctor",
			Description: "Run seven project-integrity checks in parallel (config validity, state integrity, tool availability, Nix install, MCP server health, hook deployment, permission consistency) with a 5s timeout, returning per-check pass/warning/fail and remediation.",
			InputSchema: doctorSchema(),
			Category:    middleware.CategoryDiagnostics,
			Tier:        tierStandard,
			Handler:     doc.handle,
		},
	}
}

func statusSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"tier": map[string]any{
				"type":        "string",
				"enum":        []any{"1", "2", "auto"},
				"description": "Force Tier 1 (cached) or Tier 2 (thorough). Default auto: Tier 1 when state is unchanged, else Tier 2.",
			},
		},
	}
}

func doctorSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"check": map[string]any{
				"type":        "string",
				"enum":        []any{"config", "state", "tools", "nix", "mcp", "hooks", "permissions"},
				"description": "Run only the named check. Omit to run all seven.",
			},
		},
	}
}
