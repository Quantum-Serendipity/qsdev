<!-- Source: https://github.com/es617/claude-replay -->
<!-- Retrieved: 2026-03-26 -->

# claude-replay: Interactive AI Session Replays

## Overview

**claude-replay** is a community tool that converts AI coding agent session transcripts into self-contained, interactive HTML replays. It supports Claude Code, Cursor, and Codex CLI formats without requiring external dependencies.

## Key Features

The tool generates single-file HTML outputs with:
- Interactive playback controls and speed adjustment (0.5x to 5x)
- Collapsible thinking blocks and tool call sections
- Bookmarks/chapters for navigation
- Automatic secret redaction before export
- Multiple built-in color themes
- Terminal-style bottom-to-top scrolling
- Keyboard shortcuts for navigation
- Live watch mode for real-time session monitoring

## Installation

Install globally via npm:
```bash
npm install -g claude-replay
```

Or run directly without installation:
```bash
npx claude-replay
```

Docker support is available:
```bash
docker run --rm -p 7331:7331 \
  -v ~/.claude/projects:/root/.claude/projects:ro \
  ghcr.io/es617/claude-replay
```

## Quick Start

Launch the web editor (auto-discovers sessions):
```bash
claude-replay
```

Generate a replay from command line:
```bash
claude-replay session-id -o replay.html
claude-replay ~/.claude/projects/path/session.jsonl -o replay.html
```

Chain multiple sessions:
```bash
claude-replay session1-id session2-id -o combined.html
```

## Web Editor

Running `claude-replay` without arguments opens a browser-based editor offering:
- Auto-discovery of sessions from standard directories
- Turn-by-turn editing capabilities
- Live preview rendering
- Visual redaction rule configuration
- Direct HTML export

The editor runs locally on 127.0.0.1 and never modifies original files.

## Command-Line Options

**Output & Filtering:**
- `-o, --output FILE` -- save HTML to file
- `--turns N-M` -- include specific turn range
- `--exclude-turns N,N,...` -- skip numbered turns
- `--from / --to TIMESTAMP` -- filter by time (ISO 8601)

**Playback:**
- `--speed N` -- initial playback speed
- `--timing MODE` -- `auto`, `real`, or `paced`

**Display:**
- `--theme NAME` -- select built-in theme
- `--theme-file FILE` -- custom theme JSON
- `--no-thinking` -- hide thinking blocks
- `--no-tool-calls` -- hide tool calls

**Metadata:**
- `--title TEXT` -- page title
- `--description TEXT` -- meta description
- `--og-image URL` -- link preview image

**Advanced:**
- `--redact "text"` -- replace occurrences
- `--mark "N:Label"` -- add bookmarks
- `--serve --watch` -- live preview mode
- `--no-minify` -- unminified output
- `--no-compress` -- disable data compression
- `--open` -- launch in browser

## Timing Modes

| Mode | Behavior |
|------|----------|
| `auto` | Uses real timestamps if available, falls back to paced |
| `real` | Preserves original transcript timestamps |
| `paced` | Generates synthetic timing based on content length |

## Player Controls

**Keyboard Shortcuts:**
- `Space`/`K` -- play/pause
- Right/`L` -- step forward one block
- Left/`H` -- step back one block
- `Shift+Right`/`L` -- jump to next turn
- `Shift+Left`/`H` -- jump to previous turn
- `T` -- jump to next thinking/tool block
- `Shift+T` -- jump to previous thinking/tool block

## Themes

Built-in themes: `tokyo-night` (default), `monokai`, `solarized-dark`, `github-light`, `dracula`, `bubbles`

## Supported Formats

| Source | Location |
|--------|----------|
| Claude Code | `~/.claude/projects/<project>/` |
| Cursor | `~/.cursor/projects/<project>/agent-transcripts/<id>/` |
| Codex CLI | `~/.codex/sessions/<date>/` |

## How It Works

1. **Parser** reads JSONL transcripts line-by-line, handling Claude Code's streaming format
2. **Grouping** organizes user messages, assistant responses, tool calls, and results into logical turns
3. **Rendering** compresses parsed data (deflate + base64) and injects into HTML template
4. **Player** uses vanilla JavaScript with browser-native `DecompressionStream` API -- no external requests or frameworks

Output files reduce size by ~60-70% through compression.

## Repository Stats

- **License:** MIT
- **Node.js:** 18+
- **Dependencies:** 0 (zero external dependencies)
- **Stars:** 573 | **Forks:** 33
