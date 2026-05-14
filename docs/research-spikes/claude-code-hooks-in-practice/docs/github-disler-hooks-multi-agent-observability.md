# Claude Code Hooks Multi-Agent Observability - disler/claude-code-hooks-multi-agent-observability
- **Source**: https://github.com/disler/claude-code-hooks-multi-agent-observability
- **Retrieved**: 2026-03-27

## Overview
Real-time monitoring for Claude Code agents through comprehensive hook event tracking. Captures lifecycle events, stores them in SQLite, and visualizes them through a Vue 3 dashboard with WebSocket streaming.

## Hook Events Monitored (12 Total)

1. **PreToolUse** - Before tool execution
2. **PostToolUse** - After tool completion
3. **PostToolUseFailure** - Tool execution failures
4. **PermissionRequest** - Permission requests
5. **Notification** - User interactions
6. **UserPromptSubmit** - User prompt submissions
7. **Stop** - Response completion
8. **SubagentStart** - Subagent lifecycle start
9. **SubagentStop** - Subagent task completion
10. **PreCompact** - Context compaction
11. **SessionStart** - Session initialization
12. **SessionEnd** - Session termination

## Architecture

### Real-Time Monitoring Flow
```
Claude Agents → Hook Scripts → HTTP POST → Bun Server → SQLite
→ WebSocket → Vue Client
```

### Process
1. Claude Code triggers an action (tool execution, notification, etc.)
2. Corresponding hook script executes based on `settings.json` configuration
3. Hook gathers context (tool name, inputs, outputs, session ID)
4. `send_event.py` sends JSON payload to server via HTTP POST
5. Server validates, stores in SQLite with timestamp
6. Server broadcasts via WebSocket to all connected clients
7. Vue app updates timeline in real-time with filtered events

## Hook Scripts Architecture
Located in `.claude/hooks/`, uses Python scripts executed via `uv run`.

**Core Sender:** `send_event.py` — Universal event dispatcher supporting all 12 event types with server connectivity validation.

**Event-Specific Handlers:**
- Tool hooks validate inputs/outputs and detect MCP server usage
- Notification hooks include notification-type-aware text-to-speech
- User prompt hooks support JSON validation with blocking capability
- Session hooks track agent type, model, and end reasons
- Subagent hooks monitor agent lifecycle with transcript path tracking

## Data Collected

**Event Payload Fields:**
- `source_app` — Project identifier
- `session_id` — Unique session tracking
- `hook_event_type` — Event classification
- `timestamp` — Event occurrence time
- Event-specific fields (tool_name, tool_use_id, agent_id, notification_type, etc.)
- Chat transcript storage capability

**Tool-Specific Data:**
- Tool name and type (Bash, Read, Write, Edit, Task, etc.)
- Tool inputs/outputs
- MCP server detection with server and tool names
- Execution results or failure details

## Configuration
Each hook event configured with command pattern:
```json
"uv run .claude/hooks/send_event.py --source-app PROJECT_NAME --event-type EVENT_TYPE"
```

## Dashboard Features
- Dual-color system: app color (left border) + session color (second border)
- Multi-criteria filtering (app, session, event type)
- Live pulse chart with session-colored bars
- Chat transcript viewer with syntax highlighting
- Dark/light theme support
