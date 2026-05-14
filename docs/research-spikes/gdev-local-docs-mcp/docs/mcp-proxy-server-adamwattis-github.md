<!-- Source: https://github.com/adamwattis/mcp-proxy-server -->
<!-- Retrieved: 2026-05-14 -->

# MCP Proxy Server (adamwattis)

An MCP proxy server that aggregates and serves multiple MCP resource servers through a single interface.

## Core Functionality

Connects to and manages multiple MCP resource servers while exposing their combined capabilities through a unified interface.

## Key Features

**Resource Management:** Discovers and connects to multiple MCP servers, aggregates resources while maintaining consistent URI schemes across servers, handles resource routing and resolution.

**Tool Aggregation:** Tools from all connected servers are exposed through the proxy, routes tool calls to appropriate backend servers while maintaining state and managing responses.

**Prompt Handling:** Aggregates prompts from multiple backends and routes requests appropriately, supports multi-server prompt responses.

## Configuration

```json
{
  "servers": [
    {
      "name": "Server Name",
      "transport": {
        "command": "/path/to/server/build/index.js",
        "args": ["--option"],
        "env": ["API_KEY"]
      }
    }
  ]
}
```

## Transport Support

- **Stdio**: Direct command execution
- **SSE**: HTTP-based connections (e.g., "http://localhost:8080/sse")

## Notable

KEEP_SERVER_OPEN environment variable for maintaining SSE connections when multiple clients connect. No specific failover or priority routing mechanisms documented.

198 stars, 44 forks, TypeScript 94.2%.
