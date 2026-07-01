// Package devenv implements the development-environment MCP tool handlers for the
// universal qsdev server (Phase 32, Unit 32.9): env_info (PATH, listening-port,
// tool-catalog, and filtered-environment probing) and nix_run (executing a Nix
// package in a managed process group with a timeout).
//
// env_info never emits the values of sensitive environment variables, filtering
// them at the source in addition to the ContentSafety middleware. nix_run runs
// its target in a dedicated process group so a timeout kills the whole group,
// and is concurrency-limited (max 3) by the CategoryProcess rate limiter.
package devenv

import (
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/middleware"
	"github.com/Quantum-Serendipity/qsdev/internal/mcpserve/spi"
)

// tierStandard mirrors projectctx's standard pruning tier.
const tierStandard = 1

// Tools returns the two devenv tool registrations bound to projectRoot.
func Tools(projectRoot string) []spi.ToolRegistration {
	env := newEnvInfo(projectRoot)
	nix := newNixRunner()

	return []spi.ToolRegistration{
		{
			Name:        "qsdev_env_info",
			Description: "Probe the development environment: PATH composition (system/user/nix/tool-managed), listening TCP ports, the qsdev-managed tool catalog, and a filtered process-environment snapshot. Sensitive variable values are never emitted.",
			InputSchema: envInfoSchema(),
			Category:    middleware.CategoryEnvironment,
			Tier:        tierStandard,
			Handler:     env.handle,
		},
		{
			Name:        "qsdev_nix_run",
			Description: "Execute a Nix package via `nix run <command> -- <args>` in a dedicated process group with a timeout (default 30s). Captures stdout, stderr, exit code, and duration; on timeout the entire process group is killed. Limited to 3 concurrent executions.",
			InputSchema: nixRunSchema(),
			Category:    middleware.CategoryProcess,
			Tier:        tierStandard,
			Handler:     nix.handle,
		},
	}
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
			"timeout": map[string]any{"type": "string", "description": "Timeout as a Go duration (e.g. \"30s\") or seconds. Default 30s."},
		},
		"required": []any{"command"},
	}
}
