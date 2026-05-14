<!-- Source: https://github.com/KyleAMathews/claude-code-ui -->
<!-- Retrieved: 2026-03-26 -->

# Claude Code Session Tracker UI

## Overview

Real-time dashboard for monitoring Claude Code sessions across multiple projects. Track session status, view AI-powered summaries, and monitor associated pull requests and CI status.

## Core Features

- **Real-time synchronization** using Durable Streams technology
- **Kanban-style board** organizing sessions into status columns (Working, Needs Approval, Waiting, Idle)
- **AI-generated summaries** leveraging Claude Sonnet for activity analysis
- **PR and CI monitoring** with branch detection
- **Multi-repository organization** grouping sessions by GitHub projects

## System Architecture

**Daemon Service** (`packages/daemon`): Monitors `~/.claude/projects/` directory, parses JSONL logs incrementally, determines status via XState state machine, calls Claude Sonnet API for summaries, polls git branch/PR/CI, publishes state via Durable Streams.

**React Interface** (`packages/ui`): Subscribes to Durable Streams, groups sessions by repository, displays session cards with goals and summaries, provides hover-based output preview.

## Session Status State Machine

Four operational states:
- `idle`: Inactivity lasting 5+ minutes
- `working`: Active Claude processing
- `waiting_for_approval`: Pending user authorization for tools
- `waiting_for_input`: Awaiting user response after Claude completion

## Installation

```
pnpm install
pnpm run setup  # Optional: installs PermissionRequest hook
pnpm start      # Launch daemon + UI
```

Requires `ANTHROPIC_API_KEY` environment variable for AI summaries.

## Technical Stack

@durable-streams for synchronization, @tanstack/db for reactive UI state, xstate for state machine logic, chokidar for file watching, @radix-ui/themes for UI components.
