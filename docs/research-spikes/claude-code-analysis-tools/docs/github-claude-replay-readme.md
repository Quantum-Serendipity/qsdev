<!-- Source: https://github.com/es617/claude-replay -->
<!-- Retrieved: 2026-03-26 -->

# Claude-Replay: Interactive AI Session Replays

## Overview

Claude-Replay converts session transcripts from Claude Code, Cursor, and Codex CLI into self-contained, interactive HTML replays. These files require no external dependencies and can be embedded anywhere—blogs, documentation, or shared via email.

## Core Features

- **Self-contained output**: Single HTML file with embedded data, no external requests
- **Interactive playback**: Speed control, step-through navigation, progress tracking
- **Collapsible blocks**: Hide/show thinking sections and tool calls during playback
- **Bookmarks/chapters**: Mark important moments for quick navigation
- **Secret redaction**: Automatic detection and masking of API keys and tokens
- **Multiple themes**: Built-in dark/light options plus custom theme support
- **Live monitoring**: Watch active sessions with `--serve --watch` flags
- **Web editor**: Visual interface for browsing, editing, and exporting sessions

## Installation

Install globally via npm:
```bash
npm install -g claude-replay
```

Or run directly without installation:
```bash
npx claude-replay
```

Docker support is also available for containerized deployments.

## Quick Start

Launch the browser-based editor with:
```bash
claude-replay
```

Generate a replay from the command line:
```bash
claude-replay session.jsonl -o replay.html
```

The tool auto-detects Claude Code, Cursor, and Codex CLI transcripts stored in standard locations (`~/.claude/projects/`, `~/.cursor/projects/`, `~/.codex/sessions/`).

## Key Usage Options

- **Filtering**: `--turns N-M` to include specific turn ranges, or `--from`/`--to` for timestamp-based filtering
- **Playback**: `--speed N` sets initial playback speed; `--timing` controls how timestamps are interpreted
- **Display**: `--no-thinking` and `--no-tool-calls` hide blocks by default
- **Redaction**: `--redact "text"` removes sensitive strings; automatic redaction is enabled by default
- **Theming**: `--theme dracula` selects built-in themes; `--theme-file` loads custom JSON

## Player Controls

The generated replay includes:
- Play/pause and step-through controls
- Speed adjustment (0.5x to 5x)
- Progress bar with time tracking
- Keyboard shortcuts (Space for play, arrow keys for navigation)
- Toggle checkboxes for thinking and tool blocks

## Technical Details

**Architecture**:
1. Parser reads JSONL transcripts, grouping turns as user message + assistant response
2. Renderer compresses data using deflate and base64, injecting it into an HTML template
3. Player decompresses using browser-native `DecompressionStream` API

**Optimization**:
- CSS/JavaScript minified with esbuild
- Data compressed ~60-70% by default (use `--no-compress` for older browsers)
- Zero external dependencies in output

## Use Cases

Organizations use claude-replay for:
- Blog posts demonstrating AI-assisted workflows
- Documentation with embedded debugging sessions
- Reproducible demos without video files
- Bug reports with interactive transcripts
- Educational walkthroughs of AI reasoning
- Real-time monitoring of remote agent sessions

## Embedding

Place replays in blog posts via iframe:
```html
<iframe src="replay.html" width="100%" height="600"></iframe>
```

The output is fully self-contained with no external asset requirements.
