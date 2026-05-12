<!-- Source: https://code.claude.com/docs/en/mcp -->
<!-- Retrieved: 2026-05-12 -->

# Connect Claude Code to tools via MCP

> Learn how to connect Claude Code to your tools with the Model Context Protocol.

Claude Code can connect to hundreds of external tools and data sources through the Model Context Protocol (MCP), an open source standard for AI-tool integrations. MCP servers give Claude Code access to your tools, databases, and APIs.

Connect a server when you find yourself copying data into chat from another tool, like an issue tracker or a monitoring dashboard. Once connected, Claude can read and act on that system directly instead of working from what you paste.

## What you can do with MCP

With MCP servers connected, you can ask Claude Code to:

* **Implement features from issue trackers**: "Add the feature described in JIRA issue ENG-4521 and create a PR on GitHub."
* **Analyze monitoring data**: "Check Sentry and Statsig to check the usage of the feature described in ENG-4521."
* **Query databases**: "Find emails of 10 random users who used feature ENG-4521, based on our PostgreSQL database."
* **Integrate designs**: "Update our standard email template based on the new Figma designs that were posted in Slack"
* **Automate workflows**: "Create Gmail drafts inviting these 10 users to a feedback session about the new feature."
* **React to external events**: An MCP server can also act as a channel that pushes messages into your session, so Claude reacts to Telegram messages, Discord chats, or webhook events while you're away.

## Installing MCP servers

MCP servers can be configured in three different ways depending on your needs:

### Option 1: Add a remote HTTP server

HTTP servers are the recommended option for connecting to remote MCP servers. This is the most widely supported transport for cloud-based services.

```bash
# Basic syntax
claude mcp add --transport http <name> <url>

# Real example: Connect to Notion
claude mcp add --transport http notion https://mcp.notion.com/mcp

# Example with Bearer token
claude mcp add --transport http secure-api https://api.example.com/mcp \
  --header "Authorization: Bearer your-token"
```

When configuring MCP servers via JSON in `.mcp.json`, `~/.claude.json`, or `claude mcp add-json`, the `type` field accepts `streamable-http` as an alias for `http`. The MCP specification uses the name `streamable-http` for this transport, so configurations copied from server documentation work without modification.

### Option 2: Add a remote SSE server

> Warning: The SSE (Server-Sent Events) transport is deprecated. Use HTTP servers instead, where available.

```bash
# Basic syntax
claude mcp add --transport sse <name> <url>

# Real example: Connect to Asana
claude mcp add --transport sse asana https://mcp.asana.com/sse
```

### Option 3: Add a local stdio server

Stdio servers run as local processes on your machine. They're ideal for tools that need direct system access or custom scripts.

Claude Code sets `CLAUDE_PROJECT_DIR` in the spawned server's environment to the project root, so your server can resolve project-relative paths without depending on the working directory.

```bash
# Basic syntax
claude mcp add [options] <name> -- <command> [args...]

# Real example: Add Airtable server
claude mcp add --transport stdio --env AIRTABLE_API_KEY=YOUR_KEY airtable \
  -- npx -y airtable-mcp-server
```

**Important: Option ordering** — All options (`--transport`, `--env`, `--scope`, `--header`) must come before the server name. The `--` (double dash) then separates the server name from the command and arguments that get passed to the MCP server.

### Managing your servers

```bash
# List all configured servers
claude mcp list

# Get details for a specific server
claude mcp get github

# Remove a server
claude mcp remove github

# (within Claude Code) Check server status
/mcp
```

The `/mcp` panel shows the tool count next to each connected server and flags servers that advertise the tools capability but expose no tools.

### Dynamic tool updates

Claude Code supports MCP `list_changed` notifications, allowing MCP servers to dynamically update their available tools, prompts, and resources without requiring you to disconnect and reconnect.

### Automatic reconnection

If an HTTP or SSE server disconnects mid-session, Claude Code automatically reconnects with exponential backoff: up to five attempts, starting at a one-second delay and doubling each time. The server appears as pending in `/mcp` while reconnection is in progress. After five failed attempts the server is marked as failed and you can retry manually from `/mcp`. Stdio servers are local processes and are not reconnected automatically.

### Push messages with channels

An MCP server can also push messages directly into your session so Claude can react to external events like CI results, monitoring alerts, or chat messages.

