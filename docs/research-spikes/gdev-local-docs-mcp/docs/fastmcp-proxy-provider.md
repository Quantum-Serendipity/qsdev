<!-- Source: https://gofastmcp.com/servers/providers/proxy -->
<!-- Retrieved: 2026-05-14 -->

# FastMCP Proxy Provider

## Core Functionality

The Proxy Provider (v2.0.0+) sources components from other MCP servers through client connections. It lets you expose any MCP server's tools, resources, and prompts through your own server.

## Primary Use Cases

1. Transport Bridging - HTTP to stdio or vice versa
2. Server Aggregation - Multiple servers into single endpoint
3. Security Gateway - Controlled access with auth
4. Endpoint Stability - Consistent access when backends change

## Simple Proxy

```python
from fastmcp.server import create_proxy
proxy = create_proxy("http://example.com/mcp", name="MyProxy")
proxy.run()
```

## Multi-Server Aggregation (v2.4.0+)

```python
config = {
    "mcpServers": {
        "weather": {"url": "https://weather-api.example.com/mcp", "transport": "http"},
        "calendar": {"url": "https://calendar-api.example.com/mcp", "transport": "http"}
    }
}
composite = create_proxy(config, name="Composite")
```

Generates prefixed components: weather_get_forecast, calendar_add_event.

## Namespace Prefixing

| Component | Pattern |
|-----------|---------|
| Tools | {prefix}_{tool_name} |
| Prompts | {prefix}_{prompt_name} |
| Resources | protocol://{prefix}/path |

## Session Management

Default (v2.10.3+): Each request gets isolated backend session.
Shared: Pass pre-connected client to reuse session (risks context mixing in concurrent scenarios).

## Performance

Proxying adds 300-400ms latency for list_tools() vs 1-2ms local.
v3.2.0 adds caching with configurable TTL.

## Mounting in Existing Servers

```python
server = FastMCP("My Server")

@server.tool
def local_tool() -> str:
    return "Local result"

external = create_proxy("http://external-server/mcp")
server.mount(external)
```

## Limitations

No explicit failover or priority mechanisms documented. Components are read-only mirrors. Modification requires creating local copies.
