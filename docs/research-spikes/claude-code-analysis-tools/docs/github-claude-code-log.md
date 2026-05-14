<!-- Source: https://github.com/daaain/claude-code-log -->
<!-- Retrieved: 2026-03-26 -->

# claude-code-log

Python CLI tool that converts Claude Code transcript JSONL files into readable HTML and Markdown formats.

## Core Features

- Interactive TUI for browsing sessions with summaries, timestamps, token counts
- Runtime message filtering via JavaScript in HTML output
- Interactive table of contents with session navigation
- Syntax-highlighted code blocks with Markdown rendering
- Date range filtering using natural language expressions
- Cross-session summary matching
- Responsive design for desktop and mobile

## Installation

```bash
pip install claude-code-log
uvx claude-code-log@latest
```

## Usage

```bash
claude-code-log --open-browser    # Process all projects
claude-code-log --tui             # Launch TUI
claude-code-log transcript.jsonl  # Single file
```
