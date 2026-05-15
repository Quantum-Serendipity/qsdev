# Sense internal/mcpserver/server.go Overview
- **Source**: https://raw.githubusercontent.com/luuuc/sense/main/internal/mcpserver/server.go
- **Retrieved**: 2026-05-15
- **Note**: WebFetch returned a summary rather than verbatim code. Key details extracted below.

---

## Package Purpose
Exposes graph, blast, and status tools over the Model Context Protocol (MCP), built on the mark3labs/mcp-go SDK.

## Main Functions
- `Run()` and `RunWithOptions()` - Start the MCP stdio server
- `buildMCPServer()` - Initialize server with handlers and cleanup

## Tool Handlers
1. `handleSearch()` - Semantic and keyword symbol matching across indexed code
2. `handleGraph()` - Structural relationships (callers, callees, inheritance, composition)
3. `handleBlast()` - Impact analysis showing what would break if a symbol changed
4. `handleConventions()` - Detects project patterns and recurring styles
5. `handleStatus()` - Reports index health, coverage, and session metrics

## Supporting Functions
- Symbol resolution with disambiguation
- Dispatch caller inference for interface methods
- Response compaction for large result sets
- Freshness computation and stale file detection
- Language breakdown and structural analysis
- Framework and entry point detection

## Constants
Configuration thresholds including interface resolution (3 callers), dispatch confidence (0.8), edge compaction (10 results), and maximum query depths.

## State Management
The `handlers` struct maintains adapter connection, search engine, metrics tracker, and symbol cache with mutex-protected access.
