// Package devenv implements the development-environment MCP tool handlers for the
// universal qsdev server (Phase 32, Unit 32.9): env_info (PATH, listening-port,
// tool-catalog, and filtered-environment probing) and nix_run (executing a Nix
// package in a managed process group with a timeout).
//
// env_info never emits the values of sensitive environment variables, filtering
// them at the source in addition to the ContentSafety middleware. nix_run runs
// its target in a dedicated process group so a timeout kills the whole group,
// with an environment that withholds every variable env_info withholds (see
// childEnv), and is concurrency-limited (max 3) by the CategoryProcess rate
// limiter.
package devenv

import (
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// tierStandard mirrors projectctx's standard tool tier.
const tierStandard = 1

// Tools returns the devenv tool registrations bound to projectRoot. nixRun
// registers qsdev_nix_run; the serve command leaves it out in gateway mode
// unless the operator opts in.
func Tools(projectRoot string, nixRun bool) []spi.ToolRegistration {
	env := newEnvInfo()
	regs := []spi.ToolRegistration{
		{
			Name:        "qsdev_env_info",
			Description: "Probe the development environment: PATH composition (system/user/nix/tool-managed), listening TCP ports, the qsdev-managed tool catalog, and a filtered process-environment snapshot. Sensitive variable values are never emitted.",
			InputSchema: envInfoSchema(),
			Category:    middleware.CategoryEnvironment,
			Tier:        tierStandard,
			Annotations: spi.ReadOnlyAnnotations(false),
			Handler:     env.handle,
		},
	}
	if !nixRun {
		return regs
	}
	nix := newNixRunner(projectRoot)
	return append(regs, spi.ToolRegistration{
		Name:        "qsdev_nix_run",
		Description: "Execute a Nix package via `nix run <command> -- <args>` from the project root, in a dedicated process group with a timeout (default 30s, max 10m) and an environment stripped of credential-bearing variables. Remote flake references (URLs, github: and other schemes) and paths outside the project are rejected. Captures stdout, stderr (each capped at 1 MiB; excess is discarded and flagged *_truncated), exit code, and duration; on timeout the entire process group is killed. Limited to 3 concurrent executions.",
		InputSchema: nixRunSchema(),
		Category:    middleware.CategoryProcess,
		Tier:        tierStandard,
		Handler:     nix.handle,
	})
}

func envInfoSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"probe": map[string]any{
				"type":        "string",
				"enum":        []any{"path", "ports", "tools", "env", "all"},
				"description": "Which probe to run (default all).",
			},
		},
	}
}

func nixRunSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"command": map[string]any{"type": "string", "description": "Nix installable to run, e.g. nixpkgs#jq."},
			"args": map[string]any{
				"type":        "array",
				"items":       map[string]any{"type": "string"},
				"description": "Arguments passed to the program after `--`.",
			},
			"stdin":   map[string]any{"type": "string", "description": "Optional standard input piped to the program."},
			"timeout": map[string]any{"type": "string", "description": "Timeout as a Go duration (e.g. \"30s\") or seconds. Default 30s; values above 10m are clamped."},
		},
		"required": []any{"command"},
	}
}
