<!-- Source: https://github.com/swyxio/claude-compaction-viewer -->
<!-- Retrieved: 2026-03-26 -->

# Claude Code Compaction Viewer

**Repository:** swyxio/claude-compaction-viewer
**License:** MIT
**Language:** Python 100%
**Requirements:** Python 3.11+

## Overview

This is a terminal user interface (TUI) and command-line tool designed to inspect Claude Code conversation histories, specifically focusing on "compaction events" -- moments when long sessions trigger automatic context compression.

## What is Compaction?

When Claude Code sessions extend beyond context window limits, the system performs compaction rather than crashing. The process:

1. Inserts a boundary marker in the conversation
2. Generates a structured summary of prior history
3. Continues with the summary replacing full history in active context

Complete conversations remain preserved as JSONL files at `~/.claude/projects/`. The tool reveals where compaction occurred, what summaries contain, pre-compaction token counts, and whether events were automatic or user-triggered via `/compact` command.

## Installation

**Using uv (recommended):**
```
uv tool install claude-compaction-viewer
```

**Using pip:**
```
pip install claude-compaction-viewer
```

**Without installing (via uvx):**
```
uvx claude-compaction-viewer --scan
```

## Core Features & Usage

### 1. Scan All Conversations
```
ccv --scan
```

Generates a comprehensive table across all projects showing project paths, session identifiers, line counts per conversation, number of compaction events, token consumption metrics, and session duration.

### 2. View Compaction Summaries
```
ccv --summary ~/.claude/projects/<project>/<session>.jsonl
```

Displays full details for each compaction event including trigger type, token counts, and complete summary text visible to Claude post-compaction.

### 3. Interactive Terminal UI
```
ccv
```

Launches a comprehensive interface featuring:
- **Left sidebar:** Hierarchical tree of projects and conversation files
- **Stats bar:** Message counts, token usage, model information, duration
- **Compaction bar:** Highlighted summary of all compaction events
- **Message table:** Scrollable list of all message types
- **Detail panel:** Full content and metadata for selected messages

#### Keyboard Controls

| Key | Action |
|-----|--------|
| `c` | Navigate to next compaction boundary |
| `Shift+C` | Navigate to previous compaction boundary |
| `s` | Display all summaries in detail panel |
| `t` | Toggle progress message visibility |
| `j` / `k` | Scroll down/up |
| `q` | Exit application |

## Data Storage Structure

Claude Code maintains JSONL conversation files at:
```
~/.claude/projects/<project-path>/<session-uuid>.jsonl
```

Each line represents a JSON object with a `type` field:

| Type | Purpose |
|------|---------|
| `user` | User messages and tool execution results |
| `assistant` | Claude responses, reasoning, and tool invocations |
| `system` | System messages, including compaction boundaries |
| `progress` | Status updates during extended operations |
| `file-history-snapshot` | File state records for undo functionality |

Compaction events create two sequential lines:
1. System message with `subtype: "compact_boundary"` containing metadata about the compaction trigger and pre-compaction token count
2. User message with `isCompactSummary: true` holding the structured summary
