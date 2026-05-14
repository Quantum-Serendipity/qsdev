<!-- Source: https://github.com/matt1398/claude-devtools -->
<!-- Retrieved: 2026-03-26 -->

# Claude DevTools

Desktop application that reconstructs and visualizes Claude Code sessions by parsing raw logs from ~/.claude/. "Terminal tells you nothing. This shows you everything."

## Key Features

**Context Reconstruction**: Per-turn token attribution across seven categories: CLAUDE.md, skill activations, @-mentions, tool I/O, extended thinking, team overhead, user prompts.

**Compaction Visualization**: Detects when Claude Code silently compresses conversations, showing token deltas before/after each compaction event.

**Notification Triggers**: Custom regex-based alerts for sensitive file access, execution errors, high token consumption.

**Tool Call Inspector**: Syntax-highlighted reads, inline diffs for edits, bash output, expandable subagent trees.

**Team & Subagent Visualization**: Isolates subagent sessions as expandable cards with independent metrics, color-codes teammate messages.

**SSH Remote Sessions**: Inspect Claude Code sessions on remote machines via SFTP log streaming.

**Multi-Pane Layout**: Multiple sessions side-by-side with drag-and-drop tabs and split views.

## Installation

```bash
brew install --cask claude-devtools    # macOS
```

Also: .AppImage/.deb/.rpm for Linux, .exe for Windows, Docker support.

## Why It Exists

Claude Code replaced detailed output with opaque counters ("Read 3 files") and a three-segment context bar lacking breakdown. --verbose dumps raw JSON without structure.

## License
MIT
