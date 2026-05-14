# Claude Code Session Data Format

## Overview

Claude Code stores all local state in `~/.claude/`. Session conversations are stored as JSONL (JSON Lines) files — one JSON object per line — under a project-keyed directory hierarchy. This format is append-only, making it crash-resilient and easy to parse incrementally. There is no official schema documentation from Anthropic; this report is based on direct inspection of live session data and community documentation.

## Directory Structure

```
~/.claude/
├── .credentials.json          # Authentication credentials
├── settings.json              # Global user settings (env vars, effort level)
├── history.jsonl              # Global command history across all projects
├── stats-cache.json           # Aggregated usage statistics
├── backups/                   # Timestamped backups of .claude.json
├── cache/                     # Cached data (e.g., changelog)
├── commands/                  # Custom slash commands
├── debug/                     # Debug logs per session (plain text)
├── file-history/              # File version snapshots per session
│   └── <session-uuid>/
│       └── <file-hash>@v<N>  # Versioned file content snapshots
├── ide/                       # IDE integration data
├── paste-cache/               # Clipboard/paste content cache
├── plans/                     # Execution plan markdown files (named by session slug)
├── plugins/                   # Plugin data
├── projects/                  # Per-project session data (the main data store)
│   └── <encoded-cwd>/        # Project path with / → - encoding
│       ├── <session-uuid>.jsonl        # Full conversation transcript
│       ├── <session-uuid>/             # Session artifacts directory
│       │   └── subagents/
│       │       ├── agent-<id>.jsonl       # Subagent conversation transcript
│       │       └── agent-<id>.meta.json   # Subagent metadata
│       └── memory/
│           └── MEMORY.md      # Auto-memory persisted across conversations
├── session-env/               # Session environment snapshots
│   └── <session-uuid>/       # (contents may be empty or env data)
├── shell-snapshots/           # Shell state snapshots (zsh/bash)
├── statsig/                   # Feature flag / analytics
├── tasks/                     # Internal task tracking (per session)
├── telemetry/                 # Usage telemetry
└── todos/                     # Internal todo state (per session/agent)
```

### Project Path Encoding

Project directories use the absolute working directory path with every `/` replaced by `-`. For example:
- `/home/colin/Repos/research` becomes `-home-colin-Repos-research`

### Session Files

Each session produces:
- **`<uuid>.jsonl`** — The full conversation transcript (the primary data artifact)
- **`<uuid>/`** — An optional directory containing subagent transcripts and other session artifacts

Session files range from ~2KB (minimal interactions) to 28MB+ (long working sessions). A typical substantive session is 1-3MB.

## JSONL Session Format

Each line in a session `.jsonl` file is a self-contained JSON object. Every object has a `type` field that determines its schema.

### Common Fields (Present on Most Message Types)

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Message type: `user`, `assistant`, `system`, `progress`, `file-history-snapshot` |
| `uuid` | string | Unique identifier for this message |
| `parentUuid` | string/null | UUID of the parent message (forms conversation tree) |
| `timestamp` | string | ISO 8601 timestamp |
| `sessionId` | string | UUID of the containing session |
| `cwd` | string | Working directory at time of message |
| `version` | string | Claude Code CLI version (e.g., `"2.1.74"`) |
| `gitBranch` | string | Current git branch |
| `isSidechain` | boolean | Whether this is a sidechain (subagent) message |
| `userType` | string | User classification (e.g., `"external"`) |
| `slug` | string | Human-readable session name (e.g., `"wise-tumbling-dahl"`) — appears after first assistant response |

### Message Type: `user`

User messages come in three variants:

**1. Direct user input** (typed by human):
```json
{
  "type": "user",
  "message": {
    "role": "user",
    "content": "the user's message text"
  },
  "uuid": "...",
  "parentUuid": null,
  "timestamp": "2026-03-26T19:13:11.236Z",
  "userType": "external",
  "cwd": "/home/colin/Repos/research",
  "sessionId": "...",
  "version": "2.1.74",
  "gitBranch": "main"
}
```

