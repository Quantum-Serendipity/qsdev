# Claude Code: MCP Integration — Official Documentation
- **Source**: https://code.claude.com/docs/en/mcp
- **Retrieved**: 2026-03-15
- **Type**: Official documentation

## Overview
Claude Code connects to hundreds of external tools via Model Context Protocol (MCP), an open source standard for AI-tool integrations. MCP servers give access to tools, databases, and APIs.

## Transport Types
1. **HTTP** (recommended): Remote servers, cloud-based services
2. **SSE** (deprecated): Server-Sent Events, being replaced by HTTP
3. **stdio**: Local processes, direct system access
4. **WebSocket**: Also supported

## Installation
```bash
claude mcp add --transport http <name> <url>
claude mcp add --transport stdio --env KEY=value <name> -- <command> [args...]
```

## Scopes
| Scope | Location | Sharing |
|---|---|---|
| Local (default) | ~/.claude.json under project path | Private, current project |
| Project | .mcp.json at project root | Team via version control |
| User | ~/.claude.json | Private, all projects |

## Authentication
- OAuth 2.0 support for remote servers
- /mcp command for browser-based auth flow
- Pre-configured credentials with --client-id and --client-secret
- Token stored securely in system keychain

## MCP Tool Search
Automatically enabled when MCP tool descriptions exceed 10% of context window. Tools loaded on-demand instead of all upfront. Configurable via ENABLE_TOOL_SEARCH env var.

## Key Features
- Dynamic tool updates via list_changed notifications
- MCP resources via @ mentions
- MCP prompts available as /mcp__servername__promptname commands
- Plugin-provided MCP servers
- Managed MCP configuration for organizations (allowlists/denylists)
- Environment variable expansion in .mcp.json
- Elicitation support (MCP servers can request user input)
- Claude Code itself can serve as an MCP server (claude mcp serve)

## Context Considerations
- MCP tools add definitions to every request
- Output warning at 10,000 tokens, max 25,000 by default
- MAX_MCP_OUTPUT_TOKENS to increase limit
- /mcp to check per-server context costs
