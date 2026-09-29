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
// imported only from instance/runtime.go.
//
// This is required because the `serve` command is wired into the `qsdev mcp`
// command group, which lives in addons/claudecode. That makes addons/claudecode
// import internal/mcpserve. If mcpserve (or its spi sub-package, or any other
// transitive dependency) imported addons/claudecode in return, the build would
// contain an import cycle. Concrete framework adapters legitimately need to
// delegate to addons/claudecode, so they live in their own sub-packages under
// internal/mcpserve/adapters. Each exposes a New constructor and performs no
// init()-time self-registration; the program's runtime wiring
// (instance.RegisterFrameworkAdapters, installed by instance.Main) registers
// every adapter in adapters.All explicitly into spi.DefaultRegistry — never
// this package. A new adapter is added to that list.
//
// # Protocol
//
// The server advertises MCP protocol revision 2025-11-25 (the latest stable
// revision known to the vendored mcp-go). Agent identity (spi.ToolCallContext
// AgentID) resolves, in precedence order, to: the verified transport identity
// (an mTLS client-certificate CN/SAN, which nothing else can override), else the
// optional per-request _meta override under the reverse-DNS key
// com.quantumserendipity.qsdev/agentId, else the initialize handshake's
// clientInfo name, else "unknown". Only the verified identity is trustworthy; a
// verified certificate that yields no usable name is rejected at the transport.
// Per-caller state such as rate limiting is keyed on the transport-stable
// Principal, never on the self-asserted _meta override.
package mcpserve
