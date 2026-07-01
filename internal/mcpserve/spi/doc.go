// Package spi defines the framework-neutral service-provider interface for the
// universal MCP server.
//
// It is a LEAF contracts package: it depends only on the standard library plus
// the dependency-light internal/registry helper and pkg/aiframework's
// FrameworkID. It MUST NOT import the vendored mcp-go library, any addon, or the
// root mcpserve assembly package. Keeping spi free of mcp-go is what lets the
// middleware chain, tool registrations, and adapter contracts be unit-tested
// without a live MCP transport, and lets concrete framework adapters depend on
// these contracts without pulling in protocol machinery.
//
// The root mcpserve package bridges between these neutral types and the
// concrete mcp-go types (see mcpserve/bridge.go).
package spi
