# mcph Roadmap
- **Source**: https://raw.githubusercontent.com/YawLabs/mcph/main/ROADMAP.md
- **Retrieved**: 2026-05-14

## Phase 1 — v0.1 (Complete)

All items marked complete:
- Cloud-configured, locally-executed MCP orchestrator
- Discover / load / unload meta-tools
- Local server spawning (stdio) + remote server connections (HTTP)
- Namespace-based tool routing
- Tools/list_changed notifications on load/unload
- 60-second config polling with version hash comparison
- Auto-unload servers idle for 10+ tool calls
- Directive tool descriptions for context-aware LLM behavior
- Graceful shutdown (SIGTERM/SIGINT)
- Plan-based server limits (free: 3, paid: unlimited)

## Phase 2 — Smart Routing & Observability (Complete)

All items marked complete, including:
- Context cost estimates in discover() showing token cost per server
- Usage pattern hints tracking frequently-loaded server combinations
- Suggested load with mcp_connect_suggest and mcp_connect_discover
- Automatic load capability via MCPH_AUTO_LOAD=1 environment variable
- Routing analytics upload to mcp.hosting dashboard
- Error tracking displaying server health in discover()
- Concurrent server cap (default 6, configurable via MCPH_SERVER_CAP)
- Resource and prompt proxying from upstream servers
- Cross-session persistence via ~/.mcph/state.json
- Per-tool load filtering with mcp_connect_activate
- Signature-on-demand meta-tool (mcp_connect_read_tool)
- Orchestration sandbox (mcp_connect_exec) with 16-step limit
- Marketplace integration linking to mcp.hosting/explore
- Multi-device config sync across machines

## Phase 3 — Platform Intelligence (Partially Complete)

- Server recommendation engine (not started)
- Pre-built orchestrator configs via mcp_connect_bundles (complete)
- Compliance-aware routing with MCPH_MIN_COMPLIANCE setting (complete)
- Tool deduplication surfacing overlapping tools (complete)
- Conversation-aware routing (not started; awaits future MCP spec)
