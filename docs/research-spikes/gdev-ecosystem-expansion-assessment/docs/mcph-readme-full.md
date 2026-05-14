# mcph README — Full Content
- **Source**: https://raw.githubusercontent.com/YawLabs/mcph/main/README.md
- **Retrieved**: 2026-05-14

## Overview

The mcph project is a centralized MCP server orchestrator. As stated in the documentation: "One install. All your MCP servers. Managed from the cloud."

## Core Purpose

mcph acts as a single point of contact between your AI client and multiple MCP servers. Rather than configuring each server individually per client and machine, you configure them once on mcp.hosting and every installation syncs automatically.

## Key Problems It Solves

The documentation identifies several pain points:

1. **Multi-device management** — "Add a server once on the dashboard; every client/device picks it up on the next poll"
2. **Context bloat** — A ranking system loads only relevant servers, preventing "hundreds" of tools from surfacing by default
3. **Credential management** — "Credentials live encrypted on mcp.hosting and inject at spawn time"
4. **Compliance visibility** — Servers display A-F compliance grades before activation

## Architecture

mcph sits between your AI client and all MCP servers, handling tool routing through meta-tools like `mcp_connect_dispatch`, `mcp_connect_discover`, and `mcp_connect_activate`.

## Installation

The quickest approach uses:
```bash
npx -y @yawlabs/mcph install <client-name> --token mcp_pat_your_token_here
```

This modifies the client config file and stores the token in `~/.mcph/config.json`.

Supported clients: Claude Code, Claude Desktop, Cursor, VS Code.

## Configuration Files

mcph reads configuration from three scopes (highest precedence first):
- `./<project>/.mcph/config.local.json` (machine-local, not committed)
- `./<project>/.mcph/config.json` (team-shared)
- `~/.mcph/config.json` (personal default)

## Meta-Tools Available

The system provides these control mechanisms:

- **dispatch** — describe a task; mcph ranks servers and loads the best match
- **discover** — list all servers with optional relevance ranking
- **activate/deactivate** — manually load or unload specific servers
- **install** — add new servers to your account
- **import** — bulk-load servers from existing client configs
- **health** — show call counts, errors, and latency
- **suggest** — surface learned multi-server workflows
- **read_tool** — inspect a tool schema without loading the full server
- **exec** — run declarative pipelines of tool calls
- **bundles** — activate curated multi-server presets

## Ranking Intelligence

When the backend has a Voyage embeddings key configured, mcph performs two-stage ranking: BM25 locally, then semantic reordering via the backend. Without that key, it degrades gracefully to BM25-only.

Three client-side signals adjust scores:
- **Health-aware** — failed or high-error servers get down-ranked
- **Learning** — previously successful servers receive a small boost (max +10%)
- **Sampling tiebreak** — when top candidates are within 10%, the model chooses

## Config Polling & Sync

"mcph polls [mcp.hosting] every 60 seconds for config changes." This enables multi-device synchronization without requiring manual file sync across machines.

## Environment Variables

Key controls include:
- `MCPH_TOKEN` — personal access token (overrides file-based token)
- `MCPH_URL` — API endpoint (defaults to https://mcp.hosting)
- `LOG_LEVEL` — logging verbosity
- `MCPH_AUTO_ACTIVATE` — auto-load winning server from discover
- `MCPH_MIN_COMPLIANCE` — minimum compliance grade filter (A-F)
- `MCPH_DISABLE_PERSISTENCE` — disable cross-session learning
- `MCPH_AUTO_LOAD` — pre-activate top pack on startup

## Diagnostic Commands

```bash
mcph doctor              # health check and config report
mcph servers [filter]    # list installed servers
mcph bundles [action]    # browse or match presets
mcph compliance <target> # audit an MCP server
mcph --version           # show version
```

## Security Model

mcph emphasizes transparency rather than sandboxing:

- **Compliance testing** — 88 behavioral tests rate servers A-F; minimum grades can be enforced
- **Source visibility** — exact command/args/URL shown before installation
- **Encrypted credentials** — secrets stored encrypted on the backend
- **Response pruning** — large payloads redacted before reaching the LLM (prevents prompt injection)
- **Namespace isolation** — tools prefixed to prevent impersonation

The documentation notes: "mcph does not prevent a server you deliberately installed from doing harmful things inside its own process."

## Project Guides

Drop `MCPH.md` inside `.mcph/` to provide project-specific routing conventions and credential guidance. When both user and project guides exist, they combine with project guidance appearing last.

## Runtime Detection

On startup, mcph probes for Node.js, Python, Docker, and other runtimes, reporting to the dashboard. It includes automatic `uv`/`uvx` bootstrapping for Python-based servers, downloading Astral's standalone binary if needed.

## Requirements

- Node.js 18+
- mcp.hosting account with API token

## Persistence & Learning

By default, mcph stores learning state in `~/.mcph/state.json` across sessions. Set `MCPH_DISABLE_PERSISTENCE=1` to disable this (useful in ephemeral environments like CI).

The `mcp_connect_suggest` tool surfaces "recurring multi-server workflows mcph has learned from persisted pack history," enabling one-click activation of common tool combinations.

## Additional Resources

- Dashboard: mcp.hosting
- Compliance suite: @yawlabs/mcp-compliance on npm
- Source: GitHub YawLabs/mcph
- Security contact: support@mcp.hosting
