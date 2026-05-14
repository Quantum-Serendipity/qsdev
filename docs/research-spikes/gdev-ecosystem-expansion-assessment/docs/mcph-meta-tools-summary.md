# mcph Meta-Tools — Complete Reference
- **Source**: https://raw.githubusercontent.com/YawLabs/mcph/main/src/meta-tools.ts
- **Retrieved**: 2026-05-14

## 11 Meta-Tools

### Core Tools

**mcp_connect_discover** — List MCP servers installed on the user's mcp.hosting account. Accepts optional context to rank results by relevance. Shows tool counts, token estimates, compliance grades [A]-[F], and usage hints.

**mcp_connect_activate** — Loads one or more server namespaces into the session. Supports filtering via optional `tools` parameter to expose only specific tool names. Respects `MCPH_MIN_COMPLIANCE` floor.

**mcp_connect_deactivate** — Unloads servers from the session to free context. Servers remain installed and can be reloaded. Auto-unloads idle servers after 10+ tool calls to others.

**mcp_connect_dispatch** — "PREFERRED entry point when the task is already concrete." Ranks installed servers via BM25, loads the top match(es), and exposes tools in one call. Default budget is 1 server; cap is 10.

### Installation & Management

**mcp_connect_install** — Installs new servers by specifying namespace, type ("local" or "remote"), command/args, and optional env vars. Namespace must match `/^[a-z][a-z0-9_]{0,29}$/`.

**mcp_connect_import_config** — Imports MCP servers from existing client configs (Claude Desktop, Cursor, VS Code). Reads `mcpServers` section and creates matching mcp.hosting entries.

**mcp_connect_read_tool** — Returns a single tool's full input schema without loading its server.

### Discovery & Workflow

**mcp_connect_suggest** — Surfaces recurring multi-server patterns as "packs" from prior usage. Persists patterns across restarts via `~/.mcph/state.json`.

**mcp_connect_bundles** — Lists curated multi-server presets (e.g., pr-review, devops-incident). Supports "list" (full catalog) or "match" (partition against installed servers).

### Execution & Monitoring

**mcp_connect_exec** — Runs declarative pipelines of up to 16 sequential tool calls with data flow via `{"$ref": "<stepId>.path"}` substitution. Not a code sandbox: no expression language, no loops, no branching.

**mcp_connect_health** — Shows health stats for loaded servers: total calls, error count, latency, and last error.

## Key Constraints

- Namespaces: 30 chars max, lowercase, start with letter
- Remote URLs must use HTTPS (HTTP only for localhost loopback)
- Max 50 args, 50 env vars, 500-char descriptions
- Free tier: 3-server cap (shows upgrade URL on 403)
