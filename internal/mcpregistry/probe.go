package mcpregistry

import (
	"strings"

	"github.com/Quantum-Serendipity/qsdev/internal/mcphealth"
	"github.com/Quantum-Serendipity/qsdev/pkg/branding"
)

// ProbeSkipReason reports why a live health probe must not start cfg, or ""
// when starting it is safe. Health probes run on behalf of diagnostics (often
// triggered by an agent tool call), so they must never download and execute a
// package: a package launcher fetches whatever version is currently published,
// outside the package guard. The launcher is detected whether it is the command
// itself or is started through a wrapper (`cmd /c npx ...`, `sh -c "uvx ..."`,
// `env npx ...`). A probe must also not spawn another copy of the qsdev MCP
// server that may be answering the very call doing the probing.
func ProbeSkipReason(cfg mcphealth.ServerConfig) string {
	switch {
	case cfg.URL != "":
		return ""
	case cfg.Command == "":
		return "no command configured"
	}
	if launcher := networkLauncherIn(cfg.Command, cfg.Args); launcher != "" {
		return "package launcher " + launcher + " would download and run the package"
	}
	if isSelfServer(cfg) {
		return "this qsdev MCP server"
	}
	return ""
}

// networkLauncherIn returns the name of the package launcher the invocation
// runs (including subcommand and env-wrapped launchers), or of the first
// always-fetching launcher found in any whitespace-separated word of the
// command or its arguments (a shell or cmd /c wrapper), or "".
func networkLauncherIn(command string, args []string) string {
	if launcherFetches(command, args) {
		return commandName(command)
	}
	words := append([]string{command}, args...)
	for _, w := range words {
		for _, field := range strings.Fields(w) {
			if LaunchesFromNetwork(field) {
				return commandName(field)
			}
		}
	}
	return ""
}

// isSelfServer reports whether cfg launches qsdev's own MCP server
// (`qsdev mcp serve ...`).
func isSelfServer(cfg mcphealth.ServerConfig) bool {
	return commandName(cfg.Command) == branding.Get().AppName &&
		len(cfg.Args) >= 2 && cfg.Args[0] == "mcp" && cfg.Args[1] == "serve"
}
