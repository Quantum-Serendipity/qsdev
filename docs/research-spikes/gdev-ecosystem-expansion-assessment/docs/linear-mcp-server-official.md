<!-- Source: https://linear.app/docs/mcp -->
<!-- Retrieved: 2026-05-14 -->

# Linear MCP Server (Official)

## Official Status
Official from Linear, centrally hosted managed service.

## Core Capabilities
- Finding objects in Linear (issues, projects, etc.)
- Creating objects (issues, projects, comments)
- Updating objects with more functionality planned

## Authentication & Security Model
- Primary: OAuth 2.1 with dynamic client registration at https://mcp.linear.app/mcp
- Alternative: Direct authentication via Bearer tokens (OAuth access tokens, API keys including restricted read-only variants, app user authentication)

## Supported Clients
Native integration with Claude (desktop and claude.ai), Cursor, Codex, Jules, v0, Windsurf, Zed, VS Code (via mcp-remote module), plus hundreds of others.

## Transport Protocol
Streamable HTTP transports (standard MCP communication method).

## Limitations
Current tool set is incomplete relative to Linear's full API surface -- more functionality on the way.
