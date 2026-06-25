// Package container holds the Docker-deployment surface of the universal qsdev
// MCP server (Phase 32, Unit 32.10): the Gateway enforcement interceptor, the
// deploy-mode selection logic, and the container-config generator consumed by
// `qsdev update`.
//
// It is a leaf package: it depends only on the neutral spi types, the built-in
// middleware layers it REUSES (Guardrail, RateLimit, ContentSafety via
// middleware.DefaultChain), and pkg/aiframework for framework identity and
// enforcement tiers. It never imports the mcpserve server root, so it cannot
// create an import cycle; command.go imports this package, not the reverse.
//
// The three deployment modes are:
//
//   - native     — qsdev installed locally; `qsdev mcp serve` runs as a process
//     over stdio with the standard middleware chain (the default).
//   - gateway    — the container runs as an enforcing MCP proxy. GatewayChain
//     wraps the standard chain with an outer authentication layer and applies
//     stricter per-category rate limits, extending policy enforcement to
//     frameworks that lack native pre-tool hooks.
//   - standalone — the container runs with an explicitly mounted project root
//     (CWD is meaningless in a container) and exposes an HTTP /health endpoint
//     for orchestration.
package container
