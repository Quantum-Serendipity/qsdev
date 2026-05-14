<!-- Source: https://github.com/raine/claude-history -->
<!-- Retrieved: 2026-03-26 -->

# claude-history: Claude Code Conversation Search Tool

## Overview

claude-history is a terminal UI companion for Claude Code that enables fuzzy searching of recent conversations stored in Claude's local project history. Users can browse transcripts, view full conversations with scrolling capabilities, and resume sessions directly from the terminal.

## Installation

**Quick install:**
```
curl -fsSL https://raw.githubusercontent.com/raine/claude-history/main/scripts/install.sh | bash
```

**Homebrew:**
```
brew install raine/claude-history/claude-history
```

**Cargo:**
```
cargo install claude-history
```

## Key Features

**Search Capabilities:** Fuzzy word matching that is case-insensitive, treats underscores as separators, supports prefix matching at word boundaries, and employs multi-word AND logic. Indexes content within tool output (bash results, file contents, grep results) alongside user/assistant text.

**Conversation Management:** Displays conversations sorted by recency across all projects. Toggle between global view and current workspace filtering. Tab key switches between scopes, `-L/--local` flag starts in workspace-only mode.

**Resume Functionality:** `--resume` hands off to `claude --resume <id>`, enabling conversation continuation. `--fork-session` creates new branches from existing sessions.

**Display Options:** Tools default to truncated mode (header plus first lines). `t` key cycles through off/truncated/full modes. Extended thinking models show reasoning steps, togglable with `T` or `--show-thinking`. Markdown rendering can be disabled with `--plain` for raw output.

## Usage Modes

**Interactive TUI:** Running `claude-history` launches a searchable list interface with vim-style navigation (j/k for movement). Enter opens conversation viewer with full transcript display.

**Direct File Input:** Bypass list interface by specifying a JSONL file directly: `claude-history /path/to/conversation.jsonl`

**Plain Output:** `--plain` flag produces simple role/content output suitable for piping to other tools or LLMs.

## Keyboard Navigation

**List Mode:** Arrow keys move selection; Ctrl+W deletes words; Ctrl+R/F/X control resume/fork/delete operations. Tab toggles scope filtering.

**Viewer Mode:** j/k scroll, J/K jump between messages, `/` searches within conversation, `n/N` navigate matches. `t` cycles tool display modes, `T` toggles thinking blocks.

## Configuration

Settings in `~/.config/claude-history/config.toml` with sections for display, resume, and keys.

## Technical Details

Written in Rust. Automatically discovers project history folders when run from a project directory.