**2. Injected system/skill context** (`isMeta: true`):
```json
{
  "type": "user",
  "isMeta": true,
  "message": {
    "role": "user",
    "content": [{"type": "text", "text": "skill instructions..."}]
  }
}
```
These are skill/command instructions injected by the system, not typed by the user.

**3. Tool results** (tool execution output returned to the model):
```json
{
  "type": "user",
  "message": {
    "role": "user",
    "content": [{
      "tool_use_id": "toolu_01NVPM6JtP4tk3Gn9a87zHKm",
      "type": "tool_result",
      "content": "...tool output text..."
    }]
  },
  "toolUseResult": {
    "type": "text",
    "file": {
      "filePath": "/path/to/file",
      "content": "...full file content...",
      "numLines": 13,
      "startLine": 1,
      "totalLines": 13
    }
  },
  "sourceToolAssistantUUID": "..."
}
```
The `toolUseResult` field contains the *structured* tool output, while `message.content` contains the serialized version sent to the API. For file reads, `toolUseResult.file` includes the full file content, path, and line metadata.

### Message Type: `assistant`

```json
{
  "type": "assistant",
  "message": {
    "model": "claude-opus-4-6",
    "id": "msg_...",
    "type": "message",
    "role": "assistant",
    "content": [
      {"type": "thinking", "thinking": "...", "signature": "..."},
      {"type": "text", "text": "...response text..."},
      {"type": "tool_use", "id": "toolu_...", "name": "Read", "input": {...}, "caller": {"type": "direct"}}
    ],
    "stop_reason": "end_turn" | "tool_use" | null,
    "stop_sequence": null,
    "usage": {
      "input_tokens": 2,
      "cache_creation_input_tokens": 12355,
      "cache_read_input_tokens": 8806,
      "cache_creation": {
        "ephemeral_5m_input_tokens": 0,
        "ephemeral_1h_input_tokens": 12355
      },
      "output_tokens": 30,
      "service_tier": "standard",
      "inference_geo": "not_available"
    }
  },
  "requestId": "req_..."
}
```

Key details:
- **`message.content`** is an array of content blocks, which can be `thinking`, `text`, or `tool_use`
- **`message.usage`** contains detailed token accounting including cache breakdown
- **`stop_reason`** is `null` for streaming intermediate chunks, `"end_turn"` for final response, `"tool_use"` when the model wants to call a tool
- **`thinking`** blocks contain extended thinking content with cryptographic `signature` for verification
- **`tool_use`** blocks include the tool `name`, `input` parameters, and a `caller` field

### Message Type: `system`

```json
{
  "type": "system",
  "subtype": "turn_duration",
  "durationMs": 38249,
  "isMeta": false
}
```

System messages record metadata events. Known subtypes:
- `turn_duration` — Records how long a complete turn (user → assistant response cycle) took

### Message Type: `progress`

```json
{
  "type": "progress",
  "data": {
    "type": "hook_progress",
    "hookEvent": "PostToolUse",
    "hookName": "PostToolUse:Read",
    "command": "callback"
  },
  "parentToolUseID": "toolu_...",
  "toolUseID": "toolu_..."
}
```

Progress messages track tool execution lifecycle events, including hook executions. The `data` field varies by progress type.

### Message Type: `file-history-snapshot`

```json
{
  "type": "file-history-snapshot",
  "messageId": "...",
  "snapshot": {
    "messageId": "...",
    "trackedFileBackups": {
      "relative/path/to/file.md": {
        "backupFileName": null,
        "version": 1,
        "backupTime": "2026-03-26T19:13:29.201Z"
      }
    },
    "timestamp": "2026-03-26T19:13:11.237Z"
  },
  "isSnapshotUpdate": false | true
}
```

