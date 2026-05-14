<!-- Source: https://github.com/simonw/claude-code-transcripts -->
<!-- Retrieved: 2026-03-26 -->

# claude-code-transcripts (by Simon Willison)

A Python tool for converting Claude Code session files into clean, paginated HTML transcripts.

## Installation

```bash
uv tool install claude-code-transcripts
```

## Core Features

Four primary commands:
- **local** (default): Browse and convert sessions from `~/.claude/projects`
- **web**: Import sessions directly from the Claude API
- **json**: Convert specific JSON/JSONL session files
- **all**: Batch convert your entire local session archive

## Key Capabilities

- Produces index page with timeline plus numbered transcript pages (page-001.html, page-002.html, etc.)
- Automatic GitHub Gist uploads for sharing
- Inclusion of original session JSON data
- Browser auto-launch
- Interactive picker showing sessions grouped by associated GitHub repository

## Usage Examples

```bash
claude-code-transcripts           # Quick start with recent local session
claude-code-transcripts all       # Convert all sessions to browsable archive
claude-code-transcripts --gist    # Export to GitHub Gist
```

**Note:** Web-based commands currently have issues due to API changes. Local and JSON conversion features remain functional.
