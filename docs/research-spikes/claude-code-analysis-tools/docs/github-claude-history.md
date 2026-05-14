<!-- Source: https://github.com/raine/claude-history -->
<!-- Retrieved: 2026-03-26 -->

# claude-history - Fuzzy-search Claude Code conversation history

Terminal UI tool for searching and viewing Claude Code conversation transcripts. Written in Rust.

## Key Features

- Fuzzy search across all conversation history (case-insensitive, multi-word)
- Built-in viewer with vim-style navigation
- Tool output indexing (search inside bash outputs, file contents, grep results)
- UUID lookup to jump directly to a session
- Resume conversations (Ctrl+R) and fork sessions (Ctrl+F)
- Thinking blocks support
- Markdown rendering in terminal
- Multi-project browsing
- Light/dark theme detection

## Installation

```bash
brew install raine/claude-history/claude-history
cargo install claude-history
```

## Usage

```bash
claude-history          # opens searchable list sorted by recency
```

## Key Controls

- Arrow keys / j/k: navigate
- Enter: open conversation
- Tab: toggle all/workspace scope
- Ctrl+R: resume, Ctrl+F: fork and resume
- t: cycle tool display, T: toggle thinking blocks
- e: export to file

## Configuration

~/.config/claude-history/config.toml
