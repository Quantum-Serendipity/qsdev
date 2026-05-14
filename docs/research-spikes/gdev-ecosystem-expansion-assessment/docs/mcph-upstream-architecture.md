# mcph Upstream Connection Architecture
- **Source**: https://raw.githubusercontent.com/YawLabs/mcph/main/src/upstream.ts
- **Retrieved**: 2026-05-14

## Transport Types

**Local (Stdio):** Spawns child processes with configurable environment variables. Rewrites `uv`/`uvx` to managed binary when user doesn't have one on PATH.

**Remote (HTTP/SSE):** Connects to remote endpoints via StreamableHTTPClientTransport or SSEClientTransport based on config.

## Timeout Strategy

- CONNECT_TIMEOUT (default 15s): Initial handshake window
- LIST_TIMEOUT (default 15s): Per-request inventory calls after connection
- Uses Promise.race() to enforce boundaries

## Error Categorization

Failures classified into 5 categories:
- spawn_failure (command not found)
- install_failure (process exits pre-handshake)
- init_timeout (handshake incomplete)
- protocol_error (post-handshake failures)
- unknown

Stderr capture (8KB rolling buffer) provides diagnostic context for local transports.

## Capability Discovery

Fetches tools, resources, and prompts with per-category limits (1000 each) to prevent memory exhaustion from buggy servers. Per-category chain prevents race conditions from rapid notifications.

## Dynamic Updates

Subscribes to ToolListChanged, ResourceListChanged, and PromptListChanged events. Serializes updates per category to maintain consistency.
