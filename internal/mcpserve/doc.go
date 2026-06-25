// Package mcpserve assembles qsdev's universal MCP server by wrapping the
// vendored mcp-go library (github.com/mark3labs/mcp-go). It exposes the
// `qsdev mcp serve` subcommand, constructs the underlying *server.MCPServer,
// resolves caller identity and project root, and bridges between the
// framework-neutral spi contracts and mcp-go's concrete protocol types.
//
// # Layering invariant (load-bearing)
//
// The root mcpserve package and everything it transitively imports stay free of
// addons/claudecode; all addon delegation is quarantined in mcpserve/adapters/*,
// blank-imported only from cmd/qsdev/main.go.
//
// This is required because the `serve` command is wired into the `qsdev mcp`
// command group, which lives in addons/claudecode. That makes addons/claudecode
// import internal/mcpserve. If mcpserve (or its spi sub-package, or any other
// transitive dependency) imported addons/claudecode in return, the build would
// contain an import cycle. Concrete framework adapters legitimately need to
// delegate to addons/claudecode, so they live in their own sub-packages under
// internal/mcpserve/adapters and self-register into spi.DefaultRegistry from the
// program entry point (cmd/qsdev/main.go) via blank imports — never from inside
// this package.
//
// # Protocol
//
// The server advertises MCP protocol revision 2025-11-25 (the latest stable
// revision known to the vendored mcp-go). Agent identity is resolved from the
// initialize handshake clientInfo, with an optional per-request _meta override
// under the reverse-DNS key com.quantumserendipity.qsdev/agentId.
package mcpserve