These track file modifications for undo/redo capability. `isSnapshotUpdate: false` appears at session start; `true` appears as files are modified. Actual file content is stored in `~/.claude/file-history/<session-uuid>/<hash>@v<N>`.

## Conversation Tree Structure

Messages form a tree via `parentUuid` → `uuid` linkage. This supports:
- **Linear conversations** — simple parent-child chains
- **Branching** — when the user rewinds and takes a different path (via Escape double-tap)
- **Subagent spawns** — subagent messages have `isSidechain: true` and their own `agentId`

## Global History File

`~/.claude/history.jsonl` is a separate file that logs every user input across all projects:

```json
{
  "display": "the user's input text",
  "pastedContents": {},
  "timestamp": 1771508921910,
  "project": "/home/colin/Repos/research",
  "sessionId": "92ee189f-59ca-4701-a9a4-660c13ffa892"
}
```

Note: timestamps here are Unix epoch milliseconds, not ISO 8601.

## Stats Cache

`~/.claude/stats-cache.json` contains aggregated daily usage metrics:

```json
{
  "version": 2,
  "lastComputedDate": "2026-03-01",
  "dailyActivity": [
    {
      "date": "2026-02-19",
      "messageCount": 884,
      "sessionCount": 4,
      "toolCallCount": 372
    }
  ],
  "dailyModelTokens": [
    {
      "date": "2026-02-19",
      "tokensByModel": {
        "claude-opus-4-5-20251101": 219652
      }
    }
  ]
}
```

## Subagent Data

When Claude Code spawns subagents (for delegated tasks), their transcripts are stored in:
```
<session-uuid>/subagents/agent-<hex-id>.jsonl
<session-uuid>/subagents/agent-<hex-id>.meta.json
```

The meta.json contains: `{"agentType":"general-purpose"}`

Subagent JSONL files follow the same format as main session files, with these additions:
- `isSidechain: true` on all messages
- `agentId` field identifying the subagent
- `promptId` on the initial user message (the task prompt)

## Other Supporting Files

| File/Directory | Format | Content |
|---------------|--------|---------|
| `settings.json` | JSON | Global config: env vars, effort level |
| `debug/<session-uuid>.txt` | Plain text | Debug/trace logs |
| `plans/<slug>.md` | Markdown | Execution plans generated during sessions |
| `shell-snapshots/snapshot-*.sh` | Shell script | Shell environment state captures |
| `file-history/<session-uuid>/<hash>@v<N>` | Raw file content | Versioned file backups for undo |
| `todos/<session-uuid>-agent-<id>.json` | JSON | Internal todo/task state (often empty `[]`) |

## What's Available for Analysis Tools

### High-Value Data
1. **Full conversation transcripts** — Every user input, assistant response, tool call, and tool result
2. **Token usage per API call** — Input, output, cache creation, cache read breakdowns
3. **Tool call details** — Tool name, parameters, results, duration (via system turn_duration)
4. **File modification history** — What files were read/written, with versioned snapshots
5. **Conversation tree structure** — Parent-child UUID chains enable branching/rewind reconstruction
6. **Timing data** — Timestamps on every message, turn duration on system messages
7. **Model identification** — Which model was used for each response
8. **Subagent traces** — Full transcripts of delegated work with linkage to parent session

### Aggregate Data
1. **Daily activity metrics** — Message counts, session counts, tool call counts (stats-cache.json)
2. **Token usage by model by day** — Model-level token accounting
3. **Global command history** — Every input across all projects with timestamps

### Limitations
- No official schema documentation from Anthropic — format is empirically observed and may change
- `stop_reason` is `null` on streaming intermediate chunks (only meaningful on final message of a turn)
- Thinking block content is present but cryptographically signed (signature verification unclear)
- File content in tool results can be very large, making session files range from KB to tens of MB
- The `slug` field (human-readable session name) only appears after the first assistant response, not on the initial user message
