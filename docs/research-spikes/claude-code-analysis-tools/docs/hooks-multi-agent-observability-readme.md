<!-- Source: https://github.com/disler/claude-code-hooks-multi-agent-observability -->
<!-- Retrieved: 2026-03-26 -->

# Multi-Agent Observability System for Claude Code

## Overview

This system provides real-time monitoring and visualization for Claude Code agents through comprehensive hook event tracking. It captures all 12 Claude Code lifecycle events, stores them in SQLite, and broadcasts updates via WebSocket to a Vue 3 dashboard.

**Core Stack:** Claude Agents -> Python Hook Scripts -> Bun Server -> SQLite -> WebSocket -> Vue Client

## Key Features

- **12 Event Types Tracked:** PreToolUse, PostToolUse, PostToolUseFailure, PermissionRequest, Notification, Stop, SubagentStart, SubagentStop, PreCompact, UserPromptSubmit, SessionStart, SessionEnd
- **Multi-Agent Support:** Monitor multiple concurrent agents with session-based color coding
- **Real-Time Dashboard:** Live event timeline with filtering, chat transcripts, and activity charts
- **Event Visualization:** Tool-specific emojis, session colors, and combo indicators
- **MCP Tool Detection:** Identifies Model Context Protocol tool usage automatically

## Installation

### Requirements

- Claude Code CLI
- Astral uv (Python package manager)
- Bun, npm, or yarn
- just (optional task runner)

### Quick Start

```bash
# 1. Start server and client
just start  # or ./scripts/start-system.sh

# 2. Open http://localhost:5173

# 3. Run Claude Code commands to generate events
```

## Project Structure

```
apps/
  server/          # Bun TypeScript (HTTP/WebSocket endpoints)
  client/          # Vue 3 dashboard (filtering, charts, transcripts)

.claude/
  hooks/           # 12 event handler scripts
  agents/team/     # Builder & Validator agent definitions
  commands/        # /plan_w_team slash command
  settings.json    # Hook configuration
```

## Agent Teams

- **Builder:** Engineering agent with linting hooks (ruff, type checking)
- **Validator:** Read-only inspection agent (no file modifications)

Use `/plan_w_team "feature description"` to generate implementation plans.

## Configuration

- Server port: 4000
- Client port: 5173
- Events stored in SQLite
