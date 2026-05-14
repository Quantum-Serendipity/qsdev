<!-- Source: https://github.com/jazzyalex/agent-sessions -->
<!-- Retrieved: 2026-03-26 -->

# Agent Sessions macOS App

## What It Is

Agent Sessions is a native macOS application (requires macOS 14+) that serves as a unified session browser and management tool for multiple AI coding agents.

## Supported Agents

- Claude Code
- Codex CLI
- Gemini CLI
- GitHub Copilot CLI
- OpenCode
- Factory CLI (Droid)

## Core Capabilities

**Session Management:**
- Search across large session histories with unified indexing
- Browse transcripts with image support
- Archive and filter sessions
- Copy exact CLI resume commands via right-click context menu

**Agent Cockpit (Beta):**
A live HUD displaying real-time status for active iTerm2 sessions, including:
- Active/waiting session summaries
- Live Claude usage tracking
- Quick navigation to individual sessions

**Search & Navigation:**
- Cross-session unified search functionality
- In-session "Find" feature for transcript navigation
- "Readable tool calls/outputs" with structured navigation between prompts, tools, and errors

## Installation

**DMG Download:** Version 3.3.2 from GitHub releases, drag to Applications.

**Homebrew:**
```
brew tap jazzyalex/agent-sessions
brew install --cask agent-sessions
```

Automatic updates use Sparkle (signed + notarized).

## Privacy & Security

- Local-only operation with no telemetry
- Read-only access to session directories
- No data transmission beyond the local machine

## Development Stack

**Language:** Primarily Swift (88.1%), with Shell (5.7%) and Python (5.5%)

**License:** MIT

**Repository Stats:** 406 stars, 1,032 commits, 4 contributors
