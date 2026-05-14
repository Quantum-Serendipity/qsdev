# @yawlabs/tailscale-mcp README
- **Source**: https://raw.githubusercontent.com/YawLabs/tailscale-mcp/main/README.md
- **Retrieved**: 2026-05-14

## Core Purpose

MCP server enabling AI agents to manage Tailscale networks through conversational queries. Lets agents "compose multi-endpoint workflows in one turn without writing a script."

## Supported Operations (89 core tools + 4 optional CLI tools)

- Status monitoring
- Device management (17 tools)
- ACL/policy handling with HuJSON preservation
- DNS configuration (11 tools)
- Authentication keys and OAuth clients
- User administration (7 tools)
- Webhooks, posture integrations, and log streaming
- Audit and network flow logging

## Authentication Methods

1. API Key via `TAILSCALE_API_KEY` environment variable
2. OAuth credentials: `TAILSCALE_OAUTH_CLIENT_ID` and `TAILSCALE_OAUTH_CLIENT_SECRET`

## Key Features

- Built-in 429 rate-limit retry logic with exponential backoff
- Typed, Zod-validated tool inputs and responses
- Safety hints (readOnlyHint, destructiveHint) for client confirmation gating
- 700+ unit tests plus optional live-tailnet integration testing
- Configurable tool filtering via profiles (minimal, core, full)
- Read-only mode and concurrent request limiting options
