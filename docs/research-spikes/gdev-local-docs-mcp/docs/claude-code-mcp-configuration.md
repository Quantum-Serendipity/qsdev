<!-- Source: https://code.claude.com/docs/en/mcp -->
<!-- Retrieved: 2026-05-14 -->

# Connect Claude Code to Tools via MCP - Configuration Reference

## MCP Server Scopes

| Scope   | Loads in             | Shared with team | Stored in                 |
|---------|----------------------|------------------|---------------------------|
| Local   | Current project only | No               | ~/.claude.json            |
| Project | Current project only | Yes              | .mcp.json in project root |
| User    | All your projects    | No               | ~/.claude.json            |

## Scope Hierarchy and Precedence

When the same server is defined in more than one place, Claude Code connects to it once, using the definition from the highest-precedence source:

1. Local scope
2. Project scope
3. User scope
4. Plugin-provided servers
5. claude.ai connectors

The three scopes match duplicates by name. Plugins and connectors match by endpoint.

## .mcp.json Format

```json
{
  "mcpServers": {
    "server-name": {
      "command": "/path/to/server",
      "args": [],
      "env": {}
    }
  }
}
```

HTTP server:
```json
{
  "mcpServers": {
    "api-server": {
      "type": "http",
      "url": "https://api.example.com/mcp",
      "headers": {
        "Authorization": "Bearer ${API_KEY}"
      }
    }
  }
}
```

## Environment Variable Expansion

- ${VAR} - Expands to value of VAR
- ${VAR:-default} - Expands to VAR if set, otherwise uses default

Supported in: command, args, env, url, headers

## Tool Search (Default Enabled)

MCP tools are deferred rather than loaded into context upfront. Claude uses a search tool to discover relevant ones when a task needs them. Only the tools Claude actually uses enter context.

| ENABLE_TOOL_SEARCH | Behavior |
|--------------------|----------|
| (unset)            | All MCP tools deferred and loaded on demand |
| true               | All deferred, forces beta header |
| auto               | Tools load upfront if within 10% of context window |
| auto:N             | Custom threshold percentage |
| false              | All tools loaded upfront |

## alwaysLoad

Set alwaysLoad: true in server config to exempt from tool search deferral.

```json
{
  "mcpServers": {
    "core-tools": {
      "type": "http",
      "url": "https://mcp.example.com/mcp",
      "alwaysLoad": true
    }
  }
}
```

Individual tools can be marked with "anthropic/alwaysLoad": true in _meta.

## Dynamic Tool Updates

Claude Code supports MCP list_changed notifications, allowing servers to dynamically update available tools without reconnection.

## Automatic Reconnection

HTTP/SSE servers: exponential backoff, up to 5 attempts, starting at 1s doubling each time.
Stdio servers: local processes, not reconnected automatically.

## Output Limits

- Warning threshold: 10,000 tokens
- Default maximum: 25,000 tokens (configurable via MAX_MCP_OUTPUT_TOKENS)
- Per-tool override: anthropic/maxResultSizeChars in _meta (up to 500,000 chars)

## Managed MCP Configuration

managed-mcp.json provides exclusive control over MCP servers.
- Linux: /etc/claude-code/managed-mcp.json
- macOS: /Library/Application Support/ClaudeCode/managed-mcp.json

Policy-based control available via mcpPolicy with allowlist/denylist patterns.

## Key Behavioral Notes

- MCP servers are per-session; each pane spins up its own instances
- MCP connections can fail silently mid-session
- Server name "workspace" is reserved
- CLAUDE_PROJECT_DIR is set in spawned server environment