## MCP installation scopes

MCP servers can be configured at three scopes:

| Scope   | Loads in             | Shared with team         | Stored in                   |
|---------|----------------------|--------------------------|-----------------------------|
| Local   | Current project only | No                       | `~/.claude.json`            |
| Project | Current project only | Yes, via version control | `.mcp.json` in project root |
| User    | All your projects    | No                       | `~/.claude.json`            |

### Local scope (default)

A local-scoped server loads only in the project where you added it and stays private to you.

```bash
claude mcp add --transport http stripe https://mcp.stripe.com
```

Configuration in `~/.claude.json`:
```json
{
  "projects": {
    "/path/to/your/project": {
      "mcpServers": {
        "stripe": {
          "type": "http",
          "url": "https://mcp.stripe.com"
        }
      }
    }
  }
}
```

### Project scope

Project-scoped servers enable team collaboration by storing configurations in a `.mcp.json` file at your project's root directory. This file is designed to be checked into version control.

```bash
claude mcp add --transport http paypal --scope project https://mcp.paypal.com/mcp
```

Resulting `.mcp.json`:
```json
{
  "mcpServers": {
    "shared-server": {
      "command": "/path/to/server",
      "args": [],
      "env": {}
    }
  }
}
```

For security, Claude Code prompts for approval before using project-scoped servers from `.mcp.json` files.

### User scope

User-scoped servers are stored in `~/.claude.json` and provide cross-project accessibility.

```bash
claude mcp add --transport http hubspot --scope user https://mcp.hubspot.com/anthropic
```

### Scope hierarchy and precedence

1. Local scope
2. Project scope
3. User scope
4. Plugin-provided servers
5. claude.ai connectors

### Environment variable expansion in `.mcp.json`

Supported syntax:
* `${VAR}` - Expands to the value of environment variable `VAR`
* `${VAR:-default}` - Expands to `VAR` if set, otherwise uses `default`

Expansion locations: `command`, `args`, `env`, `url`, `headers`

```json
{
  "mcpServers": {
    "api-server": {
      "type": "http",
      "url": "${API_BASE_URL:-https://api.example.com}/mcp",
      "headers": {
        "Authorization": "Bearer ${API_KEY}"
      }
    }
  }
}
```

## Plugin-provided MCP servers

Plugins can bundle MCP servers, automatically providing tools and integrations when the plugin is enabled.

Example `.mcp.json` at plugin root:
```json
{
  "mcpServers": {
    "database-tools": {
      "command": "${CLAUDE_PLUGIN_ROOT}/servers/db-server",
      "args": ["--config", "${CLAUDE_PLUGIN_ROOT}/config.json"],
      "env": {
        "DB_URL": "${DB_URL}"
      }
    }
  }
}
```

Plugin features:
* **Automatic lifecycle**: Servers for enabled plugins connect automatically at session startup
* **Environment variables**: use `${CLAUDE_PLUGIN_ROOT}`, `${CLAUDE_PLUGIN_DATA}`, and `${CLAUDE_PROJECT_DIR}`
* **Multiple transport types**: Support stdio, SSE, and HTTP transports

## Managed MCP configuration

Administrators can deploy servers at the enterprise level via managed configuration.

## Tool Discovery

If a server's tools should always be visible to Claude without a search step, set `alwaysLoad` to true in that server's configuration. Every tool from that server then loads into context at session start regardless of the ENABLE_TOOL_SEARCH setting.

## Configuration file locations

MCP configuration can be stored in multiple locations:
- Project-scoped MCP: `.mcp.json`
- Project-specific: `.claude/settings.local.json`
- User-specific local: `~/.claude/settings.local.json`
- User-specific global: `~/.claude/settings.json`
- Main Claude.json: `~/.claude.json`
- Dedicated MCP file: `~/.claude/mcp_servers.json`

## Tips

* Use the `--scope` flag to specify where the configuration is stored
* Set environment variables with `--env` flags
* Configure MCP server startup timeout using the MCP_TIMEOUT environment variable
* Claude Code will display a warning when MCP tool output exceeds 10,000 tokens (configurable via MAX_MCP_OUTPUT_TOKENS)
* Use `/mcp` to authenticate with remote servers that require OAuth 2.0 authentication
