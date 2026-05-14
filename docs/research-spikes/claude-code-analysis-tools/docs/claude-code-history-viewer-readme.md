<!-- Source: https://github.com/jhlee0409/claude-code-history-viewer -->
<!-- Retrieved: 2026-03-26 -->

# Claude Code History Viewer (CCHV)

A unified desktop application and headless server for browsing, searching, and analyzing conversation histories from multiple AI coding assistants -- Claude Code, Codex CLI, and OpenCode -- with 100% offline operation and zero telemetry.

## Core Features

**Multi-Provider Support**
Consolidates conversations from three different AI assistants into one interface with provider filtering.

**Conversation Management**
- Navigate sessions organized by project with worktree grouping
- Global search across all provider conversations
- Real-time file watching for instant updates
- Session context menus for copying IDs, resume commands, and file paths

**Analytics & Visualization**
- Dual-mode token statistics (billing versus conversation metrics)
- Cost breakdown charts and provider distribution analysis
- Session board with pixel view, attribute brushing, and activity timelines
- Analytics dashboard displaying comprehensive usage patterns

**Server Mode (v1.6.0)**
- Run as headless HTTP server accessible from any browser
- Real-time SSE updates when Claude Code sessions change
- Bearer token authentication (auto-generated or custom)
- Docker and systemd support for persistent deployment
- Single embedded binary containing complete frontend

**Additional Capabilities**
- Screenshot capture with range selection and multi-export
- Archive management (create, browse, rename, download)
- File modification history with restoration
- Auto-updater with skip/postpone options
- Multi-language support (English, Korean, Japanese, Chinese variants)

## Installation Methods

**macOS (Homebrew)**
```
brew install --cask jhlee0409/tap/claude-code-history-viewer
```

**Windows/Linux**
Download `.exe` or `.AppImage` from releases page.

**Server Binary**
```
brew install jhlee0409/tap/cchv-server
cchv-server --serve  # Starts at http://localhost:3727
```

## Technical Stack

| Component | Technology |
|-----------|-----------|
| Backend | Rust + Tauri v2 |
| Frontend | React 19, TypeScript |
| Styling | Tailwind CSS |
| State Management | Zustand |
| Build Tool | Vite |
| Internationalization | i18next (5 languages) |

## Data Source Locations

| Provider | Location | Content |
|----------|----------|---------|
| Claude Code | `~/.claude/projects/` | Full conversation history, tool use, thinking, costs |
| Codex CLI | `~/.codex/sessions/` | Session rollouts with agent responses |
| OpenCode | `~/.local/share/opencode/` | Conversation sessions and tool results |

## Data Privacy & Security

100% offline operation -- no data transmission to external servers, no analytics, no tracking, no telemetry.

**Repository Stats:** 727 stars, 732 commits, MIT License
